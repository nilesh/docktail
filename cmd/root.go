package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nilesh/docktail/internal/app"
	"github.com/nilesh/docktail/internal/docker"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"

	project    string
	containers []string
	since      string
	timestamps bool
	wrap       bool
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:   "docktail",
	Short: "A TUI for monitoring Docker container logs",
	Long:  "Docktail provides a terminal UI for streaming, searching, and managing Docker container logs across your Compose project.",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&project, "project", "p", "", "Docker Compose project name (default: auto-detect)")
	rootCmd.Flags().StringSliceVarP(&containers, "containers", "c", nil, "Specific containers to monitor (default: all)")
	rootCmd.Flags().StringVarP(&since, "since", "s", "", "Show logs since timestamp (e.g., '1h', '2024-01-01')")
	rootCmd.Flags().BoolVarP(&timestamps, "timestamps", "t", true, "Show timestamps")
	rootCmd.Flags().BoolVarP(&wrap, "wrap", "w", false, "Wrap long lines")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colors")
	rootCmd.Version = version
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer client.Close()

	projectName := project
	if projectName == "" {
		projectName, err = docker.DetectProject()
		if err != nil {
			return fmt.Errorf("could not detect Docker Compose project: %w\nUse --project to specify one", err)
		}
	}

	containerList, err := client.ListContainers(projectName, containers)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containerList) == 0 {
		return fmt.Errorf("no containers found for project %q", projectName)
	}

	opts := app.Options{
		Project:    projectName,
		Containers: containerList,
		Client:     client,
		Timestamps: timestamps,
		Wrap:       wrap,
		Since:      since,
	}

	m := app.New(opts)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running docktail: %v\n", err)
		os.Exit(1)
	}

	return nil
}
