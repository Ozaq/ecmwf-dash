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
- `internal/github/` — GitHub API client wrapper. `types.go` defines all data models + `isInternal()`. `review.go` contains `DeriveReviewStatus()` (extracted, testable). `helpers.go` has `sanitizeLabelColor()`, `computeTextColor()`, and `computeLabelStyle()` for safe label rendering. `checks.go` has `ClassifyCheck()` — the single source of truth for check status classification. Separate files for issues, pulls, and actions fetching. Rate info returned from fetch functions (no extra API calls).
- `internal/storage/` — `Store` interface + `Memory` implementation. Thread-safe with deep-copy on both reads and writes. `LastFetchTimes()` and `RepoFetchTimes(category)` on the Store interface for health endpoint and per-repo staleness. `Merge*` methods for partial-failure updates (preserve existing data for failed repos, update succeeded repos). Category constants (`CategoryIssues`, `CategoryPRs`, `CategoryChecks`) prevent string-literal typos.
- `internal/handlers/` — HTTP handlers for three views: `/builds`, `/pulls`, `/issues`. `render.go` provides buffered template execution. `params.go` has shared input validation (`sanitizeSort`, `sanitizeOrder`, `sanitizeRepo`) and `paginate()`. `staleness.go` has `staleRepos()` helper, `computeStaleness()` method, and `FetchIntervals` type (deliberately separate from config to avoid import dependency). `funcmap.go` exports `TemplateFuncs()` and `Affirmations` — single source of truth for template functions. `HandlerConfig` struct replaces positional params in `New()`.
- `internal/fetcher/` — Orchestrates background polling goroutines with context cancellation. Uses `Store.Merge*` (not `Set*`) for partial-failure tolerance. No test coverage — depends on concrete `*github.Client` (no interface).

**Frontend**: Server-rendered Go templates in `web/templates/`, static assets in `web/static/`.

**Template structure**: Uses Go template inheritance via `base.html` which defines the shared `<head>`, nav, header, and footer. Page templates (`builds.html`, `dashboard.html`, `pullrequests.html`) define `title`, `stats`, `extra-css`, and `content` blocks. All pages use `renderTemplate()` for buffered output.

