package docker

import (
	"context"

	"github.com/nilesh/docktail/internal/backend"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/ui"
)

// Ensure Client satisfies backend.Backend.
var _ backend.Backend = (*Client)(nil)

// ListWorkloads implements backend.Backend by listing Docker containers.
func (c *Client) ListWorkloads(ctx context.Context, scope string, filterNames []string) ([]*model.Container, error) {
	return c.ListContainers(scope, filterNames)
}

// StreamLogs implements backend.Backend by streaming Docker container logs.
func (c *Client) StreamLogs(ctx context.Context, container *model.Container, since string) <-chan backend.LogMessage {
	dockerCh := c.streamLogs(ctx, container, since)
	ch := make(chan backend.LogMessage, 64)
	go func() {
		defer close(ch)
		for msg := range dockerCh {
			ch <- backend.LogMessage{Entry: msg.Entry, Err: msg.Err}
		}
	}()
	return ch
}

// CreateExec implements backend.Backend by creating a Docker exec session.
func (c *Client) CreateExec(ctx context.Context, workloadID string) (ui.ExecSession, error) {
	return c.createExec(ctx, workloadID)
}

// StartWorkload implements backend.Backend.
func (c *Client) StartWorkload(ctx context.Context, id string) error {
	return c.StartContainer(id)
}

// StopWorkload implements backend.Backend.
func (c *Client) StopWorkload(ctx context.Context, id string) error {
	return c.StopContainer(id)
}

// RestartWorkload implements backend.Backend.
func (c *Client) RestartWorkload(ctx context.Context, id string) error {
	return c.RestartContainer(id)
}

// PauseWorkload implements backend.Backend.
func (c *Client) PauseWorkload(ctx context.Context, id string) error {
	return c.PauseContainer(id)
}

// UnpauseWorkload implements backend.Backend.
func (c *Client) UnpauseWorkload(ctx context.Context, id string) error {
	return c.UnpauseContainer(id)
}
