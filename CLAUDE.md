# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go web dashboard that monitors GitHub issues, pull requests, and CI check statuses across multiple ECMWF organization repositories. It uses background polling with configurable intervals and renders server-side HTML templates.

## Build & Run

```bash
# Requires Go 1.26
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
docker run -e GITHUB_TOKEN=<token> -p 8000:8000 ecmwf-dash

# Or with docker-compose
GITHUB_TOKEN=<token> docker compose up
```

Note: `config.yaml` IS baked into the Docker image at build time. To use a different config, mount over it: `-v $(pwd)/config.yaml:/config.yaml`. The Docker image includes a HEALTHCHECK via `/healthcheck` binary.

## Architecture

**Entry point**: `cmd/server/main.go` — wires up config (with validation), GitHub client, storage, fetcher, handlers, security headers middleware (including CSP), and HTTP routes.

**Data flow**: Three background goroutines (`internal/fetcher/`) poll the GitHub API at configured intervals and write results into an in-memory store (`internal/storage/memory.go`, RWMutex-protected). The store deep-copies all data on both get AND set operations to prevent shared mutable state. HTTP handlers read from the `storage.Store` interface and render Go HTML templates via buffered execution (`render.go`).

**Key packages**:
- `internal/config/` — Loads and validates `config.yaml` (repos, branches, fetch intervals, server settings)
- `internal/github/` — GitHub API client wrapper. `types.go` defines all data models + `isInternal()`. `review.go` contains `DeriveReviewStatus()` (extracted, testable). `helpers.go` has `sanitizeLabelColor()`, `computeTextColor()`, and `computeLabelStyle()` for safe label rendering. `checks.go` has `ClassifyCheck()` — the single source of truth for check status classification. Separate files for issues, pulls, and actions fetching. Rate info returned from fetch functions (no extra API calls).
- `internal/storage/` — `Store` interface + `Memory` implementation. Thread-safe with deep-copy on both reads and writes. `LastFetchTimes()` and `RepoFetchTimes(category)` on the Store interface for health endpoint and per-repo staleness. `Merge*` methods for partial-failure updates (preserve existing data for failed repos, update succeeded repos). Category constants (`CategoryIssues`, `CategoryPRs`, `CategoryChecks`) prevent string-literal typos.
- `internal/handlers/` — HTTP handlers for three views: `/builds`, `/pulls`, `/issues`. `render.go` provides buffered template execution. `params.go` has shared input validation (`sanitizeSort`, `sanitizeOrder`, `sanitizeRepo`) and `paginate()`. `staleness.go` has `staleRepos()` helper, `computeStaleness()` method, and `FetchIntervals` type (deliberately separate from config to avoid import dependency). `funcmap.go` exports `TemplateFuncs()` — single source of truth for template functions. `HandlerConfig` struct replaces positional params in `New()`.
- `internal/fetcher/` — Orchestrates background polling goroutines with context cancellation. Uses `Store.Merge*` (not `Set*`) for partial-failure tolerance. No test coverage — depends on concrete `*github.Client` (no interface).

**Frontend**: Server-rendered Go templates in `web/templates/`, static assets in `web/static/`.

**Template structure**: Uses Go template inheritance via `base.html` which defines the shared `<head>`, nav, header, and footer. Page templates (`builds.html`, `dashboard.html`, `pullrequests.html`) define `title`, `stats`, `extra-css`, and `content` blocks. All pages use `renderTemplate()` for buffered output.

**CSS architecture**: Single ECMWF blue palette defined in `base.css` `:root` using CSS custom properties. Separate WCAG AA tokens for text use (`--link-color`, `--warning-text`, `--muted-text`) vs. non-text use (`--accent-color`, `--warning-color`, `--neutral-color`). No theme switching — one light theme only. `builds.css` holds build-page-specific styles. `tv.css` holds TV/kiosk-mode styles. `.tooltip-dot` in base.css provides shared tooltip pseudo-elements for both `.check-dot` and `.lane-dot`. PR-specific styles currently live in base.css (not yet extracted to a separate file).

**Label rendering**: Labels use `template.CSS` for safe inline styles. `computeLabelStyle()` produces `background-color` + WCAG-compliant text color. The `LabelStyle` field on `Label` struct is pre-computed during API fetch.

**API cost**: PR fetching is expensive — 3 API calls per open PR (list reviews, get full PR for `mergeable_state`, list check runs). Rate info is extracted from response headers (no separate rate limit API calls). All check run fetches use `filter=latest`.

