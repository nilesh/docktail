package kube

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nilesh/docktail/internal/backend"
)

func deleteOptions() metav1.DeleteOptions {
	return metav1.DeleteOptions{}
}

// StartWorkload is not supported for Kubernetes pods.
func (c *Client) StartWorkload(ctx context.Context, id string) error {
	return backend.ErrNotSupported
}

// StopWorkload deletes a pod (Kubernetes equivalent of stop).
func (c *Client) StopWorkload(ctx context.Context, id string) error {
	podName, err := c.podNameByID(ctx, id)
	if err != nil {
		return err
	}
	return c.clientset.CoreV1().Pods(c.namespace).Delete(ctx, podName, deleteOptions())
}

// RestartWorkload deletes a pod (the controller will recreate it).
func (c *Client) RestartWorkload(ctx context.Context, id string) error {
	return c.StopWorkload(ctx, id)
}

// PauseWorkload is not supported for Kubernetes pods.
func (c *Client) PauseWorkload(ctx context.Context, id string) error {
	return backend.ErrNotSupported
}

// UnpauseWorkload is not supported for Kubernetes pods.
func (c *Client) UnpauseWorkload(ctx context.Context, id string) error {
	return backend.ErrNotSupported
}
