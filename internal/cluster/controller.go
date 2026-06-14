package cluster

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Cache interface {
	SetProblems(ctx context.Context, clusterID string, problems any) error
	GetProblems(ctx context.Context, clusterID string, out any) error
	Publish(ctx context.Context, channel string, payload any) error
}

// Controller manages goroutines for all monitored clusters.
type Controller struct {
	mu       sync.RWMutex
	clusters map[string]*clusterWorker
	cache    Cache
}

type clusterWorker struct {
	cfg     ClusterConfig
	cancel  context.CancelFunc
	watcher *Watcher
}

func NewController(cache Cache) *Controller {
	return &Controller{
		clusters: make(map[string]*clusterWorker),
		cache:    cache,
	}
}

// AddCluster starts monitoring a cluster with the given kubeconfig bytes.
func (c *Controller) AddCluster(cfg ClusterConfig, kubeconfigData []byte, interval time.Duration) error {
	client, err := buildClient(kubeconfigData)
	if err != nil {
		return err
	}

	watcher := NewWatcher(client, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	if w, ok := c.clusters[cfg.ID]; ok {
		w.cancel()
	}
	c.clusters[cfg.ID] = &clusterWorker{cfg: cfg, cancel: cancel, watcher: watcher}
	c.mu.Unlock()

	go c.runWorker(ctx, cfg, watcher, interval)
	return nil
}

// RemoveCluster stops monitoring a cluster.
func (c *Controller) RemoveCluster(clusterID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if w, ok := c.clusters[clusterID]; ok {
		w.cancel()
		delete(c.clusters, clusterID)
	}
}

func (c *Controller) runWorker(ctx context.Context, cfg ClusterConfig, w *Watcher, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.poll(ctx, cfg, w)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.poll(ctx, cfg, w)
		}
	}
}

func (c *Controller) poll(ctx context.Context, cfg ClusterConfig, w *Watcher) {
	problems, err := w.Collect(ctx)
	if err != nil {
		slog.Error("cluster poll failed", "cluster", cfg.Name, "error", err)
		return
	}

	if err := c.cache.SetProblems(ctx, cfg.ID, problems); err != nil {
		slog.Error("cache set failed", "cluster", cfg.Name, "error", err)
	}

	_ = c.cache.Publish(ctx, "problems", Event{
		Type:    "cluster_status",
		Cluster: cfg.Name,
		At:      time.Now(),
	})

	slog.Debug("cluster polled", "cluster", cfg.Name, "problems", len(problems))
}

// GetProblems returns cached problems for all clusters.
func (c *Controller) GetProblems(ctx context.Context) ([]Problem, error) {
	c.mu.RLock()
	ids := make([]string, 0, len(c.clusters))
	for id := range c.clusters {
		ids = append(ids, id)
	}
	c.mu.RUnlock()

	var all []Problem
	for _, id := range ids {
		var problems []Problem
		if err := c.cache.GetProblems(ctx, id, &problems); err != nil {
			slog.Warn("cache get failed", "cluster_id", id, "error", err)
			continue
		}
		all = append(all, problems...)
	}
	return all, nil
}

func buildClient(kubeconfigData []byte) (kubernetes.Interface, error) {
	var cfg *rest.Config
	var err error

	if len(kubeconfigData) == 0 {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	}
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

// jsonRoundTrip is a helper for cache tests.
func jsonRoundTrip(v any) ([]byte, error) {
	return json.Marshal(v)
}
