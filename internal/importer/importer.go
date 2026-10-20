// Package importer runs supplier updates: it fetches batches, applies them to
// the catalog and keeps a checkpoint so an update can be resumed.
package importer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"supplier-catalog/internal/catalog"
	"supplier-catalog/internal/store"
	"supplier-catalog/internal/supplier"
)

// ErrBusy is returned when an update is already being processed.
var ErrBusy = errors.New("an update is already running")

// Supplier is what the importer needs from the supplier.
// The real implementation is supplier.Client; tests can use a fake server.
type Supplier interface {
	FetchBatch(ctx context.Context, updateID string, number int) (supplier.Batch, error)
}

// Importer processes one update at a time in a background goroutine.
type Importer struct {
	store    *store.Store
	supplier Supplier
	log      *slog.Logger

	// baseCtx is cancelled when the application shuts down.
	baseCtx context.Context

	// Retries is how many times a batch is requested before giving up.
	Retries int
	// RetryDelay is the wait before the first retry; it grows with each attempt.
	RetryDelay time.Duration
	// Now returns the current time (replaceable in tests).
	Now func() time.Time
	// AfterBatch, if set, is called after each batch is committed. Tests use
	// it to simulate a crash or an outage in the middle of an update.
	AfterBatch func(runID int64, batch int)

	mu     sync.Mutex
	active bool
	wg     sync.WaitGroup
}

// New creates an importer. baseCtx should be cancelled on shutdown.
func New(baseCtx context.Context, st *store.Store, sup Supplier, log *slog.Logger) *Importer {
	return &Importer{
		store:      st,
		supplier:   sup,
		log:        log,
		baseCtx:    baseCtx,
		Retries:    3,
		RetryDelay: time.Second,
		Now:        time.Now,
	}
}

// Recover must be called once at start-up. A run that is still "running" in
// the database was cut off when the application stopped (crash, kill, power
// loss). We mark it as interrupted so staff can see it and resume it.
func (im *Importer) Recover(ctx context.Context) error {
	ids, err := im.store.RunningRunIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		msg := "the application stopped before this update finished"
		if err := im.store.ChangeRunStatus(ctx, id, catalog.RunRunning, catalog.RunInterrupted, msg, im.Now()); err != nil {
			return err
		}
		err := im.store.AddProblem(ctx, catalog.Problem{
			RunID:     id,
			Kind:      catalog.ProblemUpdateInterrupted,
			Message:   "Update was interrupted because the application stopped. Resume it to continue from the last saved batch.",
			CreatedAt: im.Now(),
		})
		if err != nil {
			return err
		}
		im.log.Warn("found interrupted update", "run", id)
	}
	return nil
}

