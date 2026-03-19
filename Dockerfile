# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install build dependencies for cgo (needed by some sqlite drivers;
# modernc.org/sqlite is pure-Go so this is just a safety net)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/atask ./cmd/atask

# ── Final stage ───────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/atask /app/atask

# Non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir -p /app/data && chown appuser:appgroup /app/data
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/atask"]
