package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	"github.com/lambdawp-567/k8dclusterlife/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := cache.New(redisAddr)
	if err := redisClient.Ping(ctx); err != nil {
		slog.Warn("redis not available, continuing without cache", "error", err)
	} else {
		slog.Info("redis connected", "addr", redisAddr)
	}
	defer redisClient.Close()

	// PostgreSQL (optional — cluster management and settings need it)
	var db *store.DB
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		var err error
		db, err = store.Connect(ctx, dsn)
		if err != nil {
			slog.Error("postgres connect failed", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		if err := db.Migrate(); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
		slog.Info("postgres connected and migrated")
	} else {
		slog.Warn("DATABASE_URL not set — cluster management and settings disabled")
	}

	// Cluster controller
	refreshInterval := 30 * time.Second
	if v := os.Getenv("REFRESH_INTERVAL_S"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			refreshInterval = time.Duration(n) * time.Second
		}
	}
	controller := cluster.NewController(redisClient)

	// Reload persisted clusters from DB on startup
	if db != nil {
		clusters, err := db.ListClusters(ctx)
		if err != nil {
			slog.Warn("failed to load clusters from db", "error", err)
		} else {
			for _, c := range clusters {
				if err := controller.AddCluster(cluster.ClusterConfig{
					ID:   c.ID,
					Name: c.Name,
				}, nil, refreshInterval); err != nil {
					slog.Warn("cluster reload failed", "cluster", c.Name, "error", err)
				}
			}
		}
	}

	// Auto-add in-cluster config when no KUBECONFIG file is specified
	if os.Getenv("KUBECONFIG") == "" {
		if err := controller.AddCluster(cluster.ClusterConfig{
			ID:   "in-cluster",
			Name: "Local Cluster",
		}, nil, refreshInterval); err != nil {
			slog.Info("no in-cluster config available (expected outside K8s)", "error", err)
		} else {
			slog.Info("monitoring in-cluster")
		}
	}

	// Healing agent
	healingAgent := healing.New(nil)

	// Notifier
	notifier := notify.NewNotifier()
	_ = notifier

	// Auth handler — use DB-backed SSO config when DB is available
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var ssoLoader func(context.Context) ([]auth.SSOProviderConfig, error)
	if db != nil {
		ssoLoader = func(ctx context.Context) ([]auth.SSOProviderConfig, error) {
			providers, err := db.ListSSOProviders(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]auth.SSOProviderConfig, len(providers))
			for i, p := range providers {
				out[i] = auth.SSOProviderConfig{
					Provider:     p.Provider,
					Enabled:      p.Enabled,
					ClientID:     p.ClientID,
					ClientSecret: p.ClientSecret,
					TenantID:     p.TenantID,
				}
			}
			return out, nil
		}
	}

	authHandler, err := auth.New(baseURL, ssoLoader)
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

		// Protected API routes — auth enforced dynamically based on configured providers
		r.Group(func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/problems", api.HandleProblems(controller))
			r.Mount("/healing", api.HandleHealing(healingAgent))

			// DB-backed routes (only mounted when DB is available)
			if db != nil {
				r.Mount("/clusters", api.HandleClusters(db, nil, controller, "default", refreshInterval))
				r.Mount("/settings", api.HandleSettings(db))
				r.Mount("/sso", api.HandleSSO(db, authHandler))
			}
		})
	})

	// WebSocket for live cluster events
	r.Get("/ws", api.HandleWebSocket(redisClient))

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
