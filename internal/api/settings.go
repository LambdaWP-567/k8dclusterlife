package api

import (
	"context"
	"encoding/json"
	"net/http"
)

type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

var settingKeys = []string{
	"autonomy_mode",
	"countdown_seconds",
	"refresh_interval_s",
	"teams_webhook_url",
	"claude_model",
}

func HandleSettings(store SettingsStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		settings := make(map[string]string)
		for _, key := range settingKeys {
			val, err := store.GetSetting(r.Context(), key)
			if err != nil {
				continue
			}
			settings[key] = val
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
	})

	mux.HandleFunc("PUT /", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid body", http.StatusBadRequest)
			return
		}
		allowed := make(map[string]bool, len(settingKeys))
		for _, k := range settingKeys {
			allowed[k] = true
		}
		for key, val := range body {
			if !allowed[key] {
				continue
			}
			if err := store.SetSetting(r.Context(), key, val); err != nil {
				jsonError(w, "failed to save setting "+key, http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return mux
}
