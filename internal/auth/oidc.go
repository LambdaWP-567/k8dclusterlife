package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

const cookieName = "k8dlife_session"

type Provider string

const (
	ProviderEntra  Provider = "entra"
	ProviderGitHub Provider = "github"
	ProviderGoogle Provider = "google"
)

// SSOProviderConfig holds credentials for a single SSO provider loaded from DB.
type SSOProviderConfig struct {
	Provider     string
	Enabled      bool
	ClientID     string
	ClientSecret string
	TenantID     string
}

type Claims struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	jwt.RegisteredClaims
}

type Handler struct {
	baseURL   string
	jwtSecret []byte
	mu        sync.RWMutex
	providers map[Provider]*providerConfig
	ssoLoader func(context.Context) ([]SSOProviderConfig, error)
}

type providerConfig struct {
	oauth2Config *oauth2.Config
	oidcProvider *gooidc.Provider // nil for GitHub
}

// New creates an auth handler. When ssoLoader is non-nil (DB available), provider
// config is loaded from the database; otherwise env vars are used as fallback.
func New(baseURL string, ssoLoader func(context.Context) ([]SSOProviderConfig, error)) (*Handler, error) {
	h := &Handler{
		baseURL:   baseURL,
		jwtSecret: []byte(getEnv("JWT_SECRET", "dev-secret-change-me")),
		providers: make(map[Provider]*providerConfig),
		ssoLoader: ssoLoader,
	}

	ctx := context.Background()

	if ssoLoader != nil {
		if err := h.ReloadFromDB(ctx); err != nil {
			slog.Warn("SSO config from DB failed, falling back to env vars", "error", err)
			if err2 := h.loadFromEnv(ctx); err2 != nil {
				return nil, err2
			}
		}
	} else {
		if err := h.loadFromEnv(ctx); err != nil {
			return nil, err
		}
	}

	return h, nil
}

func (h *Handler) loadFromEnv(ctx context.Context) error {
	// Entra ID
	if os.Getenv("ENTRA_ENABLED") == "true" {
		tenantID := os.Getenv("ENTRA_TENANT_ID")
		issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
		p, err := gooidc.NewProvider(ctx, issuer)
		if err != nil {
			return fmt.Errorf("entra provider: %w", err)
		}
		h.providers[ProviderEntra] = &providerConfig{
			oidcProvider: p,
			oauth2Config: &oauth2.Config{
				ClientID:     os.Getenv("ENTRA_CLIENT_ID"),
				ClientSecret: os.Getenv("ENTRA_CLIENT_SECRET"),
				Endpoint:     p.Endpoint(),
				RedirectURL:  h.baseURL + "/auth/entra/callback",
				Scopes:       []string{gooidc.ScopeOpenID, "profile", "email"},
			},
		}
	}

	// Google
	if os.Getenv("GOOGLE_AUTH_ENABLED") == "true" {
		p, err := gooidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			return fmt.Errorf("google provider: %w", err)
		}
		h.providers[ProviderGoogle] = &providerConfig{
			oidcProvider: p,
			oauth2Config: &oauth2.Config{
				ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
				ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
				Endpoint:     google.Endpoint,
				RedirectURL:  h.baseURL + "/auth/google/callback",
				Scopes:       []string{gooidc.ScopeOpenID, "profile", "email"},
			},
		}
	}

	// GitHub (OAuth2 only — not OIDC)
	if os.Getenv("GITHUB_AUTH_ENABLED") == "true" {
		h.providers[ProviderGitHub] = &providerConfig{
			oauth2Config: &oauth2.Config{
				ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
				ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
				Endpoint:     github.Endpoint,
				RedirectURL:  h.baseURL + "/auth/github/callback",
				Scopes:       []string{"user:email", "read:user"},
			},
		}
	}

	return nil
}

