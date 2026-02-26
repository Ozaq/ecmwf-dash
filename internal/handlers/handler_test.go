package handlers

import (
	"html/template"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
	"github.com/ozaq/ecmwf-dash/internal/storage"
)

// templateDir returns the absolute path to web/templates/ relative to this test file.
func templateDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "web", "templates")
}

// testAffirmations is a subset of the list in cmd/server/main.go.
var testAffirmations = []string{"All clear!", "Ship it!", "Nailed it!"}

// testFuncs duplicates the template FuncMap from cmd/server/main.go.
// Duplicated here to avoid exporting it from the production binary.
var testFuncs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"mul": func(a, b int) int { return a * b },
	"affirm": func() string {
		return testAffirmations[rand.IntN(len(testAffirmations))]
	},
}

// newTestHandler creates a Handler wired to real templates and an empty store.
func newTestHandler(t *testing.T) (*Handler, *storage.Memory) {
	t.Helper()
	dir := templateDir()
	basePath := filepath.Join(dir, "base.html")

	issuesTmpl, err := template.New("base.html").Funcs(testFuncs).ParseFiles(basePath, filepath.Join(dir, "dashboard.html"))
	if err != nil {
		t.Fatalf("parse issues template: %v", err)
	}
	prsTmpl, err := template.New("base.html").Funcs(testFuncs).ParseFiles(basePath, filepath.Join(dir, "pullrequests.html"))
	if err != nil {
		t.Fatalf("parse prs template: %v", err)
	}
	buildsTmpl, err := template.New("base.html").Funcs(testFuncs).ParseFiles(basePath, filepath.Join(dir, "builds.html"))
	if err != nil {
		t.Fatalf("parse builds template: %v", err)
	}
	dashboardTmpl, err := template.New("builds_dashboard.html").Funcs(testFuncs).ParseFiles(
		filepath.Join(dir, "builds_dashboard.html"),
		filepath.Join(dir, "builds.html"),
	)
	if err != nil {
		t.Fatalf("parse dashboard template: %v", err)
	}

	store := storage.New()
	repoNames := []string{"eccodes", "atlas"}
	repoConfig := []RepoBranches{
		{Name: "eccodes", Branches: []string{"master", "develop"}},
		{Name: "atlas", Branches: []string{"main", "develop"}},
	}
	intervals := FetchIntervals{
		Issues:       5 * time.Minute,
		PullRequests: 5 * time.Minute,
		Actions:      5 * time.Minute,
	}
	h := New(HandlerConfig{
		Store:          store,
		IssuesTmpl:     issuesTmpl,
		PRsTmpl:        prsTmpl,
		BuildTmpl:      buildsTmpl,
		DashboardTmpl:  dashboardTmpl,
		Organization:   "ecmwf",
		Version:        "test",
		RepoNames:      repoNames,
		RepoConfig:     repoConfig,
		FetchIntervals: intervals,
	})
	return h, store
}

