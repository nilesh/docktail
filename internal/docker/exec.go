package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
)

// ExecSession represents an active exec session into a container.
type ExecSession struct {
	ExecID string
	Conn   types.HijackedResponse
}

// createExec starts an interactive shell session in a container.
// It probes for available shells, preferring bash over sh.
func (c *Client) createExec(ctx context.Context, containerID string) (*ExecSession, error) {
	shell := c.detectShell(ctx, containerID)

	execConfig := containerTypes.ExecOptions{
		Cmd:          []string{shell},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}

	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, err
	}

	attachResp, err := c.cli.ContainerExecAttach(ctx, execResp.ID, containerTypes.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return nil, err
	}

	return &ExecSession{
		ExecID: execResp.ID,
		Conn:   attachResp,
	}, nil
}

// detectShell probes the container for available shells by running a
// non-interactive exec. Returns /bin/bash if available, otherwise /bin/sh.
func (c *Client) detectShell(ctx context.Context, containerID string) string {
	resp, err := c.cli.ContainerExecCreate(ctx, containerID, containerTypes.ExecOptions{
		Cmd:          []string{"test", "-x", "/bin/bash"},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "/bin/sh"
	}

	if err := c.cli.ContainerExecStart(ctx, resp.ID, containerTypes.ExecStartOptions{}); err != nil {
		return "/bin/sh"
	}

	inspect, err := c.cli.ContainerExecInspect(ctx, resp.ID)
	if err != nil || inspect.ExitCode != 0 {
		return "/bin/sh"
	}

	return "/bin/bash"
}

// Write sends input to the exec session.
func (s *ExecSession) Write(data []byte) (int, error) {
	return s.Conn.Conn.Write(data)
}

// Reader returns a reader for the exec session output.
func (s *ExecSession) Reader() io.Reader {
	return s.Conn.Reader
}

// Close closes the exec session.
func (s *ExecSession) Close() error {
	s.Conn.Close()
	return nil
}

// ResizeExec resizes the TTY for an exec session.
func (c *Client) ResizeExec(execID string, height, width uint) error {
	return c.cli.ContainerExecResize(c.ctx, execID, containerTypes.ResizeOptions{
		Height: height,
		Width:  width,
	})
}
