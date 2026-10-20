#!/bin/sh
# Start fake supplier in background
/app/fakesupplier -addr :8081 -delay 100ms &

# Wait a moment for supplier to bind
sleep 1

# Start main catalog server with web UI in foreground
exec /app/server -addr :8080 -supplier http://localhost:8081 -static /app/web/dist -db /data/catalog.db
