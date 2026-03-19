package model

// ContainerStatus represents the current state of a container.
type ContainerStatus string

const (
	StatusRunning ContainerStatus = "running"
	StatusPaused  ContainerStatus = "paused"
	StatusStopped ContainerStatus = "stopped"
	StatusExited  ContainerStatus = "exited"
)

// Container holds state for a single Docker container.
type Container struct {
	ID      string
	Name    string
	Image   string
	Status  ContainerStatus
	Color   string
	Visible bool // whether its logs are shown in the view
}

// ContainerColors is a palette of distinct colors for containers.
var ContainerColors = []string{
	"#22c55e", // green
	"#3b82f6", // blue
	"#f59e0b", // amber
	"#ef4444", // red
	"#a855f7", // purple
	"#06b6d4", // cyan
	"#f97316", // orange
	"#ec4899", // pink
	"#84cc16", // lime
	"#6366f1", // indigo
}

// AssignColor returns a color for a given container index.
func AssignColor(index int) string {
	return ContainerColors[index%len(ContainerColors)]
}
