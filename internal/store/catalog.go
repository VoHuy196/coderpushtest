package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"supplier-catalog/internal/catalog"
)

// ---------- products ----------

const productColumns = `sku, name, description, price_cents, currency, status, supplier_updated_at, last_run_id, local_updated_at`

func scanProduct(row interface{ Scan(...any) error }) (catalog.Product, error) {
	var p catalog.Product
	var supplierAt, localAt string
	err := row.Scan(&p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Currency, &p.Status, &supplierAt, &p.LastRunID, &localAt)
	p.SupplierUpdatedAt = parseTime(supplierAt)
	p.LocalUpdatedAt = parseTime(localAt)
	return p, err
}

// GetProduct returns the stored product, or nil if the SKU is unknown.
func (tx *Tx) GetProduct(ctx context.Context, sku string) (*catalog.Product, error) {
	row := tx.q.QueryRowContext(ctx, `SELECT `+productColumns+` FROM products WHERE sku = ?`, sku)
	p, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SaveProduct inserts or replaces a product. The SKU is the primary key, so
// importing the same product twice can never create a duplicate row.
func (tx *Tx) SaveProduct(ctx context.Context, p catalog.Product) error {
	_, err := tx.q.ExecContext(ctx, `
		INSERT INTO products (`+productColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(sku) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			price_cents = excluded.price_cents,
			currency = excluded.currency,
			status = excluded.status,
			supplier_updated_at = excluded.supplier_updated_at,
			last_run_id = excluded.last_run_id,
			local_updated_at = excluded.local_updated_at`,
		p.SKU, p.Name, p.Description, p.PriceCents, p.Currency, p.Status,
		formatTime(p.SupplierUpdatedAt), p.LastRunID, formatTime(p.LocalUpdatedAt))
	return err
}

// ListProducts returns products ordered by SKU. status may be "" for all.
func (s *Store) ListProducts(ctx context.Context, status string) ([]catalog.Product, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+productColumns+` FROM products WHERE (? = '' OR status = ?) ORDER BY sku`, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []catalog.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// Summary is a few numbers for the dashboard.
type Summary struct {
	ActiveProducts    int `json:"active_products"`
	WithdrawnProducts int `json:"withdrawn_products"`
	OpenProblems      int `json:"open_problems"`
}

// GetSummary returns dashboard numbers.
func (s *Store) GetSummary(ctx context.Context) (Summary, error) {
	var sum Summary
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM products WHERE status = 'active'),
			(SELECT COUNT(*) FROM products WHERE status = 'withdrawn'),
			(SELECT COUNT(*) FROM problems WHERE status = 'open')`).
		Scan(&sum.ActiveProducts, &sum.WithdrawnProducts, &sum.OpenProblems)
	return sum, err
}

// ---------- change log ----------

// AddChange writes one line to a run's change log.
func (tx *Tx) AddChange(ctx context.Context, c catalog.Change) error {
	_, err := tx.q.ExecContext(ctx,
		`INSERT INTO run_changes (run_id, batch, sku, action, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.RunID, c.Batch, c.SKU, string(c.Action), c.Detail, formatTime(c.CreatedAt))
	return err
}

// ListChanges returns the change log of a run in the order it happened.
func (s *Store) ListChanges(ctx context.Context, runID int64) ([]catalog.Change, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, batch, sku, action, detail, created_at FROM run_changes WHERE run_id = ? ORDER BY id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	changes := []catalog.Change{}
	for rows.Next() {
		var c catalog.Change
		var action, created string
		if err := rows.Scan(&c.ID, &c.RunID, &c.Batch, &c.SKU, &action, &c.Detail, &created); err != nil {
			return nil, err
		}
		c.Action = catalog.Action(action)
		c.CreatedAt = parseTime(created)
		changes = append(changes, c)
	}
	return changes, rows.Err()
}

// ---------- problems ----------

func addProblem(ctx context.Context, q querier, p catalog.Problem) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO problems (run_id, batch, sku, kind, message, status, created_at) VALUES (?, ?, ?, ?, ?, 'open', ?)`,
		p.RunID, p.Batch, p.SKU, p.Kind, p.Message, formatTime(p.CreatedAt))
	return err
}

// AddProblem records a problem inside the batch transaction.
func (tx *Tx) AddProblem(ctx context.Context, p catalog.Problem) error {
	return addProblem(ctx, tx.q, p)
}

// AddProblem records a problem outside of a batch (e.g. supplier is down).
func (s *Store) AddProblem(ctx context.Context, p catalog.Problem) error {
	return addProblem(ctx, s.db, p)
}

// ResolveProductProblems closes open "invalid product" problems for a SKU,
// because the supplier has now sent valid, newer data for it.
func (tx *Tx) ResolveProductProblems(ctx context.Context, sku, resolution string, now time.Time) error {
	_, err := tx.q.ExecContext(ctx,
		`UPDATE problems SET status = 'resolved', resolved_at = ?, resolution = ?
		 WHERE sku = ? AND kind = ? AND status = 'open'`,
		formatTime(now), resolution, sku, catalog.ProblemInvalidProduct)
	return err
}

func resolveRunProblems(ctx context.Context, q querier, runID int64, resolution string, now time.Time) error {
	_, err := q.ExecContext(ctx,
		`UPDATE problems SET status = 'resolved', resolved_at = ?, resolution = ?
		 WHERE run_id = ? AND kind IN (?, ?) AND status = 'open'`,
		formatTime(now), resolution, runID, catalog.ProblemSupplierUnavailable, catalog.ProblemUpdateInterrupted)
	return err
}

// ResolveRunProblems closes a run's "supplier unavailable" and "interrupted"
// problems, e.g. once the run has completed.
func (tx *Tx) ResolveRunProblems(ctx context.Context, runID int64, resolution string, now time.Time) error {
	return resolveRunProblems(ctx, tx.q, runID, resolution, now)
}

// ResolveRunProblems is the same as Tx.ResolveRunProblems, outside a transaction.
func (s *Store) ResolveRunProblems(ctx context.Context, runID int64, resolution string, now time.Time) error {
	return resolveRunProblems(ctx, s.db, runID, resolution, now)
}

// ResolveProblem lets staff close a problem by hand.
func (s *Store) ResolveProblem(ctx context.Context, id int64, resolution string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE problems SET status = 'resolved', resolved_at = ?, resolution = ? WHERE id = ? AND status = 'open'`,
		formatTime(now), resolution, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListProblems returns problems, newest first. status may be "" for all.
func (s *Store) ListProblems(ctx context.Context, status string) ([]catalog.Problem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, run_id, batch, sku, kind, message, status, created_at, resolved_at, resolution
		FROM problems WHERE (? = '' OR status = ?) ORDER BY id DESC`, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	problems := []catalog.Problem{}
	for rows.Next() {
		var p catalog.Problem
		var created, resolved string
		if err := rows.Scan(&p.ID, &p.RunID, &p.Batch, &p.SKU, &p.Kind, &p.Message, &p.Status, &created, &resolved, &p.Resolution); err != nil {
			return nil, err
		}
		p.CreatedAt = parseTime(created)
		p.ResolvedAt = parseOptionalTime(resolved)
		problems = append(problems, p)
	}
	return problems, rows.Err()
}
