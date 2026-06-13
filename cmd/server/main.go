package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lambdawp-567/k8dclusterlife/internal/api"
	"github.com/lambdawp-567/k8dclusterlife/internal/cache"
	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
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
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		// strip redis:// prefix
		if len(redisURL) > 8 {
			redisAddr = redisURL[8:]
		}
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

	// Auto-add cluster from in-cluster config if no kubeconfig provided
	inClusterKubeconfig := os.Getenv("KUBECONFIG")
	if inClusterKubeconfig == "" {
		// Try in-cluster, ignore errors (may not be in k8s)
		if err := controller.AddCluster(cluster.ClusterConfig{
			ID:   "in-cluster",
			Name: "Local Cluster",
		}, nil, 30*time.Second); err != nil {
			slog.Info("no in-cluster config available (expected outside K8s)", "error", err)
		} else {
			slog.Info("monitoring in-cluster")
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Health
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// API
	r.Route("/api", func(r chi.Router) {
		r.Get("/problems", api.HandleProblems(controller))
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
