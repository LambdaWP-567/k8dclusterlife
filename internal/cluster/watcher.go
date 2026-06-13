package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const pendingThreshold = 5 * time.Minute

// Watcher collects all current problems from a K8s cluster.
type Watcher struct {
	client      kubernetes.Interface
	clusterID   string
	clusterName string
}

func NewWatcher(client kubernetes.Interface, cfg ClusterConfig) *Watcher {
	return &Watcher{client: client, clusterID: cfg.ID, clusterName: cfg.Name}
}

// Collect returns all current problems in the cluster.
func (w *Watcher) Collect(ctx context.Context) ([]Problem, error) {
	var problems []Problem

	pods, err := w.collectPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("pods: %w", err)
	}
	problems = append(problems, pods...)

	nodes, err := w.collectNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("nodes: %w", err)
	}
	problems = append(problems, nodes...)

	deploys, err := w.collectDeployments(ctx)
	if err != nil {
		return nil, fmt.Errorf("deployments: %w", err)
	}
	problems = append(problems, deploys...)

	ssets, err := w.collectStatefulSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("statefulsets: %w", err)
	}
	problems = append(problems, ssets...)

	dsets, err := w.collectDaemonSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemonsets: %w", err)
	}
	problems = append(problems, dsets...)

	pvcs, err := w.collectPVCs(ctx)
	if err != nil {
		return nil, fmt.Errorf("pvcs: %w", err)
	}
	problems = append(problems, pvcs...)

	certs, err := w.collectCertificates(ctx)
	if err != nil {
		return nil, fmt.Errorf("certs: %w", err)
	}
	problems = append(problems, certs...)

	return problems, nil
}

func (w *Watcher) makeProblem(kind, name, ns, status string) Problem {
	desc, cause, sev := Map(kind, status)
	return Problem{
		ID:          fmt.Sprintf("%s/%s/%s/%s", w.clusterID, kind, ns, name),
		ClusterID:   w.clusterID,
		ClusterName: w.clusterName,
		Kind:        kind,
		Name:        name,
		Namespace:   ns,
		Status:      status,
		Description: desc,
		Cause:       cause,
		Severity:    sev,
		DetectedAt:  time.Now(),
	}
}

func (w *Watcher) collectPods(ctx context.Context) ([]Problem, error) {
	list, err := w.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []Problem
	for _, pod := range list.Items {
		status := podStatus(pod)
		if status == "" {
			continue
		}
		out = append(out, w.makeProblem("Pod", pod.Name, pod.Namespace, status))
	}
	return out, nil
}

func podStatus(pod corev1.Pod) string {
	phase := string(pod.Status.Phase)

	// Skip fully running/succeeded pods
	if phase == "Running" {
		// Check if all containers are actually ready
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				reason := cs.State.Waiting.Reason
				if reason != "" && reason != "PodInitializing" {
					return reason
				}
			}
		}
		return ""
	}

	if phase == "Succeeded" {
		return ""
	}

	if phase == "Failed" {
		// Check container state for specific reason
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Terminated != nil && cs.State.Terminated.Reason == "OOMKilled" {
				return "OOMKilled"
			}
		}
		return "Error"
	}

	if phase == "Pending" {
		// Only report if pending for too long
		if pod.CreationTimestamp.Add(pendingThreshold).Before(time.Now()) {
			return "Pending"
		}
		return ""
	}

	if pod.DeletionTimestamp != nil {
		// Stuck in terminating
		if time.Since(pod.DeletionTimestamp.Time) > 5*time.Minute {
			return "Terminating"
		}
		return ""
	}

	// Check for eviction
	if pod.Status.Reason == "Evicted" {
		return "Evicted"
	}

	if phase == "Unknown" {
		return "Unknown"
	}

	// Check init containers
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return "Init:" + cs.State.Waiting.Reason
		}
	}

	return ""
}

func (w *Watcher) collectNodes(ctx context.Context) ([]Problem, error) {
	list, err := w.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	conditionStatuses := map[corev1.NodeConditionType]string{
		corev1.NodeReady:              "NotReady",
		corev1.NodeDiskPressure:       "DiskPressure",
		corev1.NodeMemoryPressure:     "MemoryPressure",
		corev1.NodePIDPressure:        "PIDPressure",
		corev1.NodeNetworkUnavailable: "NetworkUnavailable",
	}

	var out []Problem
	for _, node := range list.Items {
		for _, cond := range node.Status.Conditions {
			status, ok := conditionStatuses[cond.Type]
			if !ok {
				continue
			}
			// Ready should be True; others should be False
			isProblematic := (cond.Type == corev1.NodeReady && cond.Status != corev1.ConditionTrue) ||
				(cond.Type != corev1.NodeReady && cond.Status == corev1.ConditionTrue)
			if isProblematic {
				out = append(out, w.makeProblem("Node", node.Name, "", status))
			}
		}
	}
	return out, nil
}

