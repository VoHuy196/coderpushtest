package importer

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"supplier-catalog/internal/catalog"
	"supplier-catalog/internal/fakesupplier"
	"supplier-catalog/internal/store"
	"supplier-catalog/internal/supplier"
)

// env is a test environment: a fake supplier over HTTP and a SQLite file.
type env struct {
	t      *testing.T
	fake   *fakesupplier.Server
	url    string
	dbPath string
	store  *store.Store
}

func newEnv(t *testing.T) *env {
	t.Helper()
	fake := fakesupplier.New(fakesupplier.DemoUpdates())
	srv := httptest.NewServer(fake.Handler())
	t.Cleanup(srv.Close)

	e := &env{t: t, fake: fake, url: srv.URL, dbPath: filepath.Join(t.TempDir(), "test.db")}
	e.openStore()
	t.Cleanup(func() { e.store.Close() })
	return e
}

// openStore (re)opens the database, like the application does on start-up.
func (e *env) openStore() {
	st, err := store.Open(e.dbPath)
	if err != nil {
		e.t.Fatal(err)
	}
	e.store = st
}

func (e *env) newImporter(ctx context.Context) *Importer {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	im := New(ctx, e.store, supplier.NewClient(e.url), log)
	im.Retries = 2
	im.RetryDelay = time.Millisecond
	return im
}

// importUpdate runs one update to the end and returns the final run.
func (e *env) importUpdate(im *Importer, updateID string) catalog.Run {
	e.t.Helper()
	run, err := im.Start(context.Background(), updateID)
	if err != nil {
		e.t.Fatalf("start %s: %v", updateID, err)
	}
	im.Wait()
	return e.run(run.ID)
}

func (e *env) run(id int64) catalog.Run {
	e.t.Helper()
	run, err := e.store.GetRun(context.Background(), id)
	if err != nil {
		e.t.Fatal(err)
	}
	return run
}

func (e *env) products() map[string]catalog.Product {
	e.t.Helper()
	list, err := e.store.ListProducts(context.Background(), "")
	if err != nil {
		e.t.Fatal(err)
	}
	m := map[string]catalog.Product{}
	for _, p := range list {
		m[p.SKU] = p
	}
	return m
}

func (e *env) openProblems() []catalog.Problem {
	e.t.Helper()
	list, err := e.store.ListProblems(context.Background(), "open")
	if err != nil {
		e.t.Fatal(err)
	}
	return list
}

func TestImportCompletes(t *testing.T) {
	e := newEnv(t)
	im := e.newImporter(context.Background())

	run := e.importUpdate(im, "NW-2025-09-01")

	if run.Status != catalog.RunCompleted {
		t.Fatalf("status = %s, want completed (error: %s)", run.Status, run.LastError)
	}
	if run.BatchesDone() != 3 || run.TotalBatches != 3 {
		t.Errorf("batches done = %d / %d, want 3 / 3", run.BatchesDone(), run.TotalBatches)
	}
	want := catalog.Counts{Created: 11, Skipped: 1, Invalid: 1}
	if run.Counts != want {
		t.Errorf("counts = %+v, want %+v", run.Counts, want)
	}
	if got := len(e.products()); got != 11 {
		t.Errorf("products = %d, want 11", got)
	}
	// The record without a name must be reported.
	problems := e.openProblems()
	if len(problems) != 1 || problems[0].SKU != "NW-CHAIR-LT" {
		t.Errorf("open problems = %+v, want one for NW-CHAIR-LT", problems)
	}
}

func TestImportingSameUpdateTwiceCreatesNoDuplicates(t *testing.T) {
	e := newEnv(t)
	im := e.newImporter(context.Background())

	e.importUpdate(im, "NW-2025-09-01")
	before := e.products()

	second := e.importUpdate(im, "NW-2025-09-01")

	if second.Status != catalog.RunCompleted {
		t.Fatalf("status = %s", second.Status)
	}
	if second.Counts.Created != 0 || second.Counts.Updated != 0 {
		t.Errorf("second import changed data: %+v", second.Counts)
	}
	after := e.products()
	if len(after) != len(before) {
		t.Errorf("product count changed from %d to %d", len(before), len(after))
	}
}

func TestOutOfOrderAndOlderDataDoNotWin(t *testing.T) {
	e := newEnv(t)
	im := e.newImporter(context.Background())

	e.importUpdate(im, "NW-2025-09-01")
	e.importUpdate(im, "NW-2025-09-08")
	late := e.importUpdate(im, "NW-2025-09-05") // older update delivered late

	if late.Status != catalog.RunCompleted {
		t.Fatalf("status = %s", late.Status)
	}
	p := e.products()

	// Within update 09-08 the headlamp's newest record came first; the older one must be skipped.
	if lamp := p["NW-LAMP-200"]; lamp.Name != "Lumen 300 Headlamp" || lamp.PriceCents != 2999 {
		t.Errorf("headlamp = %q %d, want newest data", lamp.Name, lamp.PriceCents)
	}
	// The late 09-05 price must not overwrite the 09-08 price.
	if tent := p["NW-TENT-2P"]; tent.PriceCents != 13999 {
		t.Errorf("tent price = %d, want 13999", tent.PriceCents)
	}
	// The late 09-05 record must not bring back the stove withdrawn on 09-08.
	if stove := p["NW-STOVE-MINI"]; stove.Status != catalog.StatusWithdrawn {
		t.Errorf("stove status = %s, want withdrawn", stove.Status)
	}
	// Where nothing newer exists, the late update still applies.
	if poles := p["NW-POLES-CF"]; poles.PriceCents != 7999 {
		t.Errorf("poles price = %d, want 7999", poles.PriceCents)
	}
	if pad := p["NW-PAD-UL"]; pad.Status != catalog.StatusWithdrawn {
		t.Errorf("pad status = %s, want withdrawn", pad.Status)
	}
	// A withdrawal for an unknown SKU is stored as withdrawn, never active.
	if gps := p["NW-GPS-X1"]; gps.Status != catalog.StatusWithdrawn {
		t.Errorf("gps status = %q, want withdrawn", gps.Status)
	}

	// The chair problem was fixed by 09-08; the broken timestamp for NW-PACK-40 is still open.
	problems := e.openProblems()
	if len(problems) != 1 || problems[0].SKU != "NW-PACK-40" {
		t.Errorf("open problems = %+v, want only NW-PACK-40", problems)
	}
}

