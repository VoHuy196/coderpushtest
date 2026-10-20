# Build Stage for Go binaries
FROM golang:1.23-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both services and the CLI
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/fakesupplier ./cmd/fakesupplier
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/catalog ./cmd/catalog

# Final Lightweight Runtime Image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create persistent data directory for SQLite
RUN mkdir -p /data /app/web/dist

# Copy binaries and startup script
COPY --from=builder /out/server /app/server
COPY --from=builder /out/fakesupplier /app/fakesupplier
COPY --from=builder /out/catalog /app/catalog
COPY web/dist /app/web/dist
COPY start.sh /app/start.sh

RUN chmod +x /app/start.sh /app/server /app/fakesupplier /app/catalog

EXPOSE 8080

CMD ["/app/start.sh"]