## Routes

| Path | Handler | Description |
|------|---------|-------------|
| `/` | BuildStatus | Serves builds directly (no redirect) |
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

- **Templates use relative paths for reverse proxy compatibility**: All `href`/`src` in templates are relative (`href="builds"`, `href="static/base.css"`), not absolute (`/builds`). The root `/` serves builds directly (no redirect). This ensures the app works behind a reverse proxy at any path prefix (e.g., `sites.ecmwf.int/ecm7593/gh/`). Works because all routes are single-level siblings — the browser replaces the last path segment. **Do not change these to absolute paths.** Note: `http.Redirect` absolutizes relative URLs, so never use it for this purpose.
- **Builds view renders all configured branches**: The handler uses `repoConfig` (from `config.yaml`) to determine which branches to display per repo. Branch order matches config order. Repos not in config are appended alphabetically with branches from API data.
- **GitHub API rate limits**: 12 repos x 2 branches = frequent polling. Rate limit warnings appear in logs when remaining < 100.
- **`CONTRIBUTOR` is treated as external**: `isInternal` only matches `OWNER`, `MEMBER`, or `COLLABORATOR`. Past contributors with merged PRs who aren't collaborators get the "external" badge.
- **Templates parsed at startup**: No hot reload. Editing a `.html` file requires a restart. Templates validated as non-nil in handler constructor (panics on nil).
- **Handler constructor uses HandlerConfig struct**: `New()` in `dashboard.go` accepts a `HandlerConfig` struct with named fields for all 10 parameters. Templates and Store are nil-checked (panic on nil).
- **Handler data uses anonymous structs**: Each handler defines its template data as an inline `struct{...}` literal — no shared type. Adding/removing a template field requires updating each handler separately.
- **No inline scripts**: CSP uses `script-src 'self'` only. All JS is in external files.
- **CSP includes `form-action 'self'`**: Restricts form submissions to same-origin only.
- **TV mode framing**: `/builds-dashboard` omits `X-Frame-Options` but uses `frame-ancestors 'self'` in CSP (same-origin iframes only). All other routes use `DENY`/`frame-ancestors 'none'`.
- **DISMISSED reviews**: A DISMISSED review removes the reviewer from the review map, reverting to their previous state (or removing them entirely).
- **Healthcheck hardcodes port 8000**: `cmd/healthcheck/main.go` always hits `localhost:8000`. Changing the port in `config.yaml` without updating the healthcheck binary will cause Docker health checks to fail.
- **Fetchers use partial failure tolerance**: Per-repo errors are logged and skipped; `Merge*` preserves existing data for failed repos. An error is returned only if ALL repos fail. Stale repos are surfaced in the UI via a banner and per-row visual indicators (threshold: 3x fetch interval).
- **`go-github` aliased as `gh`**: All files in `internal/github/` import `github.com/google/go-github/v66/github` as `gh` to avoid collision with the internal `github` package. New fetch code must follow the `gh` convention.
- **Check classification via ClassifyCheck()**: Both PR and builds views use `github.ClassifyCheck(status, conclusion)` for consistent classification. Only explicit `"success"` passes; running states return `"running"`; everything else (including `neutral`, `stale`) returns `"failure"`.
- **Auto-refresh**: All views use fetch-and-replace (preserves scroll, expanded state, and keyboard focus). Non-TV mode shows a `role="alert"` error banner after 3 consecutive failures and keeps retrying. TV mode rebuilds countdown DOM and recalculates grid after each DOM swap; falls back to `window.location.reload()` after 3 consecutive failures.
- **Staleness computation centralized**: `Handler.computeStaleness()` in `staleness.go` handles cold-start detection, threshold computation, and nil-guarding. All four handlers call it with one line.
- **768px CSS breakpoint for both tables**: Both PR and issues tables switch to card layout at 768px. Issues table uses `.issues-table:not(.pr-table-wrap)` selectors in the same media query block.
- **Template FuncMap in handlers package**: `handlers.TemplateFuncs()` is the single source of truth used by both `main.go` and tests. New template functions go in `funcmap.go`. `affirmations` is unexported.
- **Builds template duplicates check classification in HTML**: `builds.html` uses inline template conditions to assign CSS classes (`status-running`, `status-failure`, `status-success`, `status-neutral`). This diverges slightly from `ClassifyCheck()`: a `neutral` conclusion gets the grey `status-neutral` CSS class but is counted as `"failure"` by `ClassifyCheck()`. This is intentional (grey dot = informational, but counted in failure totals). Changes to classification policy require updating both Go and template code.
- **No `aria-live` on `<main>`**: Removed the overbroad `aria-live="polite"` that announced all content on every refresh. The stale banner and refresh error banner use `role="alert"` for targeted announcements.
- **Nav accessibility**: `base.html` nav has `aria-label="Main navigation"` and `aria-current="page"` on the active link. All table headers use `scope="col"`.
- **Issues and PR tables share 768px card breakpoint**: Both tables switch to card layout at the same breakpoint for consistent tablet/mobile behavior.
- **`paginate()` clamps both bounds**: Guards both `page < 1` (clamps to 1) and `page > totalPages` (clamps to last). Returns 0-based start, exclusive end.
- **TV card flex layout**: Branch sections use `flex: 1 1 0` with `min-height: 0; overflow: hidden` so each branch gets equal card space. Headers (`card-header`, `branch-header`) use `flex-shrink: 0` to stay visible. Failure/running lists clip within their allotted space. Card uses `gap: 4px` (not margin-bottom) for branch spacing.
- **`color-mix()` has static fallbacks**: `base.css` and `tv.css` use `color-mix()` with a preceding static color fallback for older browsers (Safari <16.2, Chrome <111, Firefox <113).
- **PR vs builds check-dot CSS class naming inconsistency**: Builds use `.lane-dot` with `status-*` prefixed modifiers (`status-success`, etc. in `builds.css`), while PRs use `.check-dot` with bare modifiers (`success`, etc. in `base.css`). Both templates now use identical classification logic (queued/waiting/pending → running; timed_out/action_required/cancelled → failure).
- **Static assets have `Cache-Control: public, max-age=3600`**: A `cacheControl()` middleware wraps `http.FileServer` for `/static/` routes.
- **Lane-strip `max-height` clips checks with fade**: `builds.css` `.lane-strip` has `max-height: 38px; overflow: hidden`, allowing ~3 rows of dots. A gradient fade-out (`::after` pseudo-element) appears when 16+ dots are present (via `:has()` selector — requires Chrome 105+/Safari 15.4+/Firefox 121+; graceful degradation without fade on older browsers).
- **`DeriveReviewStatus()` nil-safe**: `review.go` skips nil `*Reviewer` values in the map iteration.
- **WCAG AA color tokens**: `--link-color` (`#A66214`) for links and active nav (4.5:1 on white). `--warning-text` (`#b45f00`) for running counts/text (4.5:1 on white). `--muted-text` (`#576069`, 5:1). `--warning-color` and `--accent-color` are kept for non-text use (dots, borders, backgrounds).
- **Both CI and release workflows run `govulncheck`**: Pinned to `v1.1.4` for deterministic CI. Prevents shipping with known Go vulnerabilities.

