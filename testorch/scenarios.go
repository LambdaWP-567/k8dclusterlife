package testorch

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Scenario represents a test failure scenario.
type Scenario struct {
	Name        string
	Description string
	Setup       func(ctx context.Context, client kubernetes.Interface, ns string) error
	Teardown    func(ctx context.Context, client kubernetes.Interface, ns string) error
}

// All returns all available test scenarios.
func All() []Scenario {
	return []Scenario{
		scenarioCrashLoop(),
		scenarioImagePullBackOff(),
		scenarioScaleZero(),
		scenarioPending(),
		scenarioNodeCordon(),
	}
}

// Get returns a scenario by name.
func Get(name string) (*Scenario, error) {
	for _, s := range All() {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("scenario %q not found", name)
}

func ptr[T any](v T) *T { return &v }

// --- CrashLoopBackOff scenario ---

func scenarioCrashLoop() Scenario {
	return Scenario{
		Name:        "crashloop",
		Description: "Deploys a Pod that immediately exits with error — triggers CrashLoopBackOff",
		Setup: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			_, err := client.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-crashloop", Namespace: ns, Labels: testLabel()},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr(int32(1)),
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test-crashloop"}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-crashloop"}},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:    "crash",
								Image:   "alpine:3.19",
								Command: []string{"sh", "-c", "exit 1"},
							}},
							RestartPolicy: corev1.RestartPolicyAlways,
						},
					},
				},
			}, metav1.CreateOptions{})
			return err
		},
		Teardown: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			return client.AppsV1().Deployments(ns).Delete(ctx, "test-crashloop", metav1.DeleteOptions{})
		},
	}
}

// --- ImagePullBackOff scenario ---

func scenarioImagePullBackOff() Scenario {
	return Scenario{
		Name:        "imagepullbackoff",
		Description: "Deploys a Pod with a non-existent image — triggers ImagePullBackOff",
		Setup: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			_, err := client.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-imagepull", Namespace: ns, Labels: testLabel()},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "pull",
						Image: "nonexistent-registry.invalid/no-such-image:latest",
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			}, metav1.CreateOptions{})
			return err
		},
		Teardown: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			return client.CoreV1().Pods(ns).Delete(ctx, "test-imagepull", metav1.DeleteOptions{})
		},
	}
}

// --- Scale-to-zero scenario ---

func scenarioScaleZero() Scenario {
	name := "test-scale-zero"
	return Scenario{
		Name:        "scale-zero",
		Description: "Creates a Deployment and scales it to 0 — triggers UnavailableReplicas",
		Setup: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			_, err := client.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: testLabel()},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr(int32(0)),
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "nginx",
								Image: "nginx:alpine",
							}},
						},
					},
				},
			}, metav1.CreateOptions{})
			return err
		},
		Teardown: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			return client.AppsV1().Deployments(ns).Delete(ctx, name, metav1.DeleteOptions{})
		},
	}
}

// --- Pending (insufficient resources) scenario ---

func scenarioPending() Scenario {
	return Scenario{
		Name:        "pending",
		Description: "Deploys a Pod requesting impossible resources — triggers Pending > 5min",
		Setup: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			_, err := client.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pending", Namespace: ns, Labels: testLabel()},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "pending",
						Image: "nginx:alpine",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("9999"),
								corev1.ResourceMemory: resource.MustParse("9999Gi"),
							},
						},
					}},
				},
			}, metav1.CreateOptions{})
			return err
		},
		Teardown: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			return client.CoreV1().Pods(ns).Delete(ctx, "test-pending", metav1.DeleteOptions{})
		},
	}
}

// --- Node cordon scenario ---

func scenarioNodeCordon() Scenario {
	var cordonedNode string
	return Scenario{
		Name:        "node-cordon",
		Description: "Cordons a node — triggers Node.Unschedulable warning",
		Setup: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
			if err != nil || len(nodes.Items) == 0 {
				return fmt.Errorf("no nodes available")
			}
			n := nodes.Items[0]
			cordonedNode = n.Name
			n.Spec.Unschedulable = true
			_, err = client.CoreV1().Nodes().Update(ctx, &n, metav1.UpdateOptions{})
			return err
		},
		Teardown: func(ctx context.Context, client kubernetes.Interface, ns string) error {
			if cordonedNode == "" {
				return nil
			}
			n, err := client.CoreV1().Nodes().Get(ctx, cordonedNode, metav1.GetOptions{})
			if err != nil {
				return err
			}
			n.Spec.Unschedulable = false
			_, err = client.CoreV1().Nodes().Update(ctx, n, metav1.UpdateOptions{})
			return err
		},
	}
}

func testLabel() map[string]string {
	return map[string]string{"k8dclusterlife/test": "true"}
}