**CSS architecture**: Single ECMWF blue palette defined in `base.css` `:root` using CSS custom properties. No theme switching — one light theme only. `builds.css` holds build-page-specific styles. `tv.css` holds TV/kiosk-mode styles. `.tooltip-dot` in base.css provides shared tooltip pseudo-elements for both `.check-dot` and `.lane-dot`. PR-specific styles currently live in base.css (not yet extracted to a separate file).

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
- **Handler constructor uses HandlerConfig struct**: `New()` in `dashboard.go` accepts a `HandlerConfig` struct with named fields for all 10 parameters. Templates are nil-checked (panic on nil), but `cfg.Store` is not — a nil Store causes a nil pointer dereference on first request.
- **Handler data uses anonymous structs**: Each handler defines its template data as an inline `struct{...}` literal — no shared type. Adding/removing a template field requires updating each handler separately.
- **No inline scripts**: CSP uses `script-src 'self'` only. All JS is in external files.
- **CSP exemption for TV mode**: `/builds-dashboard` omits `X-Frame-Options` and `frame-ancestors` to allow iframe embedding in kiosk/TV dashboarding tools. All other routes use `DENY`/`'none'`.
- **DISMISSED reviews**: A DISMISSED review removes the reviewer from the review map, reverting to their previous state (or removing them entirely).
- **Healthcheck hardcodes port 8000**: `cmd/healthcheck/main.go` always hits `localhost:8000`. Changing the port in `config.yaml` without updating the healthcheck binary will cause Docker health checks to fail.
- **Fetchers use partial failure tolerance**: Per-repo errors are logged and skipped; `Merge*` preserves existing data for failed repos. An error is returned only if ALL repos fail. Stale repos are surfaced in the UI via a banner and per-row visual indicators (threshold: 3x fetch interval).
- **`go-github` aliased as `gh`**: All files in `internal/github/` import `github.com/google/go-github/v66/github` as `gh` to avoid collision with the internal `github` package. New fetch code must follow the `gh` convention.
- **Check classification via ClassifyCheck()**: Both PR and builds views use `github.ClassifyCheck(status, conclusion)` for consistent classification. Only explicit `"success"` passes; running states return `"running"`; everything else (including `neutral`, `stale`) returns `"failure"`.
- **Auto-refresh**: All views use fetch-and-replace (preserves scroll/expanded state). TV mode falls back to `window.location.reload()` only on fetch failure. **Known bug**: TV countdown DOM references (`countdownEl`, `numSpan`) become stale after the first fetch-and-replace because `refreshDashboard()` replaces the `.tv-container` contents without re-acquiring references. The countdown display freezes after the first refresh (the refresh loop itself still works). Similarly, the grid calculation uses `n` captured at startup and is not re-run after refresh.
- **Staleness computation centralized**: `Handler.computeStaleness()` in `staleness.go` handles cold-start detection, threshold computation, and nil-guarding. All four handlers call it with one line.
- **480px CSS breakpoint scoped to issues only**: The `.issues-table` selectors at 480px use `:not(.pr-table-wrap)` to avoid overriding the 768px PR card layout.
- **Template FuncMap in handlers package**: `handlers.TemplateFuncs()` is the single source of truth used by both `main.go` and tests. New template functions go in `funcmap.go`. Note: `Affirmations` is exported and mutable — should be unexported.
- **Builds template duplicates check classification in HTML**: `builds.html` uses inline template conditions to assign CSS classes (`status-running`, `status-failure`, `status-success`, `status-neutral`). This diverges slightly from `ClassifyCheck()`: a `neutral` conclusion gets the grey `status-neutral` CSS class but is counted as `"failure"` by `ClassifyCheck()`. This is intentional (grey dot = informational, but counted in failure totals). Changes to classification policy require updating both Go and template code.
- **`aria-live="polite"` on `<main>` is overbroad**: The current placement announces all content on every 60-second refresh, which is excessive for screen readers. Should be scoped to a smaller element.
- **Issues table has no responsive treatment between 480-768px**: PR table has full card layout at 768px, but issues table only forces horizontal scroll at 768px with card layout at 480px. Tablets get a scrolling table with no cue.
- **`paginate()` does not guard `page < 1`**: Callers are responsible for clamping page to >= 1 before calling `paginate()`. The function guards `page > totalPages` but not the lower bound.
- **Dead `.pagination a.active` CSS rule**: `base.css:283-287` defines styles for `.pagination a.active`, but no template ever emits an `<a class="active">` inside `.pagination`. Current page is shown as `<span class="pagination-info">`. The rule is dead code.
- **`color-mix()` used without fallback**: `base.css` (stale-banner) and `tv.css` (failure-item, running-item, card shadow) use `color-mix()` which requires Safari 16.2+, Chrome 111+, Firefox 113+. On older browsers the declaration is silently ignored — cosmetic degradation only.
- **PR vs builds check-dot CSS class naming inconsistency**: Builds use `.lane-dot` with `status-*` prefixed modifiers (`status-success`, etc. in `builds.css`), while PRs use `.check-dot` with bare modifiers (`success`, etc. in `base.css`). The two templates also diverge in classification branching — `builds.html` catches more conclusion types explicitly than `pullrequests.html`.
- **No `Cache-Control` on static assets**: `http.FileServer` in `main.go:106` is used bare — no middleware adds `Cache-Control` headers. CSS/JS assets rely on heuristic browser caching and `Last-Modified` conditional requests only.
- **Lane-strip `max-height` clips checks silently**: `builds.css` `.lane-strip` has `max-height: 26px; overflow: hidden`, allowing ~2 rows of dots. Branches with >22-24 checks have excess dots silently hidden — no "+N more" indicator. The `lane-counts` span only shows failure/running counts, so overflow of passing checks is invisible.
- **`DeriveReviewStatus()` nil map value risk**: `review.go` iterates `map[string]*Reviewer` and dereferences each value without nil-guarding (`reviewer.State` at line 12). Callers (`pulls.go`) always insert non-nil `&Reviewer{}`, but the function contract doesn't enforce it — a nil map value causes a panic.
- **Active nav link contrast fails WCAG AA**: `.nav-links a.active` (`base.css:85-89`) uses `--accent-color` (#C87519) on white (`--card-bg`), producing ~3.49:1 contrast ratio. WCAG AA for normal text (14px/500 weight) requires 4.5:1 — this fails. Passes large-text threshold (3.0:1) but the nav links are normal-sized text.
- **Release workflow missing `govulncheck`**: CI (`ci.yml`) runs `govulncheck` but the release workflow (`release.yml`) does not — releases can ship with known Go vulnerabilities that CI would have caught on push/PR.

## Environment

- `GITHUB_TOKEN` (required) — GitHub personal access token for API access
- No CLI flags — all configuration is via `config.yaml` and environment variables
- `config.yaml` is gitignored — must be created locally or mounted at runtime
- CI/CD: `.github/workflows/release.yml` runs tests then builds and pushes Docker image to Harbor on GitHub release (tagged with both the release version and `latest`)
- `.github/workflows/ci.yml` runs tests with `-race`, vet, `govulncheck`, and a Docker build (no push) on push/PR

## Development Status

- Test suite covers storage (deep-copy, concurrency, merge semantics), config validation, review status logic, label color helpers, check classification (`ClassifyCheck`), handler HTTP responses (with real templates), builds grouping/ordering, parameter sanitization (incl. injection edge cases), sort functions (all fields + both orders), and staleness computation
- **Untested**: `internal/fetcher/` (no test files — blocked on `github.Client` having no interface), `renderTemplate` error paths, constructor nil-panic paths, `TemplateFuncs()` (exercised indirectly but no dedicated test), `computeStaleness()` method (underlying `staleRepos()` is tested but the method itself is not)
- Race detector enabled in CI and Makefile
- Security scanning via govulncheck (CI only — not in release workflow) and Trivy (release)
- Trivy scan is informational only (`exit-code: 0`) — won't block releases on vulnerabilities
- All CI actions pinned to full SHA (supply chain security)
- No linter configuration (uses `go vet`)
- go-github v66 — 4 major versions behind current (v70)
