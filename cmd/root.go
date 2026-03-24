package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nilesh/docktail/internal/app"
	"github.com/nilesh/docktail/internal/backend"
	"github.com/nilesh/docktail/internal/docker"
	"github.com/nilesh/docktail/internal/kube"
	"github.com/nilesh/docktail/internal/theme"
	"github.com/nilesh/docktail/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"

	project     string
	containers  []string
	since       string
	timestamps  bool
	wrap        bool
	noColor     bool
	themeFlag   string
	kubeContext string
	namespace   string
)

var rootCmd = &cobra.Command{
	Use:   "docktail",
	Short: "A TUI for monitoring Docker container logs",
	Long:  "Docktail provides a terminal UI for streaming, searching, and managing Docker container logs across your Compose project.",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&project, "project", "p", "", "Docker Compose project name (default: auto-detect)")
	rootCmd.Flags().StringSliceVarP(&containers, "containers", "c", nil, "Specific containers/pods to monitor (default: all)")
	rootCmd.Flags().StringVarP(&since, "since", "s", "", "Show logs since timestamp (e.g., '1h', '2024-01-01')")
	rootCmd.Flags().BoolVarP(&timestamps, "timestamps", "t", true, "Show timestamps")
	rootCmd.Flags().BoolVarP(&wrap, "wrap", "w", false, "Wrap long lines")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colors")
	rootCmd.Flags().StringVar(&themeFlag, "theme", "auto", "Color theme: dark, light, auto (default: auto)")
	rootCmd.Flags().StringVar(&kubeContext, "kube-context", "", "Kubernetes context name (enables K8s mode)")
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default: from kubeconfig)")
	rootCmd.Version = version
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	theme.SetTheme(themeFlag)

	var be backend.Backend
	var scope string
	var err error

	if kubeContext != "" || namespace != "" {
		// Kubernetes mode — uses current context from kubeconfig if --kube-context not set
		k, kubeErr := kube.NewClient(kubeContext, namespace)
		if kubeErr != nil {
			return fmt.Errorf("failed to connect to Kubernetes: %w", kubeErr)
		}
		be = k
		scope = k.Namespace()
	} else {
		// Docker mode
		client, dockerErr := docker.NewClient()
		if dockerErr != nil {
			return fmt.Errorf("failed to connect to Docker: %w", dockerErr)
		}
		be = client

		scope = project
		if scope == "" {
			scope, err = docker.DetectProject()
			if err != nil {
				scope, err = pickProject(client)
				if err != nil {
					return err
				}
			}
		}
	}
	defer be.Close()

	containerList, err := be.ListWorkloads(cmd.Context(), scope, containers)
	if err != nil {
		return fmt.Errorf("failed to list workloads: %w", err)
	}

	if len(containerList) == 0 {
		label := "containers"
		if kubeContext != "" {
			label = "pods"
		}
		return fmt.Errorf("no %s found for %q", label, scope)
	}

	opts := app.Options{
		Project:    scope,
		Containers: containerList,
		Backend:    be,
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

func pickProject(client *docker.Client) (string, error) {
	projects, err := client.ListProjects()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return "", fmt.Errorf("no Docker Compose projects found.\nStart a project with 'docker compose up' or use --project to specify one")
	}

	if len(projects) == 1 {
		return projects[0], nil
	}

	picker := ui.NewPickerModel("Select a Docker Compose project:", projects)
	p := tea.NewProgram(picker)
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("picker error: %w", err)
	}

	m := result.(ui.PickerModel)
	if m.Quit || m.Selected == "" {
		return "", fmt.Errorf("no project selected")
	}

	return m.Selected, nil
}
