# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go web dashboard that monitors GitHub issues, pull requests, and CI check statuses across multiple ECMWF organization repositories. It uses background polling with configurable intervals and renders server-side HTML templates.

## Build & Run

```bash
# Requires Go 1.22+ (Dockerfile uses 1.24)
go mod tidy

# Build
go build -o ecmwf-dash cmd/server/main.go

# Run (requires GITHUB_TOKEN env var)
GITHUB_TOKEN=<token> ./ecmwf-dash

# Run with custom CSS theme
./ecmwf-dash -css dark.css
```

The server reads `config.yaml` from the working directory and expects `web/` to be present. Default listen address is `0.0.0.0:8000`.

## Docker

```bash
docker build -t ecmwf-dash .
docker run -e GITHUB_TOKEN=<token> -p 8000:8000 ecmwf-dash
```

## Architecture

**Entry point**: `cmd/server/main.go` — wires up config, GitHub client, storage, fetcher, handlers, and HTTP routes.

**Data flow**: Three background goroutines (`internal/fetcher/`) poll the GitHub API at configured intervals and write results into an in-memory store (`internal/storage/memory.go`, RWMutex-protected). HTTP handlers read from the store and render Go HTML templates.

**Key packages**:
- `internal/config/` — Loads `config.yaml` (repos, branches, fetch intervals, server settings)
- `internal/github/` — GitHub API client wrapper. `types.go` defines all data models (Issue, PullRequest, Check, BranchCheck, Reviewer). Separate files for issues, pulls, and actions fetching.
- `internal/storage/` — Thread-safe in-memory store (no database)
- `internal/handlers/` — HTTP handlers for three views: `/builds` (default `/`), `/pulls`, `/issues`
- `internal/fetcher/` — Orchestrates background polling goroutines with context cancellation

**Frontend**: Server-rendered Go templates in `web/templates/`, static assets in `web/static/`. Three CSS themes (auto/light/dark) with client-side theme switching via localStorage. Auto-refresh configurable 0-15 min. Templates are parsed once at startup — no hot reload; editing a `.html` file requires a restart.

**Template structure**: The three templates (`builds.html`, `dashboard.html`, `pullrequests.html`) are standalone full HTML documents with no shared base template or partials. Nav, theme selector, refresh dropdown, and the anti-flicker inline script are duplicated in all three files. Any shared UI change must be made in three places.

**API cost**: PR fetching is expensive — 3 API calls per open PR (list reviews, get full PR for `mergeable_state`, list check runs). This is the main rate limit pressure point across the dashboard.

## Routes

| Path | Handler | Description |
|------|---------|-------------|
| `/` | BuildStatus | Redirects to builds view |
| `/builds` | BuildStatus | CI check status per repo/branch |
| `/pulls` | PullRequests | Open PRs with reviews and checks |
| `/issues` | Dashboard | Open issues across repos |

## Configuration

`config.yaml` must exist in the working directory. Example structure:

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

- **`default.css` bug**: The `-css` flag defaults to `"default.css"` which doesn't exist — valid themes are `auto.css`, `light.css`, `dark.css`. Worse, the inline anti-flicker `<script>` in all three templates also defaults to `default.css` and runs before `dashboard.js`, so `dashboard.js`'s `auto.css` fallback is dead code — the inline script writes `default.css` to localStorage first on new browsers.
- **WorkflowRuns are dead code**: `WorkflowRun` type, `SetWorkflowRuns`/`GetWorkflowRuns` storage methods, and the fetched data are all unused. No handler reads them.
- **Builds view only renders two branches**: The handler hardcodes `main`/`master` and `develop`. Other branch names in `config.yaml` are fetched from the API but silently discarded by the builds handler.
- **No `.gitignore`** — build artifact `ecmwf-dash` will show up in `git status`.
- **GitHub API rate limits**: 12 repos × 2 branches = frequent polling. Ensure `GITHUB_TOKEN` has appropriate scopes (`repo`, `read:checks`).
- **`CONTRIBUTOR` is treated as external**: `isInternal` only matches `OWNER`, `MEMBER`, or `COLLABORATOR`. Past contributors with merged PRs who aren't collaborators get the "external" badge.

## Environment

- `GITHUB_TOKEN` (required) — GitHub personal access token for API access
- CI/CD: `.github/workflows/release.yml` builds and pushes Docker image to Harbor on GitHub release

## Known Bugs

- **Duplicated `</body></html>`** in `dashboard.html` (~line 171-172).
- **`server.Close()` instead of `server.Shutdown(ctx)`** in `main.go` — active connections are killed immediately on SIGINT instead of draining gracefully.
- **`fmt.Printf` instead of `log.Printf`** in `internal/github/actions.go` and `pulls.go` — error output has no timestamps and won't respect any future logger config.
- **Inconsistent nil-template guard** — `/builds` and `/pulls` handlers check for nil templates before executing; `/issues` handler doesn't, so a nil `issuesTmpl` would panic instead of returning 500.
- **PR check runs don't paginate** — `FetchBranchChecks` has a pagination loop but `fetchPRDetails` caps at 100 check runs with no pagination.
- **`getAvailableCSS()` reads the filesystem on every request** — `os.ReadDir("web/static")` runs per page load to populate the theme dropdown.

## Development Status

- No test suite exists
- No linter configuration exists
