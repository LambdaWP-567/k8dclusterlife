package healing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// toolDefs returns the Anthropic tool definitions for the healing agent.
func toolDefs() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		tool("get_pod_logs", "Read recent logs from a Pod container",
			schemaProps{
				"namespace": prop("string", "Kubernetes namespace"),
				"pod":       prop("string", "Pod name"),
				"container": prop("string", "Container name (optional, first if omitted)"),
				"lines":     prop("integer", "Number of log lines (default 100)"),
			}, []string{"namespace", "pod"}),
		tool("get_events", "List recent Kubernetes events for a resource",
			schemaProps{
				"namespace": prop("string", "Kubernetes namespace"),
				"kind":      prop("string", "Resource kind: Pod, Node, Deployment, etc."),
				"name":      prop("string", "Resource name"),
			}, []string{"namespace", "kind", "name"}),
		tool("describe_resource", "Describe a Kubernetes resource (like kubectl describe)",
			schemaProps{
				"namespace": prop("string", "Kubernetes namespace (empty for cluster-scoped)"),
				"kind":      prop("string", "Resource kind"),
				"name":      prop("string", "Resource name"),
			}, []string{"kind", "name"}),
		tool("get_node_status", "Get status of all cluster nodes", schemaProps{}, nil),
		tool("delete_pod", "Delete a Pod (it will be recreated by its controller)",
			schemaProps{
				"namespace": prop("string", "Kubernetes namespace"),
				"pod":       prop("string", "Pod name"),
			}, []string{"namespace", "pod"}),
		tool("scale_deployment", "Scale a Deployment to a specific replica count",
			schemaProps{
				"namespace":  prop("string", "Kubernetes namespace"),
				"deployment": prop("string", "Deployment name"),
				"replicas":   prop("integer", "Target replica count"),
			}, []string{"namespace", "deployment", "replicas"}),
		tool("cordon_node", "Cordon a node (prevent new pod scheduling)",
			schemaProps{"node": prop("string", "Node name")}, []string{"node"}),
		tool("uncordon_node", "Uncordon a node (re-enable pod scheduling)",
			schemaProps{"node": prop("string", "Node name")}, []string{"node"}),
	}
}

type schemaProps map[string]map[string]string

// writeTools is the set of tools that modify cluster state.
var writeTools = map[string]bool{
	"delete_pod":       true,
	"scale_deployment": true,
	"cordon_node":      true,
	"uncordon_node":    true,
}

// IsWriteTool returns true for tools that change cluster state.
func IsWriteTool(name string) bool {
	return writeTools[name]
}

// Executor executes tool calls against a Kubernetes cluster.
type Executor struct {
	client kubernetes.Interface
}

func NewExecutor(client kubernetes.Interface) *Executor {
	return &Executor{client: client}
}

func (e *Executor) Execute(ctx context.Context, toolName, inputJSON string) (string, error) {
	var input map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("invalid tool input: %w", err)
	}

	switch toolName {
	case "get_pod_logs":
		return e.getPodLogs(ctx, input)
	case "get_events":
		return e.getEvents(ctx, input)
	case "describe_resource":
		return e.describeResource(ctx, input)
	case "get_node_status":
		return e.getNodeStatus(ctx)
	case "delete_pod":
		return e.deletePod(ctx, input)
	case "scale_deployment":
		return e.scaleDeployment(ctx, input)
	case "cordon_node":
		return e.cordonNode(ctx, input, true)
	case "uncordon_node":
		return e.cordonNode(ctx, input, false)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (e *Executor) getPodLogs(ctx context.Context, input map[string]any) (string, error) {
	ns := strVal(input, "namespace")
	pod := strVal(input, "pod")
	container := strVal(input, "container")
	lines := int64(100)
	if v, ok := input["lines"].(float64); ok && v > 0 {
		lines = int64(v)
	}

	opts := &corev1.PodLogOptions{TailLines: &lines}
	if container != "" {
		opts.Container = container
	}

	result, err := e.client.CoreV1().Pods(ns).GetLogs(pod, opts).DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("get logs: %w", err)
	}
	logs := string(result)
	if len(logs) > 8000 {
		logs = "…(truncated)\n" + logs[len(logs)-8000:]
	}
	return logs, nil
}

