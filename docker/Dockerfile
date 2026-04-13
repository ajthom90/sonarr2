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
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/sonarr2 /sonarr2
EXPOSE 8989
VOLUME ["/config", "/data"]
ENTRYPOINT ["/sonarr2"]
