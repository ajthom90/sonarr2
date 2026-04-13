# M23 — Release Engineering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add release workflow, update checker, and finalize the project for distribution.

**Architecture:** A GitHub Actions workflow for tagged releases, an `updatecheck` package that queries GitHub Releases API, and an update health check.

**Tech Stack:** GitHub Actions, Go stdlib (`net/http`, `encoding/json`)

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Create | `.github/workflows/release.yml` | Multi-arch binary build + GitHub Release on tag push |
| Modify | `.github/workflows/docker.yml` | Add semver tags on tag push |
| Create | `internal/updatecheck/updatecheck.go` | GitHub Releases API checker with 24h cache |
| Create | `internal/updatecheck/updatecheck_test.go` | Tests with httptest mock |
| Create | `internal/health/update.go` | Update health check |
| Create | `internal/health/update_test.go` | Update health check tests |
| Modify | `internal/app/app.go` | Wire update checker + health check |
| Modify | `README.md` | Final README update for M23 |

---

### Task 1: Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`
- Modify: `.github/workflows/docker.yml`

The release workflow builds binaries for 4 platforms and creates a GitHub Release.

### Task 2: Update Checker

**Files:**
- Create: `internal/updatecheck/updatecheck.go`
- Create: `internal/updatecheck/updatecheck_test.go`

```go
package updatecheck

type Checker struct {
	currentVersion string
	repoOwner      string
	repoName       string
	httpClient     *http.Client
	mu             sync.Mutex
	cached         *Result
	cachedAt       time.Time
	cacheTTL       time.Duration
}

type Result struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	LatestVersion   string `json:"latestVersion"`
	CurrentVersion  string `json:"currentVersion"`
}

func New(currentVersion, repoOwner, repoName string) *Checker
func (c *Checker) Check(ctx context.Context) (*Result, error)
// Queries https://api.github.com/repos/{owner}/{repo}/releases/latest
// Caches result for 24 hours
// Compares tag_name (stripped of "v" prefix) against currentVersion
```

Tests with httptest server returning a mock GitHub release response.

### Task 3: Update Health Check + Wire

**Files:**
- Create: `internal/health/update.go`
- Create: `internal/health/update_test.go`
- Modify: `internal/app/app.go`
- Modify: `README.md`

```go
type UpdateCheck struct {
	checker *updatecheck.Checker
}

func NewUpdateCheck(checker *updatecheck.Checker) *UpdateCheck
func (c *UpdateCheck) Name() string { return "UpdateCheck" }
func (c *UpdateCheck) Check(ctx context.Context) []Result
// If update available, return notice-level result
```

Wire in app.go, bump README to M23.
