// Command server runs the catalog application: the JSON API, the importer
// and (optionally) the built web UI.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"supplier-catalog/internal/api"
	"supplier-catalog/internal/importer"
	"supplier-catalog/internal/store"
	"supplier-catalog/internal/supplier"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "catalog.db", "SQLite database file")
	supplierURL := flag.String("supplier", "http://localhost:8081", "supplier API base URL")
	staticDir := flag.String("static", "web/dist", "folder with the built web UI (optional)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log, *addr, *dbPath, *supplierURL, *staticDir); err != nil {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, addr, dbPath, supplierURL, staticDir string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	// ctx is cancelled on Ctrl+C. The importer then stops after the current
	// step and marks its run as interrupted.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := supplier.NewClient(supplierURL)
	imp := importer.New(ctx, st, client, log)

	// Runs left as "running" by a crash are marked interrupted here.
	if err := imp.Recover(ctx); err != nil {
		return err
	}

	srv := &http.Server{
		Addr: addr,
		Handler: (&api.Server{
			Store:       st,
			Importer:    imp,
			Supplier:    client,
			SupplierURL: supplierURL,
			StaticDir:   staticDir,
			Log:         log,
		}).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Info("catalog server listening", "addr", addr, "supplier", supplierURL, "db", dbPath)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	imp.Wait()
	return nil
}
