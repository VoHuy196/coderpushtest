// Package fakesupplier is a small fake of the supplier's HTTP API with
// fictional data. It can simulate an outage and slow responses for demos.
package fakesupplier

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"supplier-catalog/internal/supplier"
)

// Update is one published update split into batches.
type Update struct {
	ID          string
	Description string
	PublishedAt string
	Batches     [][]supplier.Product
}

// Server serves updates over HTTP.
type Server struct {
	mu      sync.Mutex
	updates []Update
	outage  bool
	delay   time.Duration
}

// New creates a server with the given updates.
func New(updates []Update) *Server {
	return &Server{updates: updates}
}

// SetOutage turns the simulated outage on or off. During an outage every
// catalog request fails with 503.
func (s *Server) SetOutage(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outage = on
}

// SetDelay makes every batch response wait, so a demo has time to interrupt.
func (s *Server) SetDelay(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delay = d
}

func (s *Server) state() (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outage, s.delay
}

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.rootInfo)
	mux.HandleFunc("GET /updates", s.listUpdates)
	mux.HandleFunc("GET /updates/{id}/batches/{n}", s.getBatch)
	mux.HandleFunc("GET /admin/outage", s.getOutage)
	mux.HandleFunc("POST /admin/outage", s.postOutage)
	return mux
}

func (s *Server) rootInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>Fake Supplier API</title></head><body style="font-family:system-ui,sans-serif;max-width:600px;margin:3rem auto;padding:1rem;line-height:1.6;background:#f8fafc;color:#1e293b;"><div style="background:white;padding:2rem;border-radius:12px;box-shadow:0 4px 6px -1px rgba(0,0,0,0.1);border:1px solid #e2e8f0;"><h2 style="margin-top:0;color:#0f172a;">🏕️ Fake Supplier API (Port 8081)</h2><p>This service provides fictional catalog batch updates for demonstration.</p><p style="margin:1.5rem 0;padding:1rem;background:#ecfdf5;border:1px solid #a7f3d0;border-radius:8px;">👉 <b><a href="http://localhost:8080" style="color:#0f9d76;text-decoration:none;font-size:1.1rem;">Click here to open the Web Dashboard (http://localhost:8080)</a></b></p><p style="font-size:0.9rem;color:#64748b;">API Endpoints: <a href="/updates">/updates</a> &bull; <a href="/admin/outage">/admin/outage</a></p></div></body></html>`))
}

func (s *Server) listUpdates(w http.ResponseWriter, r *http.Request) {
	if outage, _ := s.state(); outage {
		http.Error(w, "supplier is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	infos := []supplier.UpdateInfo{}
	for _, u := range s.updates {
		infos = append(infos, supplier.UpdateInfo{
			ID: u.ID, Description: u.Description, PublishedAt: u.PublishedAt, TotalBatches: len(u.Batches),
		})
	}
	writeJSON(w, infos)
}

func (s *Server) getBatch(w http.ResponseWriter, r *http.Request) {
	outage, delay := s.state()
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
	}
	if outage {
		http.Error(w, "supplier is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	var update *Update
	for i := range s.updates {
		if s.updates[i].ID == r.PathValue("id") {
			update = &s.updates[i]
		}
	}
	if update == nil {
		http.Error(w, "unknown update", http.StatusNotFound)
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 || n > len(update.Batches) {
		http.Error(w, "unknown batch", http.StatusNotFound)
		return
	}

	writeJSON(w, supplier.Batch{
		UpdateID:     update.ID,
		Number:       n,
		TotalBatches: len(update.Batches),
		Last:         n == len(update.Batches),
		Products:     update.Batches[n-1],
	})
}

func (s *Server) getOutage(w http.ResponseWriter, r *http.Request) {
	outage, _ := s.state()
	writeJSON(w, map[string]bool{"enabled": outage})
}

func (s *Server) postOutage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	s.SetOutage(body.Enabled)
	writeJSON(w, map[string]bool{"enabled": body.Enabled})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