// ReloadFromDB rebuilds the active provider set from the DB-backed SSO config.
// It is safe to call concurrently with other handler methods.
func (h *Handler) ReloadFromDB(ctx context.Context) error {
	if h.ssoLoader == nil {
		return nil
	}
	configs, err := h.ssoLoader(ctx)
	if err != nil {
		return fmt.Errorf("load SSO configs: %w", err)
	}

	newProviders := make(map[Provider]*providerConfig)

	for _, cfg := range configs {
		if !cfg.Enabled || cfg.ClientID == "" || cfg.ClientSecret == "" {
			continue
		}
		p := Provider(cfg.Provider)
		switch p {
		case ProviderEntra:
			if cfg.TenantID == "" {
				continue
			}
			issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", cfg.TenantID)
			op, err := gooidc.NewProvider(ctx, issuer)
			if err != nil {
				return fmt.Errorf("entra OIDC: %w", err)
			}
			newProviders[ProviderEntra] = &providerConfig{
				oidcProvider: op,
				oauth2Config: &oauth2.Config{
					ClientID:     cfg.ClientID,
					ClientSecret: cfg.ClientSecret,
					Endpoint:     op.Endpoint(),
					RedirectURL:  h.baseURL + "/auth/entra/callback",
					Scopes:       []string{gooidc.ScopeOpenID, "profile", "email"},
				},
			}
		case ProviderGoogle:
			op, err := gooidc.NewProvider(ctx, "https://accounts.google.com")
			if err != nil {
				return fmt.Errorf("google OIDC: %w", err)
			}
			newProviders[ProviderGoogle] = &providerConfig{
				oidcProvider: op,
				oauth2Config: &oauth2.Config{
					ClientID:     cfg.ClientID,
					ClientSecret: cfg.ClientSecret,
					Endpoint:     google.Endpoint,
					RedirectURL:  h.baseURL + "/auth/google/callback",
					Scopes:       []string{gooidc.ScopeOpenID, "profile", "email"},
				},
			}
		case ProviderGitHub:
			newProviders[ProviderGitHub] = &providerConfig{
				oauth2Config: &oauth2.Config{
					ClientID:     cfg.ClientID,
					ClientSecret: cfg.ClientSecret,
					Endpoint:     github.Endpoint,
					RedirectURL:  h.baseURL + "/auth/github/callback",
					Scopes:       []string{"user:email", "read:user"},
				},
			}
		}
	}

	h.mu.Lock()
	h.providers = newProviders
	h.mu.Unlock()
	return nil
}

// EnabledProviders returns list of configured providers for the login page.
func (h *Handler) EnabledProviders() []Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []Provider
	for _, p := range []Provider{ProviderEntra, ProviderGitHub, ProviderGoogle} {
		if _, ok := h.providers[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

// HandleLogin redirects the user to the OAuth2 authorization URL.
func (h *Handler) HandleLogin(provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		pc, ok := h.providers[provider]
		h.mu.RUnlock()
		if !ok {
			http.Error(w, "provider not configured", http.StatusNotFound)
			return
		}
		state := randomState()
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, pc.oauth2Config.AuthCodeURL(state), http.StatusFound)
	}
}

// HandleCallback exchanges the authorization code for a JWT session cookie.
func (h *Handler) HandleCallback(provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		pc, ok := h.providers[provider]
		h.mu.RUnlock()
		if !ok {
			http.Error(w, "provider not configured", http.StatusNotFound)
			return
		}

		// CSRF check
		stateCookie, err := r.Cookie("oauth_state")
		if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		token, err := pc.oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "token exchange failed", http.StatusBadRequest)
			return
		}

		claims, err := h.extractClaims(ctx, provider, pc, token)
		if err != nil {
			http.Error(w, "failed to get user info", http.StatusInternalServerError)
			return
		}

		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := jwtToken.SignedString(h.jwtSecret)
		if err != nil {
			http.Error(w, "failed to sign token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    signed,
			Path:     "/",
			MaxAge:   int((8 * time.Hour).Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// HandleLogout clears the session cookie.
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// HandleMe returns the current user info as JSON (or 401).
func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"sub":      claims.Sub,
		"email":    claims.Email,
		"name":     claims.Name,
		"provider": claims.Provider,
	})
}

func (h *Handler) extractClaims(ctx context.Context, provider Provider, pc *providerConfig, token *oauth2.Token) (*Claims, error) {
	switch provider {
	case ProviderGitHub:
		return githubUserInfo(ctx, pc.oauth2Config, token)
	default:
		return oidcUserInfo(ctx, pc, token)
	}
}

func oidcUserInfo(ctx context.Context, pc *providerConfig, token *oauth2.Token) (*Claims, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}
	verifier := pc.oidcProvider.Verifier(&gooidc.Config{ClientID: pc.oauth2Config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify id_token: %w", err)
	}
	var info struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&info); err != nil {
		return nil, err
	}
	return &Claims{
		Sub:      info.Sub,
		Email:    info.Email,
		Name:     info.Name,
		Provider: string(ProviderEntra),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}, nil
}

func githubUserInfo(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*Claims, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var info struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	name := info.Name
	if name == "" {
		name = info.Login
	}
	return &Claims{
		Sub:      fmt.Sprintf("github|%d", info.ID),
		Email:    info.Email,
		Name:     name,
		Provider: string(ProviderGitHub),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}, nil
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
