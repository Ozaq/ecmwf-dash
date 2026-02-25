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

// CSSOption represents a theme file entry for the dropdown.
type CSSOption struct {
	Value       string // filename, e.g. "auto.css"
	DisplayName string // display text, e.g. "Auto"
}

const issuesPerPage = 100

type Handler struct {
	storage       storage.Store
	template      *template.Template
	prTemplate    *template.Template
	buildTemplate *template.Template
	cssFile       string
	cssFiles      []CSSOption
	organization  string
	version       string
}

func New(store storage.Store, issuesTmpl *template.Template, prsTmpl *template.Template, buildTmpl *template.Template, cssFile string, staticDir string, org string, version string) *Handler {
	if issuesTmpl == nil {
		panic("issuesTmpl must not be nil")
	}
	if prsTmpl == nil {
		panic("prsTmpl must not be nil")
	}
	if buildTmpl == nil {
		panic("buildTmpl must not be nil")
	}
	return &Handler{
		storage:       store,
		template:      issuesTmpl,
		prTemplate:    prsTmpl,
		buildTemplate: buildTmpl,
		cssFile:       cssFile,
		cssFiles:      themeFiles(),
		organization:  org,
		version:       version,
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

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "updated"
	}
	order := r.URL.Query().Get("order")
	if order == "" {
		order = "desc"
	}

	// Sort issues
	sortIssues(issues, sortBy, order)

	// Paginate
	totalPages := (len(issues) + issuesPerPage - 1) / issuesPerPage
	start := (page - 1) * issuesPerPage
	end := start + issuesPerPage
	if end > len(issues) {
		end = len(issues)
	}

	var pageIssues []github.Issue
	if start < len(issues) {
		pageIssues = issues[start:end]
	}

	data := struct {
		PageID       string
		Organization string
		Version      string
		Issues       []github.Issue
		LastUpdate   time.Time
		CurrentPage  int
		TotalPages   int
		TotalIssues  int
		Sort         string
		Order        string
		NextOrder    string
		CSSFile      string
		CSSFiles     []CSSOption
	}{
		PageID:       "issues",
		Organization: h.organization,
		Version:      h.version,
		Issues:       pageIssues,
		LastUpdate:   lastUpdate,
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalIssues:  len(issues),
		Sort:         sortBy,
		Order:        order,
		NextOrder:    getNextOrder(order),
		CSSFile:      h.cssFile,
		CSSFiles:     h.cssFiles,
	}

	renderTemplate(w, h.template, "base", data)
}

func sortIssues(issues []github.Issue, sortBy, order string) {
	switch sortBy {
	case "repo":
		sort.Slice(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Repository < issues[j].Repository
			}
			return issues[i].Repository > issues[j].Repository
		})
	case "number":
		sort.Slice(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Number < issues[j].Number
			}
			return issues[i].Number > issues[j].Number
		})
	case "title":
		sort.Slice(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Title < issues[j].Title
			}
			return issues[i].Title > issues[j].Title
		})
	case "author":
		sort.Slice(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].Author < issues[j].Author
			}
			return issues[i].Author > issues[j].Author
		})
	case "created":
		sort.Slice(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].CreatedAt.Before(issues[j].CreatedAt)
			}
			return issues[i].CreatedAt.After(issues[j].CreatedAt)
		})
	default: // "updated"
		sort.Slice(issues, func(i, j int) bool {
			if order == "asc" {
				return issues[i].UpdatedAt.Before(issues[j].UpdatedAt)
			}
			return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
		})
	}
}

func getNextOrder(current string) string {
	if current == "asc" {
		return "desc"
	}
	return "asc"
}

// themeFiles returns the allowlisted theme CSS options.
func themeFiles() []CSSOption {
	return []CSSOption{
		{Value: "auto.css", DisplayName: "Auto"},
		{Value: "light.css", DisplayName: "Light"},
		{Value: "dark.css", DisplayName: "Dark"},
	}
}
