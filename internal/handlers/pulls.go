package handlers

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

func (h *Handler) PullRequests(w http.ResponseWriter, r *http.Request) {
	prs, lastUpdate := h.storage.GetPullRequests()
	log.Printf("Serving /pulls - PRs: %d", len(prs))

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
		filtered := prs[:0]
		for _, pr := range prs {
			if pr.Repository == repo {
				filtered = append(filtered, pr)
			}
		}
		prs = filtered
	}

	// Sort PRs
	sortPullRequests(prs, sortBy, order)

	// Paginate
	start, end, totalPages := paginate(len(prs), page, itemsPerPage)
	var pagePRs []github.PullRequest
	if start < len(prs) {
		pagePRs = prs[start:end]
	}

	// Compute staleness (skip on cold start)
	var staleMap map[string]bool
	var staleList []string
	if !lastUpdate.IsZero() {
		repoTimes := h.storage.RepoFetchTimes("prs")
		threshold := h.fetchIntervals.PullRequests * 3
		staleMap = staleRepos(repoTimes, threshold, h.repoNames)
		staleList = sortedKeys(staleMap)
	}
	if staleMap == nil {
		staleMap = make(map[string]bool)
	}

	data := struct {
		PageID        string
		Organization  string
		Version       string
		PullRequests  []github.PullRequest
		LastUpdate    time.Time
		CurrentPage   int
		TotalPages    int
		TotalPRs      int
		Sort          string
		Order         string
		NextOrder     string
		Repo          string
		RepoNames     []string
		StaleRepos    map[string]bool
		StaleRepoList []string
	}{
		PageID:        "pulls",
		Organization:  h.organization,
		Version:       h.version,
		PullRequests:  pagePRs,
		LastUpdate:    lastUpdate,
		CurrentPage:   page,
		TotalPages:    totalPages,
		TotalPRs:      len(prs),
		Sort:          sortBy,
		Order:         order,
		NextOrder:     getNextOrder(order),
		Repo:          repo,
		RepoNames:     h.repoNames,
		StaleRepos:    staleMap,
		StaleRepoList: staleList,
	}

	renderTemplate(w, h.prTemplate, "base", data)
}

func sortPullRequests(prs []github.PullRequest, sortBy, order string) {
	switch sortBy {
	case "repo":
		sort.Slice(prs, func(i, j int) bool {
			if order == "asc" {
				return prs[i].Repository < prs[j].Repository
			}
			return prs[i].Repository > prs[j].Repository
		})
	case "number":
		sort.Slice(prs, func(i, j int) bool {
			if order == "asc" {
				return prs[i].Number < prs[j].Number
			}
			return prs[i].Number > prs[j].Number
		})
	case "title":
		sort.Slice(prs, func(i, j int) bool {
			if order == "asc" {
				return prs[i].Title < prs[j].Title
			}
			return prs[i].Title > prs[j].Title
		})
	case "author":
		sort.Slice(prs, func(i, j int) bool {
			if order == "asc" {
				return prs[i].Author < prs[j].Author
			}
			return prs[i].Author > prs[j].Author
		})
	case "created":
		sort.Slice(prs, func(i, j int) bool {
			if order == "asc" {
				return prs[i].CreatedAt.Before(prs[j].CreatedAt)
			}
			return prs[i].CreatedAt.After(prs[j].CreatedAt)
		})
	case "updated":
		sort.Slice(prs, func(i, j int) bool {
			if order == "asc" {
				return prs[i].UpdatedAt.Before(prs[j].UpdatedAt)
			}
			return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
		})
	}
}
