// Command fakesupplier runs a fake supplier API with fictional data.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"supplier-catalog/internal/fakesupplier"
)

func main() {
	addr := flag.String("addr", ":8081", "HTTP listen address")
	delay := flag.Duration("delay", time.Second, "wait before each batch response (gives time to interrupt in a demo)")
	outage := flag.Bool("outage", false, "start with the supplier unavailable (HTTP 503)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	srv := fakesupplier.New(fakesupplier.DemoUpdates())
	srv.SetDelay(*delay)
	srv.SetOutage(*outage)

	log.Info("fake supplier listening", "addr", *addr, "delay", *delay, "outage", *outage)
	if err := http.ListenAndServe(*addr, logRequests(log, srv.Handler())); err != nil {
		log.Error("fake supplier failed", "err", err)
		os.Exit(1)
	}
}

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
