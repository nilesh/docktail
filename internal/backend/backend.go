package backend

import (
	"context"
	"errors"

	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/ui"
)

// LogMessage is sent over a channel when a new log line arrives.
type LogMessage struct {
	Entry *model.LogEntry
	Err   error
}

// ErrNotSupported indicates that an operation is not supported by this backend.
var ErrNotSupported = errors.New("operation not supported by this backend")

// Backend is the interface that Docker and Kubernetes backends implement.
type Backend interface {
	// ListWorkloads returns containers/pods for a given scope (project name or namespace).
	// If filterNames is non-empty, only those workloads are returned.
	ListWorkloads(ctx context.Context, scope string, filterNames []string) ([]*model.Container, error)

	// StreamLogs starts streaming logs for a workload and returns a channel of log messages.
	StreamLogs(ctx context.Context, workload *model.Container, since string) <-chan LogMessage

	// CreateExec starts an interactive shell session in a workload.
	CreateExec(ctx context.Context, workloadID string) (ui.ExecSession, error)

	// Lifecycle operations
	StartWorkload(ctx context.Context, id string) error
	StopWorkload(ctx context.Context, id string) error
	RestartWorkload(ctx context.Context, id string) error
	PauseWorkload(ctx context.Context, id string) error
	UnpauseWorkload(ctx context.Context, id string) error

	// Close cleans up backend resources.
	Close() error
}
