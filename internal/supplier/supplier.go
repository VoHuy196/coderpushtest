// Package supplier contains the supplier's wire format and an HTTP client for it.
package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Product is one product record as the supplier sends it.
//
// UpdatedAt is kept as a string on purpose: a broken timestamp in one record
// should become a "problem" for that record, not break the whole batch.
type Product struct {
	SKU         string `json:"sku"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	PriceCents  int64  `json:"price_cents,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Withdrawn   bool   `json:"withdrawn,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

// Batch is one page of an update.
type Batch struct {
	UpdateID     string    `json:"update_id"`
	Number       int       `json:"batch"`
	TotalBatches int       `json:"total_batches"`
	Last         bool      `json:"last"`
	Products     []Product `json:"products"`
}

// UpdateInfo describes an update that the supplier has published.
type UpdateInfo struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	PublishedAt  string `json:"published_at"`
	TotalBatches int    `json:"total_batches"`
}

// StatusError is returned when the supplier answers with a non-200 status.
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("supplier returned HTTP %d: %s", e.Code, strings.TrimSpace(e.Body))
}

// Retryable reports whether it makes sense to try a failed request again.
// Server errors, rate limits and network errors are retryable; 4xx are not.
func Retryable(err error) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Code >= 500 || se.Code == http.StatusTooManyRequests
	}
	return true
}

// Client talks to the supplier's HTTP API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a client for the supplier at baseURL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

// ListUpdates returns the updates the supplier has published.
func (c *Client) ListUpdates(ctx context.Context) ([]UpdateInfo, error) {
	var out []UpdateInfo
	err := c.getJSON(ctx, "/updates", &out)
	return out, err
}

// FetchBatch downloads one batch of an update. Batches are numbered from 1.
func (c *Client) FetchBatch(ctx context.Context, updateID string, number int) (Batch, error) {
	var b Batch
	path := fmt.Sprintf("/updates/%s/batches/%d", url.PathEscape(updateID), number)
	err := c.getJSON(ctx, path, &b)
	return b, err
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &StatusError{Code: resp.StatusCode, Body: string(body)}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode supplier response: %w", err)
	}
	return nil
}
