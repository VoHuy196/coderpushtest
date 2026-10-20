package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"supplier-catalog/internal/catalog"
)

const runColumns = `
	id, supplier_update_id, status, next_batch, total_batches,
	created_count, updated_count, withdrawn_count, skipped_count, invalid_count,
	last_error, started_at, finished_at,
	(SELECT COUNT(*) FROM problems p WHERE p.run_id = r.id AND p.status = 'open')`

func scanRun(row interface{ Scan(...any) error }) (catalog.Run, error) {
	var r catalog.Run
	var started, finished string
	err := row.Scan(&r.ID, &r.SupplierUpdateID, &r.Status, &r.NextBatch, &r.TotalBatches,
		&r.Counts.Created, &r.Counts.Updated, &r.Counts.Withdrawn, &r.Counts.Skipped, &r.Counts.Invalid,
		&r.LastError, &started, &finished, &r.OpenProblems)
	if err != nil {
		return r, err
	}
	r.StartedAt = parseTime(started)
	r.FinishedAt = parseOptionalTime(finished)
	return r, nil
}

// CreateRun starts a new run for a supplier update. Only one unfinished run
// (running or interrupted) may exist at a time.
func (s *Store) CreateRun(ctx context.Context, updateID string, now time.Time) (catalog.Run, error) {
	var id int64
	err := s.InTx(ctx, func(tx *Tx) error {
		var open int
		err := tx.q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM update_runs WHERE status IN (?, ?)`,
			catalog.RunRunning, catalog.RunInterrupted).Scan(&open)
		if err != nil {
			return err
		}
		if open > 0 {
			return ErrOpenRun
		}
		res, err := tx.q.ExecContext(ctx,
			`INSERT INTO update_runs (supplier_update_id, status, started_at) VALUES (?, ?, ?)`,
			updateID, catalog.RunRunning, formatTime(now))
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	if err != nil {
		return catalog.Run{}, err
	}
	return s.GetRun(ctx, id)
}

// GetRun returns one run.
func (s *Store) GetRun(ctx context.Context, id int64) (catalog.Run, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM update_runs r WHERE id = ?`, id)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// ListRuns returns the newest runs first.
func (s *Store) ListRuns(ctx context.Context, limit int) ([]catalog.Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+runColumns+` FROM update_runs r ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := []catalog.Run{}
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// ChangeRunStatus moves a run from one status to another. It fails with
// ErrWrongStatus if the run is not in the "from" status, which protects us
// from two workers picking up the same run.
func (s *Store) ChangeRunStatus(ctx context.Context, id int64, from, to, lastError string, now time.Time) error {
	finished := ""
	if to == catalog.RunCompleted || to == catalog.RunAbandoned {
		finished = formatTime(now)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE update_runs SET status = ?, last_error = ?, finished_at = ? WHERE id = ? AND status = ?`,
		to, lastError, finished, id, from)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrWrongStatus
	}
	return nil
}

// RunningRunIDs returns runs that are marked as running. Right after start-up
// these can only be runs that were cut off when the application stopped.
func (s *Store) RunningRunIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM update_runs WHERE status = ?`, catalog.RunRunning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FinishBatch records that a batch was fully applied: it moves the run's
// checkpoint forward and adds the batch counts. It must be called inside the
// same transaction as the product changes, so the checkpoint and the data
// can never disagree.
//
// The "next_batch = ?" condition makes sure the same batch is never counted twice.
func (tx *Tx) FinishBatch(ctx context.Context, runID int64, batch, totalBatches int, c catalog.Counts, last bool, now time.Time) error {
	status, finished := catalog.RunRunning, ""
	if last {
		status, finished = catalog.RunCompleted, formatTime(now)
	}
	res, err := tx.q.ExecContext(ctx, `
		UPDATE update_runs SET
			next_batch = next_batch + 1,
			total_batches = ?,
			created_count = created_count + ?,
			updated_count = updated_count + ?,
			withdrawn_count = withdrawn_count + ?,
			skipped_count = skipped_count + ?,
			invalid_count = invalid_count + ?,
			status = ?,
			last_error = '',
			finished_at = ?
		WHERE id = ? AND next_batch = ? AND status = ?`,
		totalBatches, c.Created, c.Updated, c.Withdrawn, c.Skipped, c.Invalid,
		status, finished, runID, batch, catalog.RunRunning)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrWrongStatus
	}
	return nil
}
