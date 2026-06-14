package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/lambdawp-567/k8dclusterlife/internal/healing"
)

type HealingAgent interface {
	StartSession(ctx context.Context, problem cluster.Problem, mode healing.AutonomyMode, countdownSec int, events chan<- healing.StreamEvent) *healing.Session
	ApproveAction(sessionID string, approved bool) error
	GetSession(sessionID string) *healing.Session
}

type healingHandler struct {
	agent HealingAgent
}

// HandleHealing mounts the healing routes on the given router.
func HandleHealing(agent HealingAgent) http.Handler {
	h := &healingHandler{agent: agent}
	r := chi.NewRouter()
	r.Post("/", h.start)
	r.Get("/{id}", h.get)
	r.Post("/{id}/approve", h.approve)
	r.Get("/{id}/stream", h.stream)
	return r
}

type startRequest struct {
	Problem      cluster.Problem    `json:"problem"`
	AutonomyMode healing.AutonomyMode `json:"autonomy_mode"`
}

func (h *healingHandler) start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Problem.ID == "" {
		jsonError(w, "problem.id is required", http.StatusBadRequest)
		return
	}

	mode := req.AutonomyMode
	if mode == "" {
		mode = healing.AutonomyMode(getEnvFallback("AUTONOMY_MODE", "MANUAL"))
	}

	countdown := 60
	if v := os.Getenv("COUNTDOWN_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			countdown = n
		}
	}

	// Events are streamed via SSE — start goroutine, buffer 100 events
	events := make(chan healing.StreamEvent, 100)
	session := h.agent.StartSession(r.Context(), req.Problem, mode, countdown, events)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(session)
}

func (h *healingHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session := h.agent.GetSession(id)
	if session == nil {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

type approveRequest struct {
	Approved bool `json:"approved"`
}

func (h *healingHandler) approve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.agent.ApproveAction(id, req.Approved); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// stream sends Server-Sent Events for a healing session.
func (h *healingHandler) stream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check the session exists
	if session := h.agent.GetSession(id); session == nil {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Poll the session state and send updates
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastActionCount int
	var lastDialogCount int

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			session := h.agent.GetSession(id)
			if session == nil {
				return
			}

			// Send new dialog messages
			for i := lastDialogCount; i < len(session.Dialog); i++ {
				data, _ := json.Marshal(healing.StreamEvent{
					Type:    "text",
					Content: session.Dialog[i].Content,
				})
				_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
			}
			lastDialogCount = len(session.Dialog)

			// Send new actions
			for i := lastActionCount; i < len(session.Actions); i++ {
				a := session.Actions[i]
				data, _ := json.Marshal(healing.StreamEvent{
					Type:   "tool_result",
					Tool:   a.ToolName,
					Output: a.Output,
				})
				_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
			}
			lastActionCount = len(session.Actions)

			flusher.Flush()

			// End stream when session is done
			if session.Status != healing.StatusRunning {
				data, _ := json.Marshal(healing.StreamEvent{
					Type:    "done",
					Content: session.Result,
				})
				_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
				flusher.Flush()
				return
			}
		}
	}
}

func getEnvFallback(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