func (e *Executor) getEvents(ctx context.Context, input map[string]any) (string, error) {
	ns := strVal(input, "namespace")
	name := strVal(input, "name")
	kind := strVal(input, "kind")

	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s", name, kind)
	events, err := e.client.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
		Limit:         20,
	})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, ev := range events.Items {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			ev.LastTimestamp.Format(time.RFC3339),
			ev.Reason, ev.Message))
	}
	if sb.Len() == 0 {
		return "Keine Events gefunden.", nil
	}
	return sb.String(), nil
}

func (e *Executor) describeResource(ctx context.Context, input map[string]any) (string, error) {
	ns := strVal(input, "namespace")
	kind := strVal(input, "kind")
	name := strVal(input, "name")

	switch strings.ToLower(kind) {
	case "pod":
		pod, err := e.client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		data, _ := json.MarshalIndent(pod.Status, "", "  ")
		return fmt.Sprintf("Pod %s/%s Status:\n%s", ns, name, string(data)), nil
	case "node":
		node, err := e.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		data, _ := json.MarshalIndent(node.Status.Conditions, "", "  ")
		return fmt.Sprintf("Node %s Conditions:\n%s", name, string(data)), nil
	case "deployment":
		d, err := e.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		data, _ := json.MarshalIndent(d.Status, "", "  ")
		return fmt.Sprintf("Deployment %s/%s Status:\n%s", ns, name, string(data)), nil
	default:
		return fmt.Sprintf("Ressource-Typ '%s' wird nicht unterstützt.", kind), nil
	}
}

func (e *Executor) getNodeStatus(ctx context.Context) (string, error) {
	nodes, err := e.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, n := range nodes.Items {
		var conds []string
		for _, c := range n.Status.Conditions {
			conds = append(conds, fmt.Sprintf("%s=%s", c.Type, c.Status))
		}
		sb.WriteString(fmt.Sprintf("Node %s: %s\n", n.Name, strings.Join(conds, ", ")))
	}
	return sb.String(), nil
}

func (e *Executor) deletePod(ctx context.Context, input map[string]any) (string, error) {
	ns := strVal(input, "namespace")
	pod := strVal(input, "pod")
	if err := e.client.CoreV1().Pods(ns).Delete(ctx, pod, metav1.DeleteOptions{}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Pod %s/%s wurde gelöscht.", ns, pod), nil
}

func (e *Executor) scaleDeployment(ctx context.Context, input map[string]any) (string, error) {
	ns := strVal(input, "namespace")
	name := strVal(input, "deployment")
	replicas := int32(0)
	if v, ok := input["replicas"].(float64); ok {
		replicas = int32(v)
	}

	scale, err := e.client.AppsV1().Deployments(ns).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	scale.Spec.Replicas = replicas
	if _, err := e.client.AppsV1().Deployments(ns).UpdateScale(ctx, name, scale, metav1.UpdateOptions{}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Deployment %s/%s wurde auf %d Replicas skaliert.", ns, name, replicas), nil
}

func (e *Executor) cordonNode(ctx context.Context, input map[string]any, cordon bool) (string, error) {
	name := strVal(input, "node")
	node, err := e.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	node.Spec.Unschedulable = cordon
	if _, err := e.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		return "", err
	}
	action := "gecordoned"
	if !cordon {
		action = "ungecordoned"
	}
	return fmt.Sprintf("Node %s wurde %s.", name, action), nil
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func prop(typ, desc string) map[string]string {
	return map[string]string{"type": typ, "description": desc}
}

func tool(name, desc string, props schemaProps, required []string) anthropic.ToolUnionParam {
	// Convert schemaProps to the format expected by ToolInputSchemaParam
	propsAny := make(map[string]any, len(props))
	for k, v := range props {
		propsAny[k] = v
	}
	tp := anthropic.ToolParam{
		Name:        name,
		Description: anthropic.String(desc),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: propsAny,
			Required:   required,
		},
	}
	return anthropic.ToolUnionParam{OfTool: &tp}
}
