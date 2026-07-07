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

# DATA_DIR holds PocketBase's own data.db (auth/users/settings) and the
# domain atask.db SQLite file side by side (internal/config, internal/store) —
# a single directory, must be a named volume so both survive container
# restarts/recreation.
ENV DATA_DIR=/app/data
VOLUME ["/app/data"]

EXPOSE 8080

# No hardcoded subcommand. cmd/atask/main.go's hasSubcommand check only
# injects `serve --http=...` defaults when os.Args carries nothing but flags;
# real subcommands (admin create-user, admin assign-data, migrate, superuser)
# pass through to cobra untouched. That means `docker run <img>` serves, and
# `docker run <img> admin create-user ...` also works. Hardcoding
# ENTRYPOINT ["/app/atask", "serve"] would break the latter: "admin" would be
# appended as an argument to `serve` instead of dispatching to the admin
# subcommand.
ENTRYPOINT ["/app/atask"]