## Environment

- `GITHUB_TOKEN` (required) — GitHub personal access token for API access
- No CLI flags — all configuration is via `config.yaml` and environment variables
- `config.yaml` is gitignored — must be created locally. It IS baked into the Docker image at build time (not mounted).
- CI/CD: `.github/workflows/release.yml` runs tests, builds Docker image locally, scans with Trivy (blocking), then pushes to Harbor on GitHub release (tagged with both the release version and `latest`)
- `.github/workflows/ci.yml` runs tests with `-race`, vet, `govulncheck@v1.1.4`, and a Docker build (no push) on push/PR

## Development Status

- Test suite covers storage (deep-copy, concurrency, merge semantics), config validation (including `Load()` error paths), `isInternal()`, review status logic, label color helpers, check classification (`ClassifyCheck`), handler HTTP responses (with real templates), handler repo filtering, builds grouping/ordering, parameter sanitization (incl. injection edge cases), sort functions (all fields + both orders), and staleness computation
- **Untested**: `internal/fetcher/` (no test files — blocked on `github.Client` having no interface), `renderTemplate` error paths, constructor nil-panic paths
- Race detector enabled in CI and Makefile
- Security scanning via govulncheck (CI and release) and Trivy (release, blocking on CRITICAL/HIGH)
- All CI actions pinned to full SHA (supply chain security)
- No linter configuration (uses `go vet`)
- go-github v66 — 4 major versions behind current (v70)
