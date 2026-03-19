package docker

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/nilesh/docktail/internal/model"
)

// LogMessage is sent over a channel when a new log line arrives.
type LogMessage struct {
	Entry *model.LogEntry
	Err   error
}

// StreamLogs starts streaming logs for a container and sends them to the returned channel.
// It runs until the context is cancelled.
func (c *Client) StreamLogs(ctx context.Context, container *model.Container, since string) <-chan LogMessage {
	ch := make(chan LogMessage, 64)

	go func() {
		defer close(ch)

		opts := containerTypes.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Timestamps: true,
			Tail:       "100",
		}

		if since != "" {
			opts.Since = since
		}

		reader, err := c.cli.ContainerLogs(ctx, container.ID, opts)
		if err != nil {
			ch <- LogMessage{Err: err}
			return
		}
		defer reader.Close()

		// Docker multiplexes stdout/stderr with 8-byte headers.
		// Use stdcopy to demux, or if TTY is enabled, read directly.
		inspect, err := c.cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			ch <- LogMessage{Err: err}
			return
		}

		var logReader io.Reader
		if inspect.Config != nil && inspect.Config.Tty {
			logReader = reader
		} else {
			pr, pw := io.Pipe()
			go func() {
				_, _ = stdcopy.StdCopy(pw, pw, reader)
				pw.Close()
			}()
			logReader = pr
		}

		scanner := bufio.NewScanner(logReader)
		// Increase buffer for long log lines
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			entry := parseLine(line, container)
			ch <- LogMessage{Entry: entry}
		}
	}()

	return ch
}

// parseLine parses a Docker log line with timestamp prefix.
func parseLine(line string, container *model.Container) *model.LogEntry {
	entry := &model.LogEntry{
		Container: container,
		RawLine:   line,
		Timestamp: time.Now(),
	}

	// Docker timestamps look like: 2024-01-15T10:30:45.123456789Z
	// They're space-separated from the message.
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
