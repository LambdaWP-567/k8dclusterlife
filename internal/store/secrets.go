package store

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretStore stores kubeconfigs as K8s Secrets in the app namespace.
type SecretStore struct {
	client    kubernetes.Interface
	namespace string
}

func NewSecretStore(client kubernetes.Interface, namespace string) *SecretStore {
	return &SecretStore{client: client, namespace: namespace}
}

// SaveKubeconfig stores or updates a kubeconfig Secret.
func (s *SecretStore) SaveKubeconfig(ctx context.Context, secretName string, data []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "k8dclusterlife",
				"k8dclusterlife/type":          "kubeconfig",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": data,
		},
	}

	_, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		_, err = s.client.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	_, err = s.client.CoreV1().Secrets(s.namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

// GetKubeconfig retrieves kubeconfig bytes from a Secret.
func (s *SecretStore) GetKubeconfig(ctx context.Context, secretName string) ([]byte, error) {
	secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get secret %s: %w", secretName, err)
	}
	data, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("secret %s has no kubeconfig key", secretName)
	}
	return data, nil
}

// DeleteKubeconfig removes a kubeconfig Secret.
func (s *SecretStore) DeleteKubeconfig(ctx context.Context, secretName string) error {
	err := s.client.CoreV1().Secrets(s.namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}
