package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/lambdawp-567/k8dclusterlife/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock implementations ---

type mockClusterStore struct {
	clusters []store.Cluster
	err      error
}

func (m *mockClusterStore) ListClusters(_ context.Context) ([]store.Cluster, error) {
	return m.clusters, m.err
}
func (m *mockClusterStore) CreateCluster(_ context.Context, name, secretName, secretNamespace string) (*store.Cluster, error) {
	if m.err != nil {
		return nil, m.err
	}
	c := store.Cluster{ID: "test-id", Name: name, SecretName: secretName, SecretNamespace: secretNamespace}
	m.clusters = append(m.clusters, c)
	return &c, nil
}
func (m *mockClusterStore) GetCluster(_ context.Context, id string) (*store.Cluster, error) {
	for _, c := range m.clusters {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, assert.AnError
}
func (m *mockClusterStore) DeleteCluster(_ context.Context, id string) error {
	for i, c := range m.clusters {
		if c.ID == id {
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
			return nil
		}
	}
	return nil
}

type mockKubeconfigStore struct {
	data map[string][]byte
	err  error
}

func (m *mockKubeconfigStore) SaveKubeconfig(_ context.Context, name string, data []byte) error {
	if m.err != nil {
		return m.err
	}
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[name] = data
	return nil
}
func (m *mockKubeconfigStore) GetKubeconfig(_ context.Context, name string) ([]byte, error) {
	return m.data[name], nil
}
func (m *mockKubeconfigStore) DeleteKubeconfig(_ context.Context, name string) error {
	delete(m.data, name)
	return nil
}

type mockController struct {
	added   []string
	removed []string
	err     error
}

func (m *mockController) AddCluster(cfg cluster.ClusterConfig, _ []byte, _ time.Duration) error {
	m.added = append(m.added, cfg.ID)
	return m.err
}
func (m *mockController) RemoveCluster(id string) {
	m.removed = append(m.removed, id)
}

// --- tests ---

func TestListClusters_Empty(t *testing.T) {
	h := HandleClusters(&mockClusterStore{}, &mockKubeconfigStore{}, &mockController{}, "default", 30*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "[]")
}

func TestCreateCluster_Success(t *testing.T) {
	db := &mockClusterStore{}
	secrets := &mockKubeconfigStore{}
	ctrl := &mockController{}

	h := HandleClusters(db, secrets, ctrl, "default", 30*time.Second)

	body, ct := buildMultipart(t, "my-cluster", []byte("fake-kubeconfig"))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "my-cluster")
	assert.Len(t, db.clusters, 1)
	assert.NotEmpty(t, secrets.data)
}

func TestCreateCluster_MissingName(t *testing.T) {
	h := HandleClusters(&mockClusterStore{}, &mockKubeconfigStore{}, &mockController{}, "default", 30*time.Second)
	body, ct := buildMultipart(t, "", []byte("fake"))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteCluster(t *testing.T) {
	db := &mockClusterStore{clusters: []store.Cluster{{ID: "abc", Name: "test", SecretName: "kubeconfig-test"}}}
	secrets := &mockKubeconfigStore{data: map[string][]byte{"kubeconfig-test": []byte("x")}}
	ctrl := &mockController{}

	h := HandleClusters(db, secrets, ctrl, "default", 30*time.Second)
	req := httptest.NewRequest(http.MethodDelete, "/abc", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, db.clusters)
	assert.Equal(t, []string{"abc"}, ctrl.removed)
}

func TestSanitizeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My Cluster", "my-cluster"},
		{"prod_k8s", "prod-k8s"},
		{"Cluster@123!", "cluster123"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, sanitizeName(c.in), c.in)
	}
}

func buildMultipart(t *testing.T, name string, kubeconfig []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("name", name)
	if kubeconfig != nil {
		fw, _ := w.CreateFormFile("kubeconfig", "kubeconfig.yaml")
		_, _ = fw.Write(kubeconfig)
	}
	w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

func TestSanitizeName_Long(t *testing.T) {
	long := strings.Repeat("a", 100)
	result := sanitizeName(long)
	assert.LessOrEqual(t, len(result), 40)
}
