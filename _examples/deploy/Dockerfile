# Multi-stage Dockerfile for dark applications.
#
# Stage 1: Build the Go binary
# Stage 2: Minimal runtime image with views + static assets
#
# Usage:
#   docker build -t myapp .
#   docker run -p 3000:3000 myapp

# --- Build stage ---
FROM golang:1.26-bookworm AS builder

WORKDIR /src

# Cache Go module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source code.
COPY . .

# Build a statically linked binary.
# CGO_ENABLED=0 disables CGO; ramune's default JSC backend uses purego (no CGO).
RUN CGO_ENABLED=0 go build -o /app ./cmd/server

# --- Runtime stage ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create writable cache directory for island npm packages.
RUN mkdir -p /root/.cache/dark

WORKDIR /app

# Copy the binary.
COPY --from=builder /app /app/server

# Copy views (TSX templates are compiled at startup, not embedded in the binary).
COPY --from=builder /src/views ./views

# Copy static assets (if any).
COPY --from=builder /src/public ./public

EXPOSE 3000

CMD ["./server"]
