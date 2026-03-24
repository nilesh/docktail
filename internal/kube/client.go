package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nilesh/docktail/internal/model"
)

// Client wraps the Kubernetes API client.
type Client struct {
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
	namespace  string
	context    string
}

// NewClient creates a new Kubernetes client for the given context and namespace.
func NewClient(kubeContext, namespace string) (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	if kubeContext != "" {
		configOverrides.CurrentContext = kubeContext
	}
	if namespace != "" {
		configOverrides.Context.Namespace = namespace
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	// Resolve namespace from kubeconfig if not specified
	ns := namespace
	if ns == "" {
		ns, _, err = kubeConfig.Namespace()
		if err != nil || ns == "" {
			ns = "default"
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	return &Client{
		clientset:  clientset,
		restConfig: config,
		namespace:  ns,
		context:    kubeContext,
	}, nil
}

// Namespace returns the resolved namespace.
func (c *Client) Namespace() string {
	return c.namespace
}

// Close is a no-op for Kubernetes (no persistent connection to close).
func (c *Client) Close() error {
	return nil
}

// ListWorkloads lists pods in the namespace.
func (c *Client) ListWorkloads(ctx context.Context, scope string, filterNames []string) ([]*model.Container, error) {
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	nameSet := make(map[string]bool)
	for _, n := range filterNames {
		nameSet[n] = true
	}

	var result []*model.Container
	for i, pod := range pods.Items {
		name := pod.Name
		if len(nameSet) > 0 && !nameSet[name] {
			continue
		}

		status := mapPodStatus(string(pod.Status.Phase))
		result = append(result, &model.Container{
			ID:      string(pod.UID),
			Name:    name,
			Image:   podMainImage(pod),
			Status:  status,
			Color:   model.AssignColor(i),
			Visible: true,
		})
	}

	return result, nil
}

func mapPodStatus(phase string) model.ContainerStatus {
	switch phase {
	case "Running":
		return model.StatusRunning
	case "Succeeded", "Failed":
		return model.StatusExited
	default:
		return model.StatusStopped
	}
}

func podMainImage(pod corev1.Pod) string {
	if len(pod.Spec.Containers) > 0 {
		return pod.Spec.Containers[0].Image
	}
	return ""
}
