package cluster

import "time"

// Severity controls sort order on the dashboard.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Problem represents a single detected issue in a cluster.
type Problem struct {
	ID          string    `json:"id"`
	ClusterID   string    `json:"cluster_id"`
	ClusterName string    `json:"cluster_name"`
	Kind        string    `json:"kind"`        // Pod, Node, Deployment, …
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Status      string    `json:"status"`      // raw K8s status
	Description string    `json:"description"` // human-readable (DE)
	Cause       string    `json:"cause"`       // root-cause summary (DE)
	Severity    Severity  `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ClusterConfig holds connection info for one monitored cluster.
type ClusterConfig struct {
	ID              string
	Name            string
	SecretName      string
	SecretNamespace string
}

// Event is sent over WebSocket / Redis Pub/Sub when the problem list changes.
type Event struct {
	Type    string    `json:"type"` // "problem_added" | "problem_resolved" | "cluster_status"
	Cluster string    `json:"cluster"`
	Problem *Problem  `json:"problem,omitempty"`
	At      time.Time `json:"at"`
}
