package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lambdawp-567/k8dclusterlife/internal/store"
)

const secretMask = "••••••••"

type SSOStore interface {
	ListSSOProviders(ctx context.Context) ([]store.SSOProvider, error)
	GetSSOProvider(ctx context.Context, provider string) (store.SSOProvider, error)
	SetSSOProvider(ctx context.Context, p store.SSOProvider) error
}

type SSOReloader interface {
	ReloadFromDB(ctx context.Context) error
}

type ssoResponse struct {
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id,omitempty"`
}

func HandleSSO(db SSOStore, reloader SSOReloader) http.Handler {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		providers, err := db.ListSSOProviders(r.Context())
		if err != nil {
			jsonError(w, "failed to load SSO config", http.StatusInternalServerError)
			return
		}
		out := make([]ssoResponse, len(providers))
		for i, p := range providers {
			out[i] = ssoResponse{
				Provider: p.Provider,
				Enabled:  p.Enabled,
				ClientID: p.ClientID,
				TenantID: p.TenantID,
			}
			if p.ClientSecret != "" {
				out[i].ClientSecret = secretMask
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	r.Put("/{provider}", func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")
		switch providerName {
		case "entra", "github", "google":
		default:
			jsonError(w, "unknown provider", http.StatusBadRequest)
			return
		}

		var body ssoResponse
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid body", http.StatusBadRequest)
			return
		}

		existing, err := db.GetSSOProvider(r.Context(), providerName)
		if err != nil {
			jsonError(w, "failed to load existing config", http.StatusInternalServerError)
			return
		}

		secret := body.ClientSecret
		if secret == secretMask || secret == "" {
			secret = existing.ClientSecret
		}

		p := store.SSOProvider{
			Provider:     providerName,
			Enabled:      body.Enabled,
			ClientID:     body.ClientID,
			ClientSecret: secret,
			TenantID:     body.TenantID,
		}
		if err := db.SetSSOProvider(r.Context(), p); err != nil {
			jsonError(w, "failed to save SSO config", http.StatusInternalServerError)
			return
		}

		if reloader != nil {
			if err := reloader.ReloadFromDB(r.Context()); err != nil {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{
					"warning": "config saved but provider reload failed: " + err.Error(),
				})
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	})

	return r
}
