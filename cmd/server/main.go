package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/config"
	"github.com/ozaq/ecmwf-dash/internal/fetcher"
	"github.com/ozaq/ecmwf-dash/internal/github"
	"github.com/ozaq/ecmwf-dash/internal/handlers"
	"github.com/ozaq/ecmwf-dash/internal/storage"
)

var Version = "dev"

var affirmations = []string{
	"All clear!",
	"Ship it!",
	"Nailed it!",
	"All green!",
	"Smooth sailing!",
	"Looking good!",
	"Rock solid!",
	"On point!",
	"Crushing it!",
	"Zero issues!",
	"Clean sweep!",
	"Top notch!",
	"Flawless!",
	"All systems go!",
	"Nice work!",
	"Spot on!",
	"Well done!",
	"No worries!",
	"Locked in!",
	"All good!",
}

var templateFuncs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"mul": func(a, b int) int { return a * b },
	"affirm": func() string {
		return affirmations[rand.IntN(len(affirmations))]
	},
}

func main() {
	// Parse CLI flags
	cssFile := flag.String("css", "auto.css", "CSS file to use (relative to web/static/)")
	flag.Parse()

	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create GitHub client
	gh, err := github.NewClient()
	if err != nil {
		log.Fatal("Failed to create GitHub client:", err)
	}

	// Create storage
	store := storage.New()

	// Register signals before starting goroutines
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create fetcher and start background fetching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := fetcher.New(cfg, gh, store)
	f.Start(ctx)

	// Load templates: each page template is parsed together with the base template
	basePath := "web/templates/base.html"

	issuesTmpl, err := template.New("base.html").Funcs(templateFuncs).ParseFiles(basePath, "web/templates/dashboard.html")
	if err != nil {
		log.Fatal("Failed to load dashboard template:", err)
	}

	prsTmpl, err := template.New("base.html").Funcs(templateFuncs).ParseFiles(basePath, "web/templates/pullrequests.html")
	if err != nil {
		log.Fatal("Failed to load pull requests template:", err)
	}

	buildsTmpl, err := template.New("base.html").Funcs(templateFuncs).ParseFiles(basePath, "web/templates/builds.html")
	if err != nil {
		log.Fatal("Failed to load builds template:", err)
	}

	dashboardTmpl, err := template.New("builds_dashboard.html").Funcs(templateFuncs).ParseFiles("web/templates/builds_dashboard.html", "web/templates/builds.html")
	if err != nil {
		log.Fatal("Failed to load builds dashboard template:", err)
	}

	// Validate -css flag against theme allowlist
	validThemes := map[string]bool{"auto.css": true, "light.css": true, "dark.css": true}
	if !validThemes[*cssFile] {
		log.Fatalf("Invalid -css value %q: must be one of auto.css, light.css, dark.css", *cssFile)
	}

	// Extract configured repo names for the builds dashboard
	repoNames := make([]string, len(cfg.GitHub.Repositories))
	for i, repo := range cfg.GitHub.Repositories {
		repoNames[i] = repo.Name
	}

	// Create handler with cached CSS list
	handler := handlers.New(store, issuesTmpl, prsTmpl, buildsTmpl, dashboardTmpl, *cssFile, "web/static", cfg.GitHub.Organization, Version, repoNames)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/builds", handler.BuildStatus)
	mux.HandleFunc("/builds-dashboard", handler.BuildsDashboard)
	mux.HandleFunc("/pulls", handler.PullRequests)
	mux.HandleFunc("/issues", handler.Dashboard)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		issueTs, prTs, checksTs := store.LastFetchTimes()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"last_fetch": map[string]any{
				"issues": issueTs,
				"pulls":  prTs,
				"checks": checksTs,
			},
		}); err != nil {
			log.Printf("Error encoding health response: %v", err)
		}
	})

	// Root redirects to /builds; unmatched paths get 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/builds", http.StatusFound)
	})

	wrapped := securityHeaders(logMiddleware(mux))

	// Start server with timeouts (HS1)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      wrapped,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown; second signal force-quits
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		go func() { <-sigChan; log.Println("Forced shutdown"); os.Exit(1) }()
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'sha256-0jlVAe+b64UKdjnXkkbAXXq5QaZvm8bamP8+r3PxgCY='; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' https://avatars.githubusercontent.com; "+
				"base-uri 'self'; "+
				"object-src 'none'; "+
				"frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
