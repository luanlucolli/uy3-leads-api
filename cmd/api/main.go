package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"html"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/luanlucolli/uy3-leads-api/frontend"
	"github.com/luanlucolli/uy3-leads-api/internal/auth"
	"github.com/luanlucolli/uy3-leads-api/internal/config"
	"github.com/luanlucolli/uy3-leads-api/internal/database"
	"github.com/luanlucolli/uy3-leads-api/internal/handlers"
	"github.com/luanlucolli/uy3-leads-api/internal/middleware"
)

func main() {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	authService, err := auth.NewService(db, cfg.JWTSecret)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	router := buildRouter(db, authService, cfg.Uy3WebhookSecret)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("uy3-leads-api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server: %v", err)
	case <-shutdownCtx.Done():
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}

func buildRouter(db *sql.DB, authService *auth.Service, webhookSecret string) http.Handler {
	authHandler := handlers.NewAuthHandler(authService)
	webhookHandler := handlers.NewWebhookHandler(db)
	leadsHandler := handlers.NewLeadsHandler(db)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(3, "text/html", "text/css", "application/javascript"))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":    "error",
				"component": "database",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Post("/login", authHandler.Login)
	r.With(middleware.VerifyUy3Webhook(webhookSecret)).Post("/webhook", webhookHandler.Receive)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireJWT(authService))
		r.Get("/me", authHandler.Me)
		r.Get("/leads", leadsHandler.List)
		r.Get("/leads/export", leadsHandler.ExportCSV)
	})

	r.NotFound(spaHandler())

	return r
}

func spaHandler() http.HandlerFunc {
	dist, err := fs.Sub(frontend.Files, "dist")
	if err != nil {
		log.Printf("erro ao carregar frontend: %v", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend indisponível", http.StatusInternalServerError)
		}
	}

	fileServer := http.FileServer(http.FS(dist))

	return func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" {
			serveSPAIndex(w, dist)
			return
		}

		file, err := dist.Open(requestPath)
		if err == nil {
			defer file.Close()
			stat, statErr := file.Stat()
			if statErr == nil && !stat.IsDir() {
				if strings.HasPrefix(requestPath, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else if requestPath == "index.html" {
					serveSPAIndex(w, dist)
					return
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		serveSPAIndex(w, dist)
	}
}

func serveSPAIndex(w http.ResponseWriter, dist fs.FS) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "Frontend indisponível", http.StatusInternalServerError)
		return
	}

	companyName := html.EscapeString(strings.TrimSpace(os.Getenv("VITE_COMPANY_NAME")))
	indexHTML := strings.ReplaceAll(string(index), "__UY3_COMPANY_NAME__", companyName)

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(indexHTML))
}
