# syntax=docker/dockerfile:1.7

# ---- Builder ----
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source.
COPY . .

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