func (w *Watcher) collectDeployments(ctx context.Context) ([]Problem, error) {
	list, err := w.client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []Problem
	for _, d := range list.Items {
		if d.Status.UnavailableReplicas > 0 {
			out = append(out, w.makeProblem("Deployment", d.Name, d.Namespace, "Unavailable"))
		}
	}
	return out, nil
}

func (w *Watcher) collectStatefulSets(ctx context.Context) ([]Problem, error) {
	list, err := w.client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []Problem
	for _, ss := range list.Items {
		if ss.Status.ReadyReplicas < *ss.Spec.Replicas {
			out = append(out, w.makeProblem("StatefulSet", ss.Name, ss.Namespace, "NotReady"))
		}
	}
	return out, nil
}

func (w *Watcher) collectDaemonSets(ctx context.Context) ([]Problem, error) {
	list, err := w.client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []Problem
	for _, ds := range list.Items {
		if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			out = append(out, w.makeProblem("DaemonSet", ds.Name, ds.Namespace, "Mismatch"))
		}
	}
	return out, nil
}

func (w *Watcher) collectPVCs(ctx context.Context) ([]Problem, error) {
	list, err := w.client.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []Problem
	for _, pvc := range list.Items {
		switch pvc.Status.Phase {
		case corev1.ClaimPending:
			if pvc.CreationTimestamp.Add(pendingThreshold).Before(time.Now()) {
				out = append(out, w.makeProblem("PVC", pvc.Name, pvc.Namespace, "Pending"))
			}
		case corev1.ClaimLost:
			out = append(out, w.makeProblem("PVC", pvc.Name, pvc.Namespace, "Lost"))
		}
	}
	return out, nil
}

func (w *Watcher) collectCertificates(ctx context.Context) ([]Problem, error) {
	list, err := w.client.CoreV1().Secrets("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=kubernetes.io/tls",
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	warnBefore := now.Add(30 * 24 * time.Hour)

	var out []Problem
	for _, secret := range list.Items {
		certData, ok := secret.Data["tls.crt"]
		if !ok {
			continue
		}

		certs, err := parseCerts(certData)
		if err != nil || len(certs) == 0 {
			continue
		}
		cert := certs[0]

		var status string
		switch {
		case cert.NotAfter.Before(now):
			status = "Expired"
		case cert.NotAfter.Before(warnBefore):
			status = "Expiring"
		default:
			continue
		}

		p := w.makeProblem("Certificate", secret.Name, secret.Namespace, status)
		daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
		if daysLeft < 0 {
			p.Description = fmt.Sprintf("TLS-Zertifikat ist seit %d Tagen abgelaufen.", -daysLeft)
		} else {
			p.Description = fmt.Sprintf("TLS-Zertifikat läuft in %d Tagen ab.", daysLeft)
		}
		out = append(out, p)
	}
	return out, nil
}

func parseCerts(data []byte) ([]*x509.Certificate, error) {
	// Try parsing as PEM first, then DER
	var certs []*x509.Certificate
	remaining := data
	for len(remaining) > 0 {
		var block []byte
		// Simple PEM header detection
		if strings.Contains(string(remaining[:min(len(remaining), 27)]), "BEGIN") {
			idx := strings.Index(string(remaining), "-----END CERTIFICATE-----")
			if idx < 0 {
				break
			}
			block = remaining[:idx+25]
			remaining = remaining[idx+25:]
		} else {
			block = remaining
			remaining = nil
		}
		c, err := tls.X509KeyPair(block, block)
		if err == nil && len(c.Certificate) > 0 {
			parsed, err := x509.ParseCertificate(c.Certificate[0])
			if err == nil {
				certs = append(certs, parsed)
			}
		} else {
			// Try DER
			parsed, err := x509.ParseCertificate(block)
			if err == nil {
				certs = append(certs, parsed)
			}
		}
		if remaining == nil {
			break
		}
	}
	if len(certs) == 0 {
		// Final attempt: parse as x509 certificates directly
		cert, err := x509.ParseCertificate(data)
		if err == nil {
			return []*x509.Certificate{cert}, nil
		}
	}
	return certs, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
