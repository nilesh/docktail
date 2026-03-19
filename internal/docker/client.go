package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/nilesh/docktail/internal/model"
)

// Client wraps the Docker API client.
type Client struct {
	cli *client.Client
	ctx context.Context
}

// NewClient creates a new Docker client connected to the local daemon.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	ctx := context.Background()

	// Verify connection
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Docker daemon: %w", err)
	}

	return &Client{cli: cli, ctx: ctx}, nil
}

// Close closes the Docker client.
func (c *Client) Close() error {
	return c.cli.Close()
}

// Inner returns the underlying Docker client for direct API access.
func (c *Client) Inner() *client.Client {
	return c.cli
}

// Context returns the client's context.
func (c *Client) Context() context.Context {
	return c.ctx
}

// ListContainers returns containers for a given Compose project.
// If filterNames is non-empty, only those containers are returned.
func (c *Client) ListContainers(project string, filterNames []string) ([]*model.Container, error) {
	f := filters.NewArgs()
	f.Add("label", "com.docker.compose.project="+project)

	containers, err := c.cli.ContainerList(c.ctx, containerTypes.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	nameSet := make(map[string]bool)
	for _, n := range filterNames {
		nameSet[n] = true
	}

	var result []*model.Container
	for i, ct := range containers {
		name := cleanContainerName(ct.Names)
		if len(nameSet) > 0 && !nameSet[name] {
			continue
		}

		status := mapStatus(ct.State)
		result = append(result, &model.Container{
			ID:      ct.ID,
			Name:    name,
			Image:   ct.Image,
			Status:  status,
			Color:   model.AssignColor(i),
			Visible: true,
		})
	}

	return result, nil
}

// StartContainer starts a stopped container.
func (c *Client) StartContainer(id string) error {
	return c.cli.ContainerStart(c.ctx, id, containerTypes.StartOptions{})
}

// StopContainer stops a running container.
func (c *Client) StopContainer(id string) error {
	return c.cli.ContainerStop(c.ctx, id, containerTypes.StopOptions{})
}

// RestartContainer restarts a container.
func (c *Client) RestartContainer(id string) error {
	return c.cli.ContainerRestart(c.ctx, id, containerTypes.StopOptions{})
}

// PauseContainer pauses a running container.
func (c *Client) PauseContainer(id string) error {
	return c.cli.ContainerPause(c.ctx, id)
}

// UnpauseContainer unpauses a paused container.
func (c *Client) UnpauseContainer(id string) error {
	return c.cli.ContainerUnpause(c.ctx, id)
}

// InspectContainer returns the current state of a container.
func (c *Client) InspectContainer(id string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(c.ctx, id)
}

func cleanContainerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}
	name := names[0]
	// Docker prepends "/" to container names
	return strings.TrimPrefix(name, "/")
}

func mapStatus(state string) model.ContainerStatus {
	switch state {
	case "running":
		return model.StatusRunning
	case "paused":
		return model.StatusPaused
	case "exited", "dead":
		return model.StatusExited
	default:
		return model.StatusStopped
	}
}

// DetectProject tries to find a Compose project name from the current directory.
func DetectProject() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check for compose files
	composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, f := range composeFiles {
		if _, err := os.Stat(filepath.Join(cwd, f)); err == nil {
			// Use directory name as project name (Docker Compose default behavior)
			return filepath.Base(cwd), nil
		}
	}

	return "", fmt.Errorf("no docker-compose.yml or compose.yml found in %s", cwd)
}
