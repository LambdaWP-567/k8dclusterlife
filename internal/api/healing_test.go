package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/lambdawp-567/k8dclusterlife/internal/healing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHealingAgent is a test double for the healing agent.
type mockHealingAgent struct {
	sessions map[string]*healing.Session
}

func (m *mockHealingAgent) StartSession(_ context.Context, p cluster.Problem, mode healing.AutonomyMode, _ int, _ chan<- healing.StreamEvent) *healing.Session {
	if m.sessions == nil {
		m.sessions = make(map[string]*healing.Session)
	}
	s := &healing.Session{
		ID:           "test-session-1",
		ProblemID:    p.ID,
		AutonomyMode: mode,
		Status:       healing.StatusRunning,
	}
	m.sessions[s.ID] = s
	return s
}

func (m *mockHealingAgent) ApproveAction(sessionID string, approved bool) error {
	if m.sessions == nil || m.sessions[sessionID] == nil {
		return assert.AnError
	}
	return nil
}

func (m *mockHealingAgent) GetSession(sessionID string) *healing.Session {
	if m.sessions == nil {
		return nil
	}
	return m.sessions[sessionID]
}

func TestStartHealingSession(t *testing.T) {
	agent := &mockHealingAgent{}
	h := HandleHealing(agent)

	body, _ := json.Marshal(map[string]any{
		"problem": map[string]string{
			"id":          "prob-1",
			"cluster_id":  "cluster-1",
			"kind":        "Pod",
			"name":        "broken-pod",
			"namespace":   "default",
			"status":      "CrashLoopBackOff",
			"description": "Pod crasht",
			"cause":       "Fehler im Startskript",
			"severity":    "critical",
		},
		"autonomy_mode": "MANUAL",
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var session healing.Session
	err := json.Unmarshal(w.Body.Bytes(), &session)
	require.NoError(t, err)
	assert.Equal(t, "test-session-1", session.ID)
	assert.Equal(t, healing.StatusRunning, session.Status)
}

func TestStartHealingSession_MissingProblemID(t *testing.T) {
	agent := &mockHealingAgent{}
	h := HandleHealing(agent)

	body, _ := json.Marshal(map[string]any{
		"problem": map[string]string{"kind": "Pod"},
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetHealingSession(t *testing.T) {
	agent := &mockHealingAgent{sessions: map[string]*healing.Session{
		"sess-1": {ID: "sess-1", Status: healing.StatusHealed},
	}}
	h := HandleHealing(agent)

	req := httptest.NewRequest(http.MethodGet, "/sess-1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var session healing.Session
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &session))
	assert.Equal(t, healing.StatusHealed, session.Status)
}

func TestGetHealingSession_NotFound(t *testing.T) {
	agent := &mockHealingAgent{}
	h := HandleHealing(agent)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestApproveAction(t *testing.T) {
	agent := &mockHealingAgent{sessions: map[string]*healing.Session{
		"sess-1": {ID: "sess-1"},
	}}
	h := HandleHealing(agent)

	body, _ := json.Marshal(map[string]bool{"approved": true})
	req := httptest.NewRequest(http.MethodPost, "/sess-1/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
