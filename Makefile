.PHONY: build build-backend frontend test lint tidy run clean

BIN := sonarr2
OUT := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/ajthom90/sonarr2/internal/buildinfo.Version=$(VERSION) \
	-X github.com/ajthom90/sonarr2/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/ajthom90/sonarr2/internal/buildinfo.Date=$(DATE)

## frontend: build the React/Vite frontend into web/dist/
frontend:
	cd frontend && npm ci && npm run build

## build: build the Go binary (runs the frontend build first)
build: frontend
	@mkdir -p $(OUT)
	CGO_ENABLED=0 go build -ldflags='$(LDFLAGS)' -o $(OUT)/$(BIN) ./cmd/sonarr

## build-backend: build the Go binary only (skips the frontend build)
build-backend:
	@mkdir -p $(OUT)
	CGO_ENABLED=0 go build -ldflags='$(LDFLAGS)' -o $(OUT)/$(BIN) ./cmd/sonarr

## build-migrate: build the sonarr-migrate CLI tool
build-migrate:
	@mkdir -p $(OUT)
	CGO_ENABLED=0 go build -ldflags='$(LDFLAGS)' -o $(OUT)/sonarr-migrate ./cmd/sonarr-migrate

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	@fmt=$$(gofmt -l -s .); if [ -n "$$fmt" ]; then echo "gofmt issues:"; echo "$$fmt"; exit 1; fi

tidy:
	go mod tidy

run: build
	./$(OUT)/$(BIN)

clean:
	rm -rf $(OUT) web/dist
