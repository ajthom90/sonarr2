# M23 — Release Engineering

## Overview

Add a GitHub Actions release workflow triggered by git tags, multi-arch binary distribution, Docker version tagging, and a built-in update checker that reports when a newer version is available.

## Features

### 1. Release Workflow (`.github/workflows/release.yml`)

Triggered on `v*` tags (e.g., `v1.0.0`). Steps:
1. Build multi-arch Go binaries: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
2. Build `sonarr-migrate` binary for the same targets
3. Create tar.gz archives for each (e.g., `sonarr2_1.0.0_linux_amd64.tar.gz`)
4. Generate SHA256 checksums file
5. Create GitHub Release with changelog from tag annotation + attach artifacts
6. Build and push Docker images with version tags (`:v1.0.0`, `:v1.0`, `:v1`, `:latest`)

### 2. Docker Version Tags

Enhance the existing `docker.yml` merge step to also push semantic version tags when triggered by a tag push. The metadata-action already supports this via `type=semver`.

### 3. Update Checker

New `internal/updatecheck/` package:
- Queries GitHub Releases API for the latest release
- Compares against `buildinfo.Version`
- Returns whether an update is available + the latest version
- Cached (checks at most once per 24 hours)

### 4. Update Health Check

Add `UpdateCheck` to the health system:
- Runs the update checker
- If a newer version is available, returns a `notice` (not warning — updates are informational)

### 5. System Status Enhancement

Add `latestVersion` field to the `/system/status` response so the frontend can show update availability.

## Configuration

No new config — the update checker uses the GitHub Releases API (public, no auth needed).

## Out of Scope

- Cosign signing (add later)
- Nightly channel / automatic nightly builds
- Built-in auto-updater (just checks, doesn't download)
- Windows builds
- Playwright e2e tests
- Benchmark workflows
