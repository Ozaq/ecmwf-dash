package main

import (
    "context"
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
    tmpl, err := template.ParseFiles("web/templates/dashboard.html")
    if err != nil {
        log.Fatal("Failed to load template:", err)
    }

    // Create handler
    handler := handlers.New(store, tmpl)

    // Setup routes
    http.HandleFunc("/", handler.Dashboard)

    // Start server
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Printf("Starting server on %s", addr)
    
    server := &http.Server{Addr: addr}
    
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
