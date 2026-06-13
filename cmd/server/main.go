package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lambdawp-567/k8dclusterlife/internal/api"
	"github.com/lambdawp-567/k8dclusterlife/internal/auth"
	"github.com/lambdawp-567/k8dclusterlife/internal/cache"
	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/lambdawp-567/k8dclusterlife/internal/healing"
	"github.com/lambdawp-567/k8dclusterlife/internal/metrics"
	"github.com/lambdawp-567/k8dclusterlife/internal/notify"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := cache.New(redisAddr)
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		slog.Warn("redis not available, continuing without cache", "error", err)
	} else {
		slog.Info("redis connected", "addr", redisAddr)
	}
	defer redisClient.Close()

	// Cluster controller
	controller := cluster.NewController(redisClient)

	// Auto-add in-cluster config when running inside Kubernetes
	if os.Getenv("KUBECONFIG") == "" {
		if err := controller.AddCluster(cluster.ClusterConfig{
			ID:   "in-cluster",
			Name: "Local Cluster",
		}, nil, 30*time.Second); err != nil {
			slog.Info("no in-cluster config available (expected outside K8s)", "error", err)
		} else {
			slog.Info("monitoring in-cluster")
		}
	}

	// Healing agent (executor without a fixed cluster — clusters resolved per-session)
	healingAgent := healing.New(nil)

	// Notifier
	notifier := notify.NewNotifier()
	_ = notifier // used by event handlers in production

	// Auth handler
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	authHandler, err := auth.New(baseURL)
	if err != nil {
		slog.Error("failed to initialize auth providers", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(authHandler.Middleware)

	// Health (always public)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Prometheus metrics (always public)
	r.Handle("/metrics", metrics.Handler())

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		for _, p := range []auth.Provider{auth.ProviderEntra, auth.ProviderGitHub, auth.ProviderGoogle} {
			provider := p
			r.Get("/"+string(provider)+"/login", authHandler.HandleLogin(provider))
			r.Get("/"+string(provider)+"/callback", authHandler.HandleCallback(provider))
		}
		r.Post("/logout", authHandler.HandleLogout)
		r.Get("/logout", authHandler.HandleLogout)
	})

	// API
	r.Route("/api", func(r chi.Router) {
		r.Get("/me", authHandler.HandleMe)
		r.Get("/auth/providers", func(w http.ResponseWriter, r *http.Request) {
			providers := authHandler.EnabledProviders()
			names := make([]string, len(providers))
			for i, p := range providers {
				names[i] = string(p)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(names)
		})

		// Protected API routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth)
			r.Get("/problems", api.HandleProblems(controller))
			r.Mount("/healing", api.HandleHealing(healingAgent))
		})
	})

	// Serve frontend static files
	r.Handle("/*", http.FileServer(http.Dir("web/app/dist")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
