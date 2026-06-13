package testorch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RunResult captures the outcome of a scenario run.
type RunResult struct {
	ID           string    `json:"id"`
	Scenario     string    `json:"scenario"`
	Status       string    `json:"status"` // "running" | "done" | "failed"
	Error        string    `json:"error,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

// Controller orchestrates test scenarios.
type Controller struct {
	client    kubernetes.Interface
	namespace string

	mu      sync.RWMutex
	results map[string]*RunResult
}

func NewController(kubeconfig []byte, namespace string) (*Controller, error) {
	var cfg *rest.Config
	var err error
	if len(kubeconfig) == 0 {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Controller{
		client:    client,
		namespace: namespace,
		results:   make(map[string]*RunResult),
	}, nil
}

// RunScenario starts a scenario asynchronously and returns the run ID.
func (c *Controller) RunScenario(scenarioName string) (*RunResult, error) {
	s, err := Get(scenarioName)
	if err != nil {
		return nil, err
	}

	result := &RunResult{
		ID:        fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Scenario:  scenarioName,
		Status:    "running",
		StartedAt: time.Now(),
	}

	c.mu.Lock()
	c.results[result.ID] = result
	c.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.Setup(ctx, c.client, c.namespace); err != nil {
			c.finalize(result.ID, "failed", err.Error())
			return
		}

		// Wait for the scenario to be visible (30s)
		time.Sleep(30 * time.Second)

		// Teardown
		if err := s.Teardown(ctx, c.client, c.namespace); err != nil {
			slog.Warn("teardown failed", "scenario", scenarioName, "error", err)
		}

		c.finalize(result.ID, "done", "")
	}()

	return result, nil
}

func (c *Controller) GetResult(id string) *RunResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.results[id]
}

func (c *Controller) ListResults() []*RunResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	results := make([]*RunResult, 0, len(c.results))
	for _, r := range c.results {
		results = append(results, r)
	}
	return results
}

func (c *Controller) finalize(id, status, errMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.results[id]; ok {
		r.Status = status
		r.Error = errMsg
		now := time.Now()
		r.FinishedAt = &now
	}
}

// Handler returns an HTTP handler for the test orchestrator API.
func (c *Controller) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /scenarios", func(w http.ResponseWriter, r *http.Request) {
		scenarios := All()
		type info struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		out := make([]info, len(scenarios))
		for i, s := range scenarios {
			out[i] = info{Name: s.Name, Description: s.Description}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("POST /run", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Scenario string `json:"scenario"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Scenario == "" {
			http.Error(w, `{"error":"scenario required"}`, http.StatusBadRequest)
			return
		}
		result, err := c.RunScenario(body.Scenario)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(result)
	})

	mux.HandleFunc("GET /results", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(c.ListResults())
	})

	mux.HandleFunc("GET /results/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/results/"):]
		result := c.GetResult(id)
		if result == nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})

	return mux
}
