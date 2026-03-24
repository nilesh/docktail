package kube

import (
	"context"
	"fmt"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/nilesh/docktail/internal/ui"
)

// execSession wraps a K8s exec connection to satisfy ui.ExecSession.
type execSession struct {
	stdinWriter  io.WriteCloser
	stdoutReader io.ReadCloser
}

func (s *execSession) Write(data []byte) (int, error) {
	return s.stdinWriter.Write(data)
}

func (s *execSession) Reader() io.Reader {
	return s.stdoutReader
}

func (s *execSession) Close() error {
	s.stdinWriter.Close()
	s.stdoutReader.Close()
	return nil
}

// CreateExec starts an interactive shell in a pod.
func (c *Client) CreateExec(ctx context.Context, workloadID string) (ui.ExecSession, error) {
	// Find pod name by UID
	podName, err := c.podNameByID(ctx, workloadID)
	if err != nil {
		return nil, err
	}

	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(c.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"/bin/sh"},
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(c.restConfig, http.MethodPost, req.URL())
	if err != nil {
		return nil, err
	}

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go func() {
		_ = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  stdinR,
			Stdout: stdoutW,
			Stderr: stdoutW,
			Tty:    true,
		})
		stdoutW.Close()
		stdinR.Close()
	}()

	return &execSession{
		stdinWriter:  stdinW,
		stdoutReader: stdoutR,
	}, nil
}

// podNameByID finds a pod name from its UID.
func (c *Client) podNameByID(ctx context.Context, uid string) (string, error) {
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, pod := range pods.Items {
		if string(pod.UID) == uid {
			return pod.Name, nil
		}
	}
	return "", fmt.Errorf("pod with UID %s not found", uid)
}
