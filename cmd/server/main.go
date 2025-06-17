package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ozaq/ecmwf-dash/internal/config"
	"github.com/ozaq/ecmwf-dash/internal/fetcher"
	"github.com/ozaq/ecmwf-dash/internal/github"
	"github.com/ozaq/ecmwf-dash/internal/handlers"
	"github.com/ozaq/ecmwf-dash/internal/storage"
)

func main() {
	// Parse CLI flags
	cssFile := flag.String("css", "style.css", "CSS file to use (relative to web/static/)")
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

	// Create fetcher and start background fetching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := fetcher.New(cfg, gh, store)
	f.Start(ctx)

	// Load template
	issuesTmpl := template.New("dashboard.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
	})
	issuesTmpl, err = issuesTmpl.ParseFiles("web/templates/dashboard.html")
	if err != nil {
		log.Fatal("Failed to load template:", err)
	}

	// Load template
	prsTmpl := template.New("pullrequests.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
	})
	prsTmpl, err = prsTmpl.ParseFiles("web/templates/pullrequests.html")
	if err != nil {
		log.Fatal("Failed to load template:", err)
	}


	// Create handler
	handler := handlers.New(store, issuesTmpl, prsTmpl, *cssFile)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.Dashboard)
	mux.HandleFunc("/issues", handler.Dashboard)
	mux.HandleFunc("/pulls", handler.PullRequests)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	loggedMux := logMiddleware(mux)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)

	server := &http.Server{Addr: addr, Handler: loggedMux}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		cancel()
		server.Close()
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
