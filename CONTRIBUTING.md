# Contributing to sonarr2

Thanks for your interest. This document explains how to set up a dev environment and submit changes.

## Dev requirements

- Go 1.23 or newer
- git
- Docker (for building release images and running integration tests against Postgres in later milestones)

## Getting started

```bash
git clone https://github.com/ajthom90/sonarr2.git
cd sonarr2
make test
make build
./dist/sonarr2 -port 18989 -bind 127.0.0.1 -log-format text
curl http://127.0.0.1:18989/ping
```

## Project layout

See the [design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) §3.2 for the full layout. At a high level:

- `cmd/sonarr/` — main entry point
- `internal/app/` — composition root
- `internal/api/` — HTTP server, routes, middleware
- `internal/config/` — configuration loading
- `internal/logging/` — structured logging setup
- `internal/buildinfo/` — build metadata

## Making changes

1. Open an issue first for any non-trivial change so we can discuss direction.
2. Create a branch off `main`. Use a descriptive name (e.g., `feat/rss-sync`, `fix/sqlite-busy-timeout`).
3. Follow TDD: write a failing test, make it pass, commit. Prefer many small commits over one large one.
4. Run `make lint test` before pushing.
5. Open a PR. CI must pass before review.

## Commit messages

- Lowercase, imperative, present tense.
- Use a scope: `api: return 404 on unknown routes`.
- Prefixes are free-form but common ones are `feat`, `fix`, `chore`, `docs`, `test`, `refactor`.
- Include a "why" in the body when the "what" isn't obvious from the diff.

## Coding standards

- `gofmt -s` formatted
- `go vet`, `staticcheck`, and `golangci-lint` must pass
- All exported identifiers have godoc comments
- Package-level comments on every `internal/*` package
- Tests in the same package (white-box) unless a test needs to avoid circular imports
- Table-driven tests preferred over multiple top-level test functions for closely related cases
- Run `go test -race ./...` — we gate CI on race detector output

## License

By contributing you agree your contribution will be licensed under the GPL-3.0 license (see [LICENSE](./LICENSE)).

### Porting code from Sonarr

sonarr2 and Sonarr are both GPL-3.0, so code may be studied and adapted from Sonarr directly. When a file contains code ported or adapted from Sonarr, add the following header at the top of the file to preserve upstream attribution:

```go
// SPDX-License-Identifier: GPL-3.0-or-later
// Portions adapted from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
```

Use the equivalent comment syntax for non-Go files (`//` for TS/JS, `#` for shell/yaml, `--` for SQL). Files that are fully independent work do not need the attribution line, but new files are still encouraged to carry the SPDX identifier.
