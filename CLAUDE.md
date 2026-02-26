# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go web dashboard that monitors GitHub issues, pull requests, and CI check statuses across multiple ECMWF organization repositories. It uses background polling with configurable intervals and renders server-side HTML templates.

## Build & Run

```bash
# Requires Go 1.24
go mod tidy

# Build
make build

# Run (requires GITHUB_TOKEN env var and config.yaml)
GITHUB_TOKEN=<token> ./ecmwf-dash

# Run tests
make test

# Run vet
make vet
```

The server reads `config.yaml` from the working directory and expects `web/` to be present. Default listen address is `0.0.0.0:8000`.

## Docker

```bash
make docker-build
docker run -e GITHUB_TOKEN=<token> -v $(pwd)/config.yaml:/config.yaml -p 8000:8000 ecmwf-dash

# Or with docker-compose
GITHUB_TOKEN=<token> docker compose up
```

Note: `config.yaml` is NOT baked into the Docker image. Mount it at runtime. The Docker image includes a HEALTHCHECK via `/healthcheck` binary.

## Architecture

**Entry point**: `cmd/server/main.go` — wires up config (with validation), GitHub client, storage, fetcher, handlers, security headers middleware (including CSP), and HTTP routes.

**Data flow**: Three background goroutines (`internal/fetcher/`) poll the GitHub API at configured intervals and write results into an in-memory store (`internal/storage/memory.go`, RWMutex-protected). The store deep-copies all data on both get AND set operations to prevent shared mutable state. HTTP handlers read from the `storage.Store` interface and render Go HTML templates via buffered execution (`render.go`).

**Key packages**:
- `internal/config/` — Loads and validates `config.yaml` (repos, branches, fetch intervals, server settings)
- `internal/github/` — GitHub API client wrapper. `types.go` defines all data models + `isInternal()`. `review.go` contains `DeriveReviewStatus()` (extracted, testable). `helpers.go` has `sanitizeLabelColor()`, `computeTextColor()`, and `computeLabelStyle()` for safe label rendering. Separate files for issues, pulls, and actions fetching. Rate info returned from fetch functions (no extra API calls).
- `internal/storage/` — `Store` interface + `Memory` implementation. Thread-safe with deep-copy on both reads and writes. `LastFetchTimes()` on the Store interface for the health endpoint.
- `internal/handlers/` — HTTP handlers for three views: `/builds`, `/pulls`, `/issues`. `render.go` provides buffered template execution.
- `internal/fetcher/` — Orchestrates background polling goroutines with context cancellation. No redundant timestamp tracking (uses `Store.LastFetchTimes()`).

**Frontend**: Server-rendered Go templates in `web/templates/`, static assets in `web/static/`.

**Template structure**: Uses Go template inheritance via `base.html` which defines the shared `<head>`, nav, header, and footer. Page templates (`builds.html`, `dashboard.html`, `pullrequests.html`) define `title`, `stats`, `extra-css`, and `content` blocks. All pages use `renderTemplate()` for buffered output.

**CSS architecture**: Single ECMWF blue palette defined in `base.css` `:root` using CSS custom properties. No theme switching — one light theme only. `builds.css` holds build-page-specific styles (no fallback values — relies on base.css defaults).

**Label rendering**: Labels use `template.CSS` for safe inline styles. `computeLabelStyle()` produces `background-color` + WCAG-compliant text color. The `LabelStyle` field on `Label` struct is pre-computed during API fetch.

**API cost**: PR fetching is expensive — 3 API calls per open PR (list reviews, get full PR for `mergeable_state`, list check runs). Rate info is extracted from response headers (no separate rate limit API calls). All check run fetches use `filter=latest`.

## Routes

| Path | Handler | Description |
|------|---------|-------------|
| `/` | redirect | Redirects to `/builds` |
| `/builds` | BuildStatus | CI check status per repo/branch |
| `/builds-dashboard` | BuildsDashboard | TV/kiosk mode for builds (standalone, no base.html) |
| `/pulls` | PullRequests | Open PRs with reviews and checks |
| `/issues` | Dashboard | Open issues across repos |
| `/health` | inline | JSON health check with last-fetch timestamps |
| Unmatched | 404 | Proper 404 for unknown paths |

## Configuration

`config.yaml` must exist in the working directory. Config is validated on load.

```yaml
github:
  organization: ecmwf
  repositories:
    - name: eccodes
      branches: [master, develop]

fetch_intervals:
  issues: 30m
  pull_requests: 10m
  actions: 5m

server:
  host: "0.0.0.0"
  port: 8000
```

## Gotchas

- **Builds view renders all configured branches**: The handler uses `repoConfig` (from `config.yaml`) to determine which branches to display per repo. Branch order matches config order. Repos not in config are appended alphabetically with branches from API data.
- **GitHub API rate limits**: 12 repos x 2 branches = frequent polling. Rate limit warnings appear in logs when remaining < 100.
- **`CONTRIBUTOR` is treated as external**: `isInternal` only matches `OWNER`, `MEMBER`, or `COLLABORATOR`. Past contributors with merged PRs who aren't collaborators get the "external" badge.
- **Templates parsed at startup**: No hot reload. Editing a `.html` file requires a restart. Templates validated as non-nil in handler constructor (panics on nil).
- **Handler data uses anonymous structs**: Each handler defines its template data as an inline `struct{...}` literal — no shared type. Adding/removing a template field requires updating each handler separately.
- **No inline scripts**: CSP uses `script-src 'self'` only. All JS is in external files.
- **DISMISSED reviews**: A DISMISSED review removes the reviewer from the review map, reverting to their previous state (or removing them entirely).
- **Healthcheck hardcodes port 8000**: `cmd/healthcheck/main.go` always hits `localhost:8000`. Changing the port in `config.yaml` without updating the healthcheck binary will cause Docker health checks to fail.
- **Fetchers use partial failure tolerance**: Per-repo errors are logged and skipped; an error is returned only if ALL repos fail. If one repo 404s, the others still update. This means stale data can silently persist for a single broken repo.
- **`go-github` aliased as `gh`**: All fetch files (`actions.go`, `issues.go`, `pulls.go`) import `github.com/google/go-github/v66/github` as `gh` to avoid collision with the internal `github` package. New fetch code must follow this convention.
- **Broader check failure counting**: `timed_out`, `cancelled`, `action_required`, `neutral`, `stale` all count as failures — not just `"failure"`. This is intentional but shows more red than GitHub's native UI.

## Environment

- `GITHUB_TOKEN` (required) — GitHub personal access token for API access
- No CLI flags — all configuration is via `config.yaml` and environment variables
- `config.yaml` is gitignored — must be created locally or mounted at runtime
- CI/CD: `.github/workflows/release.yml` runs tests then builds and pushes Docker image to Harbor on GitHub release (tagged with both the release version and `latest`)
- `.github/workflows/ci.yml` runs tests with `-race`, vet, `govulncheck`, and a Docker build (no push) on push/PR

## Development Status

- Test suite covers storage (including deep-copy and concurrency), config validation, review status logic, and label color helpers
- Race detector enabled in CI and Makefile
- Security scanning via govulncheck (CI) and Trivy (release)
- Trivy scan is informational only (`exit-code: 0`) — won't block releases on vulnerabilities
- All CI actions pinned to full SHA (supply chain security)
- No linter configuration (uses `go vet`)
