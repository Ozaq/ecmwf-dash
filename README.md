# ECMWF GitHub Dashboard

A Go web dashboard that monitors GitHub issues, pull requests, and CI check statuses across multiple ECMWF organization repositories. Uses background polling with configurable intervals and renders server-side HTML templates.

## Quick Start

### Prerequisites

- Go 1.24+
- A GitHub personal access token with `repo` and `read:checks` scopes
- A `config.yaml` file in the working directory

### Run locally

```bash
# Install dependencies
go mod tidy

# Build
make build

# Run
GITHUB_TOKEN=<token> ./ecmwf-dash
```

The server starts on `0.0.0.0:8000` by default.

### Run with Docker

```bash
make docker-build
docker run -e GITHUB_TOKEN=<token> -v $(pwd)/config.yaml:/config.yaml -p 8000:8000 ecmwf-dash
```

## Configuration

Create a `config.yaml` in your working directory:

```yaml
github:
  organization: ecmwf
  repositories:
    - name: eccodes
      branches: [master, develop]
    - name: atlas
      branches: [master, develop]

fetch_intervals:
  issues: 30m
  pull_requests: 10m
  actions: 5m

server:
  host: "0.0.0.0"
  port: 8000
```

### Configuration fields

| Field | Description |
|-------|-------------|
| `github.organization` | GitHub organization to monitor |
| `github.repositories` | List of repos with branch names to track |
| `fetch_intervals.issues` | How often to poll for issues |
| `fetch_intervals.pull_requests` | How often to poll for PRs |
| `fetch_intervals.actions` | How often to poll for CI checks |
| `server.host` | Listen address |
| `server.port` | Listen port (1-65535) |

## Routes

| Path | Description |
|------|-------------|
| `/` | Redirects to `/builds` |
| `/builds` | CI check status per repo/branch |
| `/pulls` | Open PRs with reviews and checks |
| `/issues` | Open issues across repos |
| `/health` | Health check with last-fetch timestamps |
| `/static/` | Static assets (CSS, JS) |

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-css` | `auto.css` | CSS theme file (relative to `web/static/`) |

## Development

```bash
make build    # Build binary
make test     # Run tests
make vet      # Run go vet
make run      # Build and run
make clean    # Remove binary
```

## Themes

Three built-in themes: `auto.css` (follows system preference), `light.css`, `dark.css`. Theme selection is stored in localStorage and persists across sessions. Switch themes via the dropdown in the header.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | Yes | GitHub personal access token |
