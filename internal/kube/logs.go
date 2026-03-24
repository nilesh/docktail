package kube

import (
	"bufio"
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nilesh/docktail/internal/backend"
	"github.com/nilesh/docktail/internal/model"
)

// StreamLogs streams logs for a pod and returns a channel of log messages.
func (c *Client) StreamLogs(ctx context.Context, workload *model.Container, since string) <-chan backend.LogMessage {
	ch := make(chan backend.LogMessage, 64)

	go func() {
		defer close(ch)

		opts := &corev1.PodLogOptions{
			Follow:     true,
			Timestamps: true,
		}

		if since != "" {
			if d, err := time.ParseDuration(since); err == nil {
				sinceSeconds := int64(d.Seconds())
				opts.SinceSeconds = &sinceSeconds
			} else if t, err := time.Parse(time.RFC3339, since); err == nil {
				metaSince := metav1FromTime(t)
				opts.SinceTime = &metaSince
			}
		} else {
			// Default: only show new logs from now
			now := metav1FromTime(time.Now())
			opts.SinceTime = &now
		}

		// Use pod name to get logs (Name holds the pod name)
		req := c.clientset.CoreV1().Pods(c.namespace).GetLogs(workload.Name, opts)
		stream, err := req.Stream(ctx)
		if err != nil {
			ch <- backend.LogMessage{Err: err}
			return
		}
		defer stream.Close()

		scanner := bufio.NewScanner(stream)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			entry := parseLine(line, workload)
			ch <- backend.LogMessage{Entry: entry}
		}
	}()

	return ch
}

// parseLine parses a Kubernetes log line with timestamp prefix.
// K8s timestamps use RFC3339Nano format, same as Docker.
func metav1FromTime(t time.Time) metav1.Time {
	return metav1.Time{Time: t}
}

func parseLine(line string, workload *model.Container) *model.LogEntry {
	entry := &model.LogEntry{
		Container: workload,
		RawLine:   line,
		Timestamp: time.Now(),
	}

	// K8s timestamps: 2024-01-15T10:30:45.123456789Z <message>
	if len(line) > 30 && line[4] == '-' && line[7] == '-' && line[10] == 'T' {
		spaceIdx := strings.IndexByte(line, ' ')
		if spaceIdx > 0 {
			ts, err := time.Parse(time.RFC3339Nano, line[:spaceIdx])
			if err == nil {
				entry.Timestamp = ts
				line = line[spaceIdx+1:]
			}
		}
	}

	entry.Message = line
	entry.Level = model.ParseLevel(line)

	return entry
}
