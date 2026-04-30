package main

import (
	"context"
	"database/sql"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database ping: %v", err)
	}

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

	go func() {
		log.Printf("uy3-leads-api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	waitForShutdown(server)
}

func buildRouter(db *sql.DB, authService *auth.Service, webhookSecret string) http.Handler {
	authHandler := handlers.NewAuthHandler(authService)
	webhookHandler := handlers.NewWebhookHandler(db)
	leadsHandler := handlers.NewLeadsHandler(db)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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
		log.Fatalf("frontend dist: %v", err)
	}

	fileServer := http.FileServer(http.FS(dist))

	return func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		file, err := dist.Open(requestPath)
		if err == nil {
			defer file.Close()
			stat, statErr := file.Stat()
			if statErr == nil && !stat.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "frontend unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	}
}

func waitForShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}