func TestSupplierOutageKeepsCatalogAndCanResume(t *testing.T) {
	e := newEnv(t)
	im := e.newImporter(context.Background())
	e.importUpdate(im, "NW-2025-09-01")
	catalogBefore := e.products()

	// Supplier goes down right after batch 2 of the big update.
	im.AfterBatch = func(runID int64, batch int) {
		if batch == 2 {
			e.fake.SetOutage(true)
		}
	}
	run := e.importUpdate(im, "NW-2025-09-15")

	if run.Status != catalog.RunInterrupted {
		t.Fatalf("status = %s, want interrupted", run.Status)
	}
	if run.BatchesDone() != 2 {
		t.Errorf("batches done = %d, want 2", run.BatchesDone())
	}
	// Existing catalog is untouched, and the 2 finished batches are kept.
	after := e.products()
	for sku := range catalogBefore {
		if _, ok := after[sku]; !ok {
			t.Errorf("product %s disappeared during the outage", sku)
		}
	}
	if got, want := len(after), len(catalogBefore)+8; got != want {
		t.Errorf("products = %d, want %d", got, want)
	}
	if !hasOpenProblem(e.openProblems(), catalog.ProblemSupplierUnavailable) {
		t.Error("expected an open supplier_unavailable problem")
	}

	// Supplier is back: resume finishes the remaining 4 batches.
	e.fake.SetOutage(false)
	im.AfterBatch = nil
	if _, err := im.Resume(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	im.Wait()

	run = e.run(run.ID)
	if run.Status != catalog.RunCompleted || run.BatchesDone() != 6 {
		t.Fatalf("after resume: status = %s, batches = %d", run.Status, run.BatchesDone())
	}
	if run.Counts.Created != 24 {
		t.Errorf("created = %d, want 24 (no batch applied twice)", run.Counts.Created)
	}
	if hasOpenProblem(e.openProblems(), catalog.ProblemSupplierUnavailable) {
		t.Error("supplier problem should be resolved after the run completed")
	}
}

func TestResumeAfterApplicationRestart(t *testing.T) {
	e := newEnv(t)

	// First "process": stops (like Ctrl+C) right after batch 3 of 6.
	ctx, cancel := context.WithCancel(context.Background())
	im := e.newImporter(ctx)
	im.AfterBatch = func(runID int64, batch int) {
		if batch == 3 {
			cancel()
		}
	}
	run := e.importUpdate(im, "NW-2025-09-15")
	if run.Status != catalog.RunInterrupted || run.BatchesDone() != 3 {
		t.Fatalf("status = %s, batches = %d; want interrupted after 3", run.Status, run.BatchesDone())
	}

	// Restart: close and reopen the database, new importer.
	e.store.Close()
	e.openStore()
	im = e.newImporter(context.Background())
	if err := im.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := im.Resume(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	im.Wait()

	run = e.run(run.ID)
	if run.Status != catalog.RunCompleted {
		t.Fatalf("status = %s, want completed", run.Status)
	}
	if run.Counts.Created != 24 || len(e.products()) != 24 {
		t.Errorf("created = %d, products = %d; want 24 each", run.Counts.Created, len(e.products()))
	}
}

func TestRecoverMarksCrashedRunInterrupted(t *testing.T) {
	e := newEnv(t)
	// A run left as "running" in the database = the process was killed.
	crashed, err := e.store.CreateRun(context.Background(), "NW-2025-09-15", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	im := e.newImporter(context.Background())
	if err := im.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}

	run := e.run(crashed.ID)
	if run.Status != catalog.RunInterrupted {
		t.Errorf("status = %s, want interrupted", run.Status)
	}
	if !hasOpenProblem(e.openProblems(), catalog.ProblemUpdateInterrupted) {
		t.Error("expected an open update_interrupted problem")
	}
	// A new update can't start while this one is unfinished.
	if _, err := im.Start(context.Background(), "NW-2025-09-01"); err == nil {
		t.Error("expected Start to fail while a run is interrupted")
	}
}

func TestAbandonedRunIsNeverCompleted(t *testing.T) {
	e := newEnv(t)
	e.fake.SetOutage(true)
	im := e.newImporter(context.Background())

	run := e.importUpdate(im, "NW-2025-09-01")
	if run.Status != catalog.RunInterrupted || run.BatchesDone() != 0 {
		t.Fatalf("status = %s, batches = %d", run.Status, run.BatchesDone())
	}
	if _, err := im.Abandon(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	if got := e.run(run.ID).Status; got != catalog.RunAbandoned {
		t.Errorf("status = %s, want abandoned", got)
	}
	if _, err := im.Resume(context.Background(), run.ID); err == nil {
		t.Error("an abandoned run must not be resumable")
	}
}

func hasOpenProblem(problems []catalog.Problem, kind string) bool {
	for _, p := range problems {
		if p.Kind == kind {
			return true
		}
	}
	return false
}
