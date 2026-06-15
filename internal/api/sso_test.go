package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lambdawp-567/k8dclusterlife/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSSOStore struct {
	providers map[string]store.SSOProvider
}

func newFakeSSOStore() *fakeSSOStore {
	s := &fakeSSOStore{providers: make(map[string]store.SSOProvider)}
	for _, p := range []string{"entra", "github", "google"} {
		s.providers[p] = store.SSOProvider{Provider: p, UpdatedAt: time.Now()}
	}
	return s
}

func (f *fakeSSOStore) ListSSOProviders(_ context.Context) ([]store.SSOProvider, error) {
	out := make([]store.SSOProvider, 0, len(f.providers))
	for _, p := range []string{"entra", "github", "google"} {
		out = append(out, f.providers[p])
	}
	return out, nil
}

func (f *fakeSSOStore) GetSSOProvider(_ context.Context, provider string) (store.SSOProvider, error) {
	return f.providers[provider], nil
}

func (f *fakeSSOStore) SetSSOProvider(_ context.Context, p store.SSOProvider) error {
	f.providers[p.Provider] = p
	return nil
}

type fakeReloader struct{ reloaded bool }

func (f *fakeReloader) ReloadFromDB(_ context.Context) error {
	f.reloaded = true
	return nil
}

func TestHandleSSO_List(t *testing.T) {
	db := newFakeSSOStore()
	db.providers["github"] = store.SSOProvider{
		Provider:     "github",
		Enabled:      true,
		ClientID:     "my-client-id",
		ClientSecret: "super-secret",
		UpdatedAt:    time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	HandleSSO(db, nil).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var providers []ssoResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &providers))
	require.Len(t, providers, 3)

	var gh ssoResponse
	for _, p := range providers {
		if p.Provider == "github" {
			gh = p
		}
	}
	assert.True(t, gh.Enabled)
	assert.Equal(t, "my-client-id", gh.ClientID)
	assert.Equal(t, secretMask, gh.ClientSecret, "secret should be masked")
}

func TestHandleSSO_Put_PreservesSecret(t *testing.T) {
	db := newFakeSSOStore()
	db.providers["github"] = store.SSOProvider{
		Provider:     "github",
		Enabled:      true,
		ClientID:     "existing-id",
		ClientSecret: "existing-secret",
		UpdatedAt:    time.Now(),
	}
	reloader := &fakeReloader{}

	body := `{"provider":"github","enabled":true,"client_id":"new-id","client_secret":"` + secretMask + `"}`
	req := httptest.NewRequest(http.MethodPut, "/github", strings.NewReader(body))
	req.SetPathValue("provider", "github")
	w := httptest.NewRecorder()

	HandleSSO(db, reloader).ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, reloader.reloaded)
	assert.Equal(t, "existing-secret", db.providers["github"].ClientSecret, "secret should be preserved when mask is sent")
	assert.Equal(t, "new-id", db.providers["github"].ClientID)
}

func TestHandleSSO_Put_UpdatesSecret(t *testing.T) {
	db := newFakeSSOStore()
	db.providers["github"] = store.SSOProvider{
		Provider:     "github",
		ClientSecret: "old-secret",
		UpdatedAt:    time.Now(),
	}
	reloader := &fakeReloader{}

	body := `{"provider":"github","enabled":true,"client_id":"id","client_secret":"new-secret"}`
	req := httptest.NewRequest(http.MethodPut, "/github", strings.NewReader(body))
	req.SetPathValue("provider", "github")
	w := httptest.NewRecorder()

	HandleSSO(db, reloader).ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "new-secret", db.providers["github"].ClientSecret)
}

func TestHandleSSO_Put_UnknownProvider(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/unknown", strings.NewReader(`{}`))
	req.SetPathValue("provider", "unknown")
	w := httptest.NewRecorder()

	HandleSSO(newFakeSSOStore(), nil).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
