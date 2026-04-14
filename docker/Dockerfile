# syntax=docker/dockerfile:1.7

# ---- Frontend ----
FROM node:22-alpine AS frontend
WORKDIR /src

# Cache npm dependencies separately from source.
COPY frontend/package*.json frontend/
RUN cd frontend && npm ci

# Copy frontend source and the web placeholder so the output dir exists.
COPY frontend/ frontend/
COPY web/ web/

# Build outputs to web/dist/ (configured in vite.config.ts outDir: '../web/dist').
RUN cd frontend && npm run build

# ---- Backend builder ----
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache Go module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy all Go source.
COPY . .

# Overwrite the placeholder web/dist with the real frontend build.
COPY --from=frontend /src/web/dist /src/web/dist

ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w \
        -X github.com/ajthom90/sonarr2/internal/buildinfo.Version=$VERSION \
        -X github.com/ajthom90/sonarr2/internal/buildinfo.Commit=$COMMIT \
        -X github.com/ajthom90/sonarr2/internal/buildinfo.Date=$DATE" \
      -o /out/sonarr2 ./cmd/sonarr

# ---- Runtime ----
# Alpine-based so we can su-exec to a user-controlled PUID/PGID. Entrypoint
# reads those env vars, ensures /config is writable, and drops privileges
# before running the binary. Matches the LinuxServer.io convention that
# most *arr users expect on bind-mounted volumes.
#
# su-exec is a tiny (~10 KB) setuid-style helper from Alpine community; it
# forwards signals cleanly and doesn't leak a shell into the app's process
# tree. ca-certificates is needed for outbound HTTPS (TVDB, indexer APIs).
# tzdata is needed so scheduled tasks respect the host's TZ env var.
FROM alpine:3.20
RUN apk add --no-cache su-exec ca-certificates tzdata
COPY --from=builder /out/sonarr2 /sonarr2
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Run from /config so sonarr2's default DSN (file:./data/sonarr2.db) lands
# inside the persisted volume (at /config/data/sonarr2.db) instead of
# colliding with the /data media-library volume.
WORKDIR /config

ENV PUID=1000 \
    PGID=1000

EXPOSE 8989
VOLUME ["/config", "/data"]
ENTRYPOINT ["/entrypoint.sh"]
