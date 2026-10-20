// Package catalog holds the core types and the rules for merging supplier data.
package catalog

import "time"

// Product statuses.
const (
	StatusActive    = "active"
	StatusWithdrawn = "withdrawn"
)

// Run statuses.
const (
	RunRunning     = "running"     // a worker is importing batches right now
	RunInterrupted = "interrupted" // stopped early (supplier error or app stopped); can be resumed
	RunCompleted   = "completed"   // every batch, including the last one, was saved
	RunAbandoned   = "abandoned"   // staff gave up on it; already saved batches stay
)

// Problem kinds.
const (
	ProblemInvalidProduct      = "invalid_product"      // a record failed validation
	ProblemSupplierUnavailable = "supplier_unavailable" // could not fetch a batch
	ProblemUpdateInterrupted   = "update_interrupted"   // the application stopped mid-update
)

// Product is our local copy of a supplier product.
//
// Withdrawn products are kept (not deleted) together with the supplier
// timestamp. That way an older record that arrives later can't bring them back.
type Product struct {
	SKU               string    `json:"sku"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	PriceCents        int64     `json:"price_cents"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	SupplierUpdatedAt time.Time `json:"supplier_updated_at"`
	LastRunID         int64     `json:"last_run_id"`
	LocalUpdatedAt    time.Time `json:"local_updated_at"`
}

// Run is one attempt to import one supplier update.
type Run struct {
	ID               int64      `json:"id"`
	SupplierUpdateID string     `json:"supplier_update_id"`
	Status           string     `json:"status"`
	NextBatch        int        `json:"next_batch"` // first batch that is NOT saved yet
	TotalBatches     int        `json:"total_batches"`
	Counts           Counts     `json:"counts"`
	LastError        string     `json:"last_error"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at"`
	OpenProblems     int        `json:"open_problems"`
}

// BatchesDone is the number of batches already saved.
func (r Run) BatchesDone() int { return r.NextBatch - 1 }

// Change is one line in a run's change log.
type Change struct {
	ID        int64     `json:"id"`
	RunID     int64     `json:"run_id"`
	Batch     int       `json:"batch"`
	SKU       string    `json:"sku"`
	Action    Action    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// Problem is something staff may need to look at.
type Problem struct {
	ID         int64      `json:"id"`
	RunID      int64      `json:"run_id"`
	Batch      int        `json:"batch"`
	SKU        string     `json:"sku"`
	Kind       string     `json:"kind"`
	Message    string     `json:"message"`
	Status     string     `json:"status"` // open | resolved
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	Resolution string     `json:"resolution"`
}

// Counts summarises what a run did.
type Counts struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Withdrawn int `json:"withdrawn"`
	Skipped   int `json:"skipped"`
	Invalid   int `json:"invalid"`
}

// Add counts one action.
func (c *Counts) Add(a Action) {
	switch a {
	case ActionCreate:
		c.Created++
	case ActionUpdate:
		c.Updated++
	case ActionWithdraw:
		c.Withdrawn++
	case ActionSkip:
		c.Skipped++
	case ActionInvalid:
		c.Invalid++
	}
}
