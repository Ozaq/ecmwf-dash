package handlers

import (
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
	"github.com/ozaq/ecmwf-dash/internal/storage"
)

type Handler struct {
	storage           storage.Store
	template          *template.Template
	prTemplate        *template.Template
	buildTemplate     *template.Template
	dashboardTemplate *template.Template
	organization      string
	version           string
	repoNames         []string
	repoConfig        []RepoBranches
	fetchIntervals    FetchIntervals
}

// HandlerConfig groups the parameters needed to construct a Handler.
type HandlerConfig struct {
	Store          storage.Store
	IssuesTmpl     *template.Template
	PRsTmpl        *template.Template
	BuildTmpl      *template.Template
	DashboardTmpl  *template.Template
	Organization   string
	Version        string
	RepoNames      []string
	RepoConfig     []RepoBranches
	FetchIntervals FetchIntervals
}

func New(cfg HandlerConfig) *Handler {
	if cfg.Store == nil {
		panic("Store must not be nil")
	}
	if cfg.IssuesTmpl == nil {
		panic("IssuesTmpl must not be nil")
	}
	if cfg.PRsTmpl == nil {
		panic("PRsTmpl must not be nil")
	}
	if cfg.BuildTmpl == nil {
		panic("BuildTmpl must not be nil")
	}
	if cfg.DashboardTmpl == nil {
		panic("DashboardTmpl must not be nil")
	}
	return &Handler{
		storage:           cfg.Store,
		template:          cfg.IssuesTmpl,
		prTemplate:        cfg.PRsTmpl,
		buildTemplate:     cfg.BuildTmpl,
		dashboardTemplate: cfg.DashboardTmpl,
		organization:      cfg.Organization,
		version:           cfg.Version,
		repoNames:         cfg.RepoNames,
		repoConfig:        cfg.RepoConfig,
		fetchIntervals:    cfg.FetchIntervals,
	}
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	issues, lastUpdate := h.storage.GetIssues()
	log.Printf("Serving /issues - Issues: %d", len(issues))

	// Get query params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	sortBy := sanitizeSort(r.URL.Query().Get("sort"))
	order := sanitizeOrder(r.URL.Query().Get("order"))
	repo := sanitizeRepo(r.URL.Query().Get("repo"), h.repoNames)

	// Filter by repo
	if repo != "" {
		filtered := issues[:0]
		for _, issue := range issues {
			if issue.Repository == repo {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}

	// Sort issues
	sortIssues(issues, sortBy, order)

	// Paginate
	start, end, totalPages := paginate(len(issues), page, itemsPerPage)
	var pageIssues []github.Issue
	if start < len(issues) {
		pageIssues = issues[start:end]
	}

	staleMap, staleList := h.computeStaleness(storage.CategoryIssues, h.fetchIntervals.Issues, lastUpdate)

	data := struct {
		PageID        string
		Organization  string
		Version       string
		Issues        []github.Issue
		LastUpdate    time.Time
		CurrentPage   int
		TotalPages    int
		TotalIssues   int
		Sort          string
		Order         string
		NextOrder     string
		Repo          string
		RepoNames     []string
		StaleRepos    map[string]bool
		StaleRepoList []string
	}{
		PageID:        "issues",
		Organization:  h.organization,
		Version:       h.version,
		Issues:        pageIssues,
		LastUpdate:    lastUpdate,
		CurrentPage:   page,
		TotalPages:    totalPages,
		TotalIssues:   len(issues),
		Sort:          sortBy,
		Order:         order,
		NextOrder:     getNextOrder(order),
		Repo:          repo,
		RepoNames:     h.repoNames,
		StaleRepos:    staleMap,
		StaleRepoList: staleList,
	}

	renderTemplate(w, h.template, "base", data)
}

func sortIssues(issues []github.Issue, sortBy, order string) {
	switch sortBy {
	case "repo":
		sort.SliceStable(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Repository < issues[j].Repository
			}
			return issues[i].Repository > issues[j].Repository
		})
	case "number":
		sort.SliceStable(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Number < issues[j].Number
			}
			return issues[i].Number > issues[j].Number
		})
	case "title":
		sort.SliceStable(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Title < issues[j].Title
			}
			return issues[i].Title > issues[j].Title
		})
	case "author":
		sort.SliceStable(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Author < issues[j].Author
			}
			return issues[i].Author > issues[j].Author
		})
	case "created":
		sort.SliceStable(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].CreatedAt.Before(issues[j].CreatedAt)
			}
			return issues[i].CreatedAt.After(issues[j].CreatedAt)
		})
	case "updated":
		sort.SliceStable(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].UpdatedAt.Before(issues[j].UpdatedAt)
			}
			return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
		})
	}
}
