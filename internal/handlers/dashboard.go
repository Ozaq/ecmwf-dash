package handlers

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
	"github.com/ozaq/ecmwf-dash/internal/storage"
)

const issuesPerPage = 100

type Handler struct {
	storage       *storage.Memory
	template      *template.Template
	prTemplate    *template.Template
	buildTemplate *template.Template
	cssFile       string
}

func New(storage *storage.Memory, issuesTmpl *template.Template, prsTmpl *template.Template, buildTmpl *template.Template, cssFile string) *Handler {
	return &Handler{
		storage:       storage,
		template:      issuesTmpl,
		prTemplate:    prsTmpl,
		buildTemplate: buildTmpl,
		cssFile:       cssFile,
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

	// Get available CSS files
	cssFiles := getAvailableCSS()

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
		Issues      []github.Issue
		LastUpdate  time.Time
		CurrentPage int
		TotalPages  int
		TotalIssues int
		Sort        string
		Order       string
		NextOrder   string
		CSSFile     string
		CSSFiles    []string
	}{
		Issues:      pageIssues,
		LastUpdate:  lastUpdate,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalIssues: len(issues),
		Sort:        sortBy,
		Order:       order,
		NextOrder:   getNextOrder(order),
		CSSFile:     h.cssFile,
		CSSFiles:    cssFiles,
	}

	if err := h.template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func getAvailableCSS() []string {
	var cssFiles []string

	files, err := os.ReadDir("web/static")
	if err != nil {
		return cssFiles
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".css") {
			cssFiles = append(cssFiles, file.Name())
		}
	}

	sort.Strings(cssFiles)
	return cssFiles
}