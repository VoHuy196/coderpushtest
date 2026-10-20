// Package main is the root entrypoint.
// It allows cloud platforms like Fly.io or Render to build directly from the repository root
// via 'go build .' without needing custom subpath configuration.
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
	"supplier-catalog/internal/fakesupplier"
	"supplier-catalog/internal/importer"
	"supplier-catalog/internal/store"
	"supplier-catalog/internal/supplier"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "catalog.db", "SQLite database file")
	supplierURL := flag.String("supplier", "http://localhost:8081", "supplier API base URL")
	staticDir := flag.String("static", "web/dist", "folder with the built web UI")
	startFakeSupplier := flag.Bool("fake-supplier", true, "start embedded fake supplier on :8081")
	supplierAddr := flag.String("supplier-addr", ":8081", "fake supplier listen address")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// If PORT environment variable is set by cloud platform (Fly.io / Render)
	if envPort := os.Getenv("PORT"); envPort != "" {
		*addr = ":" + envPort
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Optionally run embedded fake supplier so everything works in 1 single container
	if *startFakeSupplier {
		fakeServer := fakesupplier.New(fakesupplier.DemoUpdates())
		fakeServer.SetDelay(100 * time.Millisecond)

		fakeHTTP := &http.Server{
			Addr:              *supplierAddr,
			Handler:           fakeServer.Handler(),
			ReadHeaderTimeout: 5 * time.Second,
		}

		go func() {
			log.Info("embedded fake supplier listening", "addr", *supplierAddr)
			if err := fakeHTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("fake supplier error", "err", err)
			}
		}()

		go func() {
			<-ctx.Done()
			fakeHTTP.Close()
		}()
	}

	// 2. Open SQLite Store
	st, err := store.Open(*dbPath)
	if err != nil {
		log.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	client := supplier.NewClient(*supplierURL)
	imp := importer.New(ctx, st, client, log)

	if err := imp.Recover(ctx); err != nil {
		log.Error("failed recovery check", "err", err)
	}

	// 3. Start Main Server (JSON API + Web UI)
	srv := &http.Server{
		Addr: *addr,
		Handler: (&api.Server{
			Store:       st,
			Importer:    imp,
			Supplier:    client,
			SupplierURL: *supplierURL,
			StaticDir:   *staticDir,
			Log:         log,
		}).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down main server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Info("catalog server listening", "addr", *addr, "supplier", *supplierURL, "static", *staticDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
	imp.Wait()
}
