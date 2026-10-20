# Stage 1: Build Frontend (React + Vite)
FROM node:20-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Backend (Go binaries)
FROM golang:1.23-alpine AS go-builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/fakesupplier ./cmd/fakesupplier
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/catalog ./cmd/catalog

# Stage 3: Lightweight Runtime Image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

RUN mkdir -p /data /app/web/dist

COPY --from=go-builder /out/server /app/server
COPY --from=go-builder /out/fakesupplier /app/fakesupplier
COPY --from=go-builder /out/catalog /app/catalog
COPY --from=web-builder /web/dist /app/web/dist
COPY start.sh /app/start.sh

RUN chmod +x /app/start.sh /app/server /app/fakesupplier /app/catalog

EXPOSE 8080

CMD ["/app/start.sh"]
