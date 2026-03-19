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

// CreateExec starts an interactive shell session in a container.
// It tries /bin/bash first, then falls back to /bin/sh.
func (c *Client) CreateExec(ctx context.Context, containerID string) (*ExecSession, error) {
	shells := []string{"/bin/bash", "/bin/sh"}

	var lastErr error
	for _, shell := range shells {
		execConfig := containerTypes.ExecOptions{
			Cmd:          []string{shell},
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
		}

		execResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			lastErr = err
			continue
		}

		attachResp, err := c.cli.ContainerExecAttach(ctx, execResp.ID, containerTypes.ExecAttachOptions{
			Tty: true,
		})
		if err != nil {
			lastErr = err
			continue
		}

		return &ExecSession{
			ExecID: execResp.ID,
			Conn:   attachResp,
		}, nil
	}

	return nil, lastErr
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
