package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/lambdawp-567/k8dclusterlife/internal/store"
)

type ClusterStore interface {
	ListClusters(ctx context.Context) ([]store.Cluster, error)
	CreateCluster(ctx context.Context, name, secretName, secretNamespace string) (*store.Cluster, error)
	GetCluster(ctx context.Context, id string) (*store.Cluster, error)
	DeleteCluster(ctx context.Context, id string) error
}

type KubeconfigStore interface {
	SaveKubeconfig(ctx context.Context, secretName string, data []byte) error
	GetKubeconfig(ctx context.Context, secretName string) ([]byte, error)
	DeleteKubeconfig(ctx context.Context, secretName string) error
}

type ClusterController interface {
	AddCluster(cfg cluster.ClusterConfig, kubeconfigData []byte, interval time.Duration) error
	RemoveCluster(clusterID string)
}

type clustersHandler struct {
	db          ClusterStore
	secrets     KubeconfigStore
	controller  ClusterController
	namespace   string
	refreshInterval time.Duration
}

func HandleClusters(db ClusterStore, secrets KubeconfigStore, controller ClusterController, namespace string, refreshInterval time.Duration) http.Handler {
	h := &clustersHandler{
		db:              db,
		secrets:         secrets,
		controller:      controller,
		namespace:       namespace,
		refreshInterval: refreshInterval,
	}
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *clustersHandler) list(w http.ResponseWriter, r *http.Request) {
	clusters, err := h.db.ListClusters(r.Context())
	if err != nil {
		jsonError(w, "failed to list clusters", http.StatusInternalServerError)
		return
	}
	if clusters == nil {
		clusters = []store.Cluster{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clusters)
}

func (h *clustersHandler) create(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (kubeconfig file + name field)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		jsonError(w, "invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("kubeconfig")
	if err != nil {
		jsonError(w, "kubeconfig file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	kubeconfigData, err := io.ReadAll(io.LimitReader(file, 512*1024))
	if err != nil {
		jsonError(w, "failed to read kubeconfig", http.StatusBadRequest)
		return
	}

	// Sanitize name for use in K8s secret name
	secretName := fmt.Sprintf("kubeconfig-%s", sanitizeName(name))

	c, err := h.db.CreateCluster(r.Context(), name, secretName, h.namespace)
	if err != nil {
		jsonError(w, "failed to create cluster record", http.StatusInternalServerError)
		return
	}

	if err := h.secrets.SaveKubeconfig(r.Context(), secretName, kubeconfigData); err != nil {
		// Rollback DB record
		_ = h.db.DeleteCluster(r.Context(), c.ID)
		jsonError(w, "failed to store kubeconfig: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.controller.AddCluster(cluster.ClusterConfig{
		ID:   c.ID,
		Name: c.Name,
	}, kubeconfigData, h.refreshInterval); err != nil {
		// Non-fatal — cluster is saved but may not be reachable
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"cluster": c,
			"warning": "cluster saved but connection failed: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

func (h *clustersHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	c, err := h.db.GetCluster(r.Context(), id)
	if err != nil {
		jsonError(w, "cluster not found", http.StatusNotFound)
		return
	}

	h.controller.RemoveCluster(id)

	if err := h.secrets.DeleteKubeconfig(r.Context(), c.SecretName); err != nil {
		// Log but continue — delete cluster record regardless
	}

	if err := h.db.DeleteCluster(r.Context(), id); err != nil {
		jsonError(w, "failed to delete cluster", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func sanitizeName(name string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(name) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		} else if c == ' ' || c == '_' {
			b.WriteRune('-')
		}
	}
	result := b.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