func TestDashboardHandler(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues", nil)

		h.Dashboard(rec, req)

		assertResponse(t, rec, http.StatusOK, "No open issues found")
	})

	t.Run("with_data", func(t *testing.T) {
		h, store := newTestHandler(t)
		store.SetIssues([]github.Issue{
			{Repository: "eccodes", Number: 42, Title: "Fix grib decoder", Author: "alice", URL: "https://github.com/ecmwf/eccodes/issues/42", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{Repository: "atlas", Number: 7, Title: "Add mesh support", Author: "bob", URL: "https://github.com/ecmwf/atlas/issues/7", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{Repository: "eccodes", Number: 43, Title: "Label test", Author: "charlie", URL: "https://github.com/ecmwf/eccodes/issues/43", CreatedAt: time.Now(), UpdatedAt: time.Now(), Labels: []github.Label{{Name: "bug", Color: "d73a4a"}}},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues", nil)

		h.Dashboard(rec, req)

		assertResponse(t, rec, http.StatusOK, "Fix grib decoder", "Add mesh support", "eccodes", "atlas")
	})

	t.Run("query_params", func(t *testing.T) {
		h, store := newTestHandler(t)
		store.SetIssues([]github.Issue{
			{Repository: "eccodes", Number: 1, Title: "A", Author: "a", URL: "#", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues?sort=repo&order=asc&page=1", nil)

		h.Dashboard(rec, req)

		assertResponse(t, rec, http.StatusOK, "eccodes")
	})
}

func TestPullRequestsHandler(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pulls", nil)

		h.PullRequests(rec, req)

		assertResponse(t, rec, http.StatusOK, "No open pull requests found")
	})

	t.Run("with_data", func(t *testing.T) {
		h, store := newTestHandler(t)
		store.SetPullRequests([]github.PullRequest{
			{
				Repository:   "eccodes",
				Number:       100,
				Title:        "Refactor decoder",
				Author:       "alice",
				URL:          "https://github.com/ecmwf/eccodes/pull/100",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				BaseBranch:   "develop",
				HeadBranch:   "feature-x",
				ReviewStatus: "pending",
			},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pulls", nil)

		h.PullRequests(rec, req)

		assertResponse(t, rec, http.StatusOK, "Refactor decoder", "eccodes")
	})
}

func TestBuildStatusHandler(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/builds", nil)

		h.BuildStatus(rec, req)

		assertResponse(t, rec, http.StatusOK, "No build status data available")
	})

	t.Run("with_data", func(t *testing.T) {
		h, store := newTestHandler(t)
		store.SetBranchChecks([]github.BranchCheck{
			{
				Repository: "eccodes",
				Branch:     "master",
				CommitSHA:  "abc123",
				Checks:     []github.Check{{Name: "ci", Status: "completed", Conclusion: "success", URL: "#"}},
			},
			{
				Repository: "eccodes",
				Branch:     "develop",
				CommitSHA:  "def456",
				Checks:     []github.Check{{Name: "ci", Status: "completed", Conclusion: "failure", URL: "#"}},
			},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/builds", nil)

		h.BuildStatus(rec, req)

		assertResponse(t, rec, http.StatusOK, "eccodes")
	})
}

func TestBuildsDashboardHandler(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/builds-dashboard", nil)

		h.BuildsDashboard(rec, req)

		// Even empty, the dashboard still renders configured repos.
		assertResponse(t, rec, http.StatusOK, "eccodes", "atlas")
	})

	t.Run("with_data", func(t *testing.T) {
		h, store := newTestHandler(t)
		store.SetBranchChecks([]github.BranchCheck{
			{
				Repository: "eccodes",
				Branch:     "master",
				CommitSHA:  "abc123",
				Checks:     []github.Check{{Name: "ci", Status: "completed", Conclusion: "success", URL: "#"}},
			},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/builds-dashboard", nil)

		h.BuildsDashboard(rec, req)

		assertResponse(t, rec, http.StatusOK, "eccodes")
	})
}

func TestDashboardHandlerStaleness(t *testing.T) {
	t.Run("stale_repo_shows_banner", func(t *testing.T) {
		h, store := newTestHandler(t)

		// Merge issues with "atlas" failed — it gets no per-repo timestamp.
		store.MergeIssues(
			[]github.Issue{
				{Repository: "eccodes", Number: 1, Title: "Test", Author: "a", URL: "#", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			[]string{"atlas"},   // failedRepos
			[]string{"eccodes"}, // succeededRepos
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues", nil)
		h.Dashboard(rec, req)

		assertResponse(t, rec, http.StatusOK, "stale-banner", "atlas")
	})

	t.Run("no_stale_repos_no_banner", func(t *testing.T) {
		h, store := newTestHandler(t)

		// Merge issues with both repos succeeded — both get timestamps.
		store.MergeIssues(
			[]github.Issue{
				{Repository: "eccodes", Number: 1, Title: "Test", Author: "a", URL: "#", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{Repository: "atlas", Number: 2, Title: "Test2", Author: "b", URL: "#", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			nil,                          // failedRepos
			[]string{"eccodes", "atlas"}, // succeededRepos
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues", nil)
		h.Dashboard(rec, req)

		body := rec.Body.String()
		if strings.Contains(body, "stale-banner") {
			t.Error("body should not contain stale-banner when no repos are stale")
		}
		assertResponse(t, rec, http.StatusOK, "eccodes", "atlas")
	})

	t.Run("stale_row_on_issue", func(t *testing.T) {
		h, store := newTestHandler(t)

		store.MergeIssues(
			[]github.Issue{
				{Repository: "eccodes", Number: 1, Title: "Fresh", Author: "a", URL: "#", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{Repository: "atlas", Number: 2, Title: "Stale", Author: "b", URL: "#", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			[]string{"atlas"},
			[]string{"eccodes"},
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues", nil)
		h.Dashboard(rec, req)

		assertResponse(t, rec, http.StatusOK, "stale-row")
	})
}

func TestPullRequestsHandlerStaleness(t *testing.T) {
	t.Run("stale_repo_shows_banner", func(t *testing.T) {
		h, store := newTestHandler(t)

		store.MergePullRequests(
			[]github.PullRequest{
				{Repository: "eccodes", Number: 1, Title: "PR", Author: "a", URL: "#",
					CreatedAt: time.Now(), UpdatedAt: time.Now(), BaseBranch: "develop", HeadBranch: "feat", ReviewStatus: "pending"},
			},
			[]string{"atlas"},
			[]string{"eccodes"},
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pulls", nil)
		h.PullRequests(rec, req)

		assertResponse(t, rec, http.StatusOK, "stale-banner", "atlas")
	})
}

func TestBuildStatusHandlerStaleness(t *testing.T) {
	t.Run("stale_repo_shows_banner", func(t *testing.T) {
		h, store := newTestHandler(t)

		store.MergeBranchChecks(
			[]github.BranchCheck{
				{Repository: "eccodes", Branch: "master", CommitSHA: "abc",
					Checks: []github.Check{{Name: "ci", Status: "completed", Conclusion: "success", URL: "#"}}},
			},
			[]string{"atlas"},
			[]string{"eccodes"},
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/builds", nil)
		h.BuildStatus(rec, req)

		assertResponse(t, rec, http.StatusOK, "stale-banner", "atlas")
	})

	t.Run("stale_row_on_build", func(t *testing.T) {
		h, store := newTestHandler(t)

		// Single merge: atlas data is passed but atlas is in failedRepos.
		// Atlas gets no per-repo timestamp → stale, but its data renders a row.
		store.MergeBranchChecks(
			[]github.BranchCheck{
				{Repository: "eccodes", Branch: "master", CommitSHA: "abc",
					Checks: []github.Check{{Name: "ci", Status: "completed", Conclusion: "success", URL: "#"}}},
				{Repository: "atlas", Branch: "main", CommitSHA: "def",
					Checks: []github.Check{{Name: "ci", Status: "completed", Conclusion: "success", URL: "#"}}},
			},
			[]string{"atlas"},
			[]string{"eccodes"},
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/builds", nil)
		h.BuildStatus(rec, req)

		assertResponse(t, rec, http.StatusOK, "stale-row")
	})
}

// assertResponse checks status code, content type, and that the body contains all expected strings.
func assertResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantContains ...string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Errorf("status = %d, want %d", rec.Code, wantStatus)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}

	body := rec.Body.String()
	for _, s := range wantContains {
		if !strings.Contains(body, s) {
			t.Errorf("body missing %q (len=%d)", s, len(body))
		}
	}
}
