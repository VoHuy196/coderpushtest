// Package api is the JSON HTTP API used by the web UI.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"supplier-catalog/internal/importer"
	"supplier-catalog/internal/store"
	"supplier-catalog/internal/supplier"
)

// Server holds the API dependencies.
type Server struct {
	Store       *store.Store
	Importer    *importer.Importer
	Supplier    *supplier.Client
	SupplierURL string // only used by the demo outage switch
	StaticDir   string // built web UI, optional
	Log         *slog.Logger
}

// Routes returns all HTTP routes.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/summary", s.getSummary)
	mux.HandleFunc("GET /api/supplier/updates", s.listSupplierUpdates)

	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("POST /api/runs", s.startRun)
	mux.HandleFunc("GET /api/runs/{id}", s.getRun)
	mux.HandleFunc("GET /api/runs/{id}/changes", s.listChanges)
	mux.HandleFunc("POST /api/runs/{id}/resume", s.resumeRun)
	mux.HandleFunc("POST /api/runs/{id}/abandon", s.abandonRun)

	mux.HandleFunc("GET /api/products", s.listProducts)
	mux.HandleFunc("GET /api/problems", s.listProblems)
	mux.HandleFunc("POST /api/problems/{id}/resolve", s.resolveProblem)

	// Demo only: switch the fake supplier's outage on/off from the UI.
	mux.HandleFunc("GET /api/demo/outage", s.proxyOutage)
	mux.HandleFunc("POST /api/demo/outage", s.proxyOutage)

	if s.StaticDir != "" {
		absDir, err := filepath.Abs(s.StaticDir)
		if err == nil {
			if info, err := os.Stat(absDir); err == nil && info.IsDir() {
				fs := http.FileServer(http.Dir(absDir))
				mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
					// Check if physical file exists
					target := filepath.Join(absDir, filepath.Clean(r.URL.Path))
					if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
						fs.ServeHTTP(w, r)
						return
					}
					// Serve index.html for root or SPA client routes
					http.ServeFile(w, r, filepath.Join(absDir, "index.html"))
				})
			}
		}
	}
	return mux
}

func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {
	sum, err := s.Store.GetSummary(r.Context())
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) listSupplierUpdates(w http.ResponseWriter, r *http.Request) {
	updates, err := s.Supplier.ListUpdates(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach supplier: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updates)
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.Store.ListRuns(r.Context(), 50)
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) startRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UpdateID string `json:"update_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	run, err := s.Importer.Start(r.Context(), body.UpdateID)
	if err != nil {
		s.actionError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	run, err := s.Store.GetRun(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) listChanges(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	changes, err := s.Store.ListChanges(r.Context(), id)
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, changes)
}

func (s *Server) resumeRun(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	run, err := s.Importer.Resume(r.Context(), id)
	if err != nil {
		s.actionError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) abandonRun(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	run, err := s.Importer.Abandon(r.Context(), id)
	if err != nil {
		s.actionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.Store.ListProducts(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (s *Server) listProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := s.Store.ListProblems(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, problems)
}

func (s *Server) resolveProblem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	err := s.Store.ResolveProblem(r.Context(), id, "resolved by staff", time.Now())
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "open problem not found")
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// proxyOutage forwards the request to the fake supplier's /admin/outage.
func (s *Server) proxyOutage(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	req, err := http.NewRequestWithContext(r.Context(), r.Method, s.SupplierURL+"/admin/outage", bytes.NewReader(body))
	if err != nil {
		s.serverError(w, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach fake supplier: "+err.Error())
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ---------- helpers ----------

// actionError maps start/resume/abandon errors to HTTP statuses.
func (s *Server) actionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, importer.ErrBusy), errors.Is(err, store.ErrOpenRun):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		// Validation-type errors from the importer ("only interrupted updates can be resumed").
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	s.Log.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