// Start creates a run for a supplier update and processes it in the background.
func (im *Importer) Start(ctx context.Context, updateID string) (catalog.Run, error) {
	updateID = strings.TrimSpace(updateID)
	if updateID == "" {
		return catalog.Run{}, errors.New("update id is required")
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	if im.active {
		return catalog.Run{}, ErrBusy
	}

	run, err := im.store.CreateRun(ctx, updateID, im.Now())
	if err != nil {
		return catalog.Run{}, err
	}
	im.launch(run.ID)
	return run, nil
}

// Resume continues an interrupted run from its last saved batch.
func (im *Importer) Resume(ctx context.Context, runID int64) (catalog.Run, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.active {
		return catalog.Run{}, ErrBusy
	}

	err := im.store.ChangeRunStatus(ctx, runID, catalog.RunInterrupted, catalog.RunRunning, "", im.Now())
	if errors.Is(err, store.ErrWrongStatus) {
		return catalog.Run{}, errors.New("only interrupted updates can be resumed")
	}
	if err != nil {
		return catalog.Run{}, err
	}
	im.launch(runID)
	return im.store.GetRun(ctx, runID)
}

// Abandon gives up on an interrupted run. Batches already saved stay saved
// (they are valid data), but the run will never be reported as completed.
func (im *Importer) Abandon(ctx context.Context, runID int64) (catalog.Run, error) {
	err := im.store.ChangeRunStatus(ctx, runID, catalog.RunInterrupted, catalog.RunAbandoned, "abandoned by staff", im.Now())
	if errors.Is(err, store.ErrWrongStatus) {
		return catalog.Run{}, errors.New("only interrupted updates can be abandoned")
	}
	if err != nil {
		return catalog.Run{}, err
	}
	if err := im.store.ResolveRunProblems(ctx, runID, "update abandoned by staff", im.Now()); err != nil {
		return catalog.Run{}, err
	}
	return im.store.GetRun(ctx, runID)
}

// Wait blocks until the background worker has stopped.
func (im *Importer) Wait() { im.wg.Wait() }

// launch starts the background worker. im.mu must be held.
func (im *Importer) launch(runID int64) {
	im.active = true
	im.wg.Add(1)
	go func() {
		defer im.wg.Done()
		if err := im.process(im.baseCtx, runID); err != nil {
			im.log.Error("update stopped", "run", runID, "err", err)
		}
		im.mu.Lock()
		im.active = false
		im.mu.Unlock()
	}()
}

// process imports batches one by one until the last batch is saved or
// something goes wrong.
func (im *Importer) process(ctx context.Context, runID int64) error {
	run, err := im.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	im.log.Info("processing update", "run", runID, "update", run.SupplierUpdateID, "from_batch", run.NextBatch)

	for batchNo := run.NextBatch; ; batchNo++ {
		batch, err := im.fetchWithRetry(ctx, run.SupplierUpdateID, batchNo)
		if err != nil {
			return im.interrupt(ctx, runID, batchNo, err)
		}

		if err := im.applyBatch(ctx, runID, batch); err != nil {
			return im.interrupt(ctx, runID, batchNo, err)
		}
		im.log.Info("batch saved", "run", runID, "batch", batchNo, "last", batch.Last)

		if im.AfterBatch != nil {
			im.AfterBatch(runID, batchNo)
		}
		if batch.Last {
			return nil
		}
	}
}

// fetchWithRetry asks the supplier for a batch, retrying temporary errors.
func (im *Importer) fetchWithRetry(ctx context.Context, updateID string, batchNo int) (supplier.Batch, error) {
	var lastErr error
	for attempt := 1; attempt <= im.Retries; attempt++ {
		batch, err := im.supplier.FetchBatch(ctx, updateID, batchNo)
		if err == nil {
			err = checkBatch(batch, updateID, batchNo)
		}
		if err == nil {
			return batch, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return supplier.Batch{}, ctx.Err()
		}
		if !supplier.Retryable(err) || attempt == im.Retries {
			break
		}
		im.log.Warn("fetch failed, retrying", "batch", batchNo, "attempt", attempt, "err", err)
		select {
		case <-time.After(im.RetryDelay * time.Duration(attempt)):
		case <-ctx.Done():
			return supplier.Batch{}, ctx.Err()
		}
	}
	return supplier.Batch{}, lastErr
}

// checkBatch makes sure the supplier sent the batch we asked for.
func checkBatch(b supplier.Batch, updateID string, batchNo int) error {
	if b.UpdateID != updateID || b.Number != batchNo {
		return fmt.Errorf("asked for batch %d of %s but got batch %d of %s", batchNo, updateID, b.Number, b.UpdateID)
	}
	return nil
}

// applyBatch saves all records of a batch and the new checkpoint in ONE
// transaction. If the application dies halfway, nothing of this batch is
// saved and it will simply be fetched again on resume.
func (im *Importer) applyBatch(ctx context.Context, runID int64, batch supplier.Batch) error {
	now := im.Now()
	return im.store.InTx(ctx, func(tx *store.Tx) error {
		var counts catalog.Counts
		for _, rec := range batch.Products {
			action, err := im.applyRecord(ctx, tx, runID, batch.Number, rec, now)
			if err != nil {
				return err
			}
			counts.Add(action)
		}

		if err := tx.FinishBatch(ctx, runID, batch.Number, batch.TotalBatches, counts, batch.Last, now); err != nil {
			return err
		}
		if batch.Last {
			return tx.ResolveRunProblems(ctx, runID, "update completed", now)
		}
		return nil
	})
}

// applyRecord applies one supplier record and writes it to the change log.
func (im *Importer) applyRecord(ctx context.Context, tx *store.Tx, runID int64, batchNo int, rec supplier.Product, now time.Time) (catalog.Action, error) {
	rec.SKU = strings.TrimSpace(rec.SKU)
	change := catalog.Change{RunID: runID, Batch: batchNo, SKU: rec.SKU, CreatedAt: now}

	updatedAt, err := validate(rec, now)
	if err != nil {
		change.Action = catalog.ActionInvalid
		change.Detail = err.Error()
		if err := tx.AddChange(ctx, change); err != nil {
			return "", err
		}
		problem := catalog.Problem{
			RunID:     runID,
			Batch:     batchNo,
			SKU:       rec.SKU,
			Kind:      catalog.ProblemInvalidProduct,
			Message:   fmt.Sprintf("Record rejected: %v. The stored product (if any) was left unchanged.", err),
			CreatedAt: now,
		}
		return catalog.ActionInvalid, tx.AddProblem(ctx, problem)
	}

	existing, err := tx.GetProduct(ctx, rec.SKU)
	if err != nil {
		return "", err
	}

	action, reason := catalog.Decide(existing, updatedAt, rec.Withdrawn)
	change.Action = action
	change.Detail = reason

	if action != catalog.ActionSkip {
		next := buildProduct(existing, rec, updatedAt, runID, now)
		if existing != nil {
			change.Detail = joinDetail(reason, catalog.Diff(*existing, next))
		}
		if err := tx.SaveProduct(ctx, next); err != nil {
			return "", err
		}
		resolution := fmt.Sprintf("newer valid data received in run %d", runID)
		if err := tx.ResolveProductProblems(ctx, rec.SKU, resolution, now); err != nil {
			return "", err
		}
	}

	return action, tx.AddChange(ctx, change)
}

// buildProduct creates the new stored version of a product.
// A withdrawal keeps the last known details so staff can still see what it was.
func buildProduct(existing *catalog.Product, rec supplier.Product, updatedAt time.Time, runID int64, now time.Time) catalog.Product {
	p := catalog.Product{
		SKU:               rec.SKU,
		SupplierUpdatedAt: updatedAt,
		LastRunID:         runID,
		LocalUpdatedAt:    now,
	}
	if rec.Withdrawn {
		if existing != nil {
			p.Name, p.Description = existing.Name, existing.Description
			p.PriceCents, p.Currency = existing.PriceCents, existing.Currency
		}
		p.Status = catalog.StatusWithdrawn
		return p
	}
	p.Name = strings.TrimSpace(rec.Name)
	p.Description = rec.Description
	p.PriceCents = rec.PriceCents
	p.Currency = strings.ToUpper(rec.Currency)
	p.Status = catalog.StatusActive
	return p
}

func joinDetail(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}

// interrupt marks the run as interrupted and records why. It uses a fresh
// context because ctx may already be cancelled (shutdown).
func (im *Importer) interrupt(ctx context.Context, runID int64, batchNo int, cause error) error {
	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	kind := catalog.ProblemSupplierUnavailable
	msg := fmt.Sprintf("Could not import batch %d: %v. Already saved batches are kept; resume when the supplier is available.", batchNo, cause)
	if ctx.Err() != nil {
		kind = catalog.ProblemUpdateInterrupted
		msg = fmt.Sprintf("The application stopped while importing batch %d. Resume the update to continue.", batchNo)
	}

	if err := im.store.ChangeRunStatus(bg, runID, catalog.RunRunning, catalog.RunInterrupted, cause.Error(), im.Now()); err != nil {
		return fmt.Errorf("%v (and could not mark run interrupted: %w)", cause, err)
	}
	if err := im.store.AddProblem(bg, catalog.Problem{RunID: runID, Batch: batchNo, Kind: kind, Message: msg, CreatedAt: im.Now()}); err != nil {
		return err
	}
	return cause
}
