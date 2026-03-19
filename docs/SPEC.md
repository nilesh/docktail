# Docktail — Product Requirements Document

## Overview

Docktail is a terminal user interface (TUI) application for monitoring Docker container logs in real-time. It provides a unified view of logs across all containers in a Docker Compose project, with rich navigation, filtering, search, and container management capabilities.

## Problem Statement

Developers working with multi-container Docker setups (typically via Docker Compose) struggle to monitor logs across services. The current workflow involves either running `docker compose logs -f` which produces an unmanageable interleaved stream, or opening multiple terminal tabs with `docker logs -f <container>` per service. Neither approach supports searching, filtering, selecting, or copying specific log lines efficiently. There is no good way to quickly jump between log inspection and container shell access without switching windows.

## Goals

1. Provide a single-pane view of logs from all (or selected) containers in a Docker Compose project
2. Support freezing, navigating, selecting, and copying log lines with keyboard shortcuts and mouse
3. Enable real-time log level filtering and regex search across the log stream
4. Allow container lifecycle management (start, stop, restart, pause) without leaving the TUI
5. Provide an integrated shell panel for executing commands inside containers

## Non-Goals

1. **Log persistence / storage** — Docktail is a live viewer, not a log aggregation system. Use ELK/Loki for that.
2. **Remote Docker hosts** — V1 targets local Docker daemon only. Remote support is a future consideration.
3. **Kubernetes support** — This is a Docker-specific tool. Stern/k9s cover Kubernetes.
4. **Log alerting / notifications** — No automated alerting. This is an interactive monitoring tool.
5. **Custom log parsing / structured log rendering** — V1 treats logs as plain text lines.

## Target Users

- Backend developers running multi-service applications via Docker Compose
- DevOps engineers debugging containerized services locally
- Full-stack developers who need to monitor frontend, API, database, and worker containers simultaneously

## Technology Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Language | Go 1.22+ | Docker SDK is first-class in Go. Single binary distribution. Excellent concurrency for streaming. |
| TUI Framework | Bubbletea v2 | Elm-architecture TUI framework with built-in keyboard and mouse support. Most mature Go TUI framework. |
| TUI Styling | Lipgloss | Companion styling library for Bubbletea. Handles colors, borders, padding. |
| TUI Components | Bubbles | Pre-built viewport, text input, spinner, and table components. |
| Docker API | docker/docker client SDK | Official Go SDK for Docker Engine API. Container logs, exec, lifecycle management. |
| Build | Go modules | Standard Go dependency management. |
| CLI Args | cobra | Industry standard for Go CLI argument parsing. |

## Architecture

```
docktail
├── cmd/                    # CLI entry point (cobra commands)
│   └── root.go
├── internal/
│   ├── app/                # Main application (Bubbletea model, update, view)
│   │   ├── app.go          # Root model combining all panels
│   │   ├── keys.go         # Key bindings
│   │   └── mouse.go        # Mouse event handling
│   ├── docker/             # Docker client wrapper
│   │   ├── client.go       # Docker API client
│   │   ├── logs.go         # Log streaming
│   │   ├── exec.go         # Container exec (shell)
│   │   └── lifecycle.go    # Start/stop/restart/pause
│   ├── ui/                 # UI components
│   │   ├── sidebar.go      # Container list sidebar
│   │   ├── logview.go      # Main log viewport
│   │   ├── shell.go        # Shell panel
│   │   ├── search.go       # Search bar
│   │   ├── statusbar.go    # Bottom status bar
│   │   ├── titlebar.go     # Top title bar
│   │   ├── actionmenu.go   # Container action popup menu
│   │   └── help.go         # Help overlay
│   ├── model/              # Data models
│   │   ├── log.go          # Log entry struct
│   │   └── container.go    # Container state struct
│   └── theme/              # Colors, styles
│       └── theme.go
├── docs/                   # Documentation
├── CLAUDE.md               # Project memory for AI-assisted development
├── go.mod
├── go.sum
├── main.go                 # Entry point
├── Makefile
├── LICENSE
└── README.md
```

## Features

### F1: Multi-Container Log Streaming

Stream logs from all running containers in a Docker Compose project into a single, interleaved view. Each container's logs are color-coded with a persistent color assignment.

**Requirements:**
- P0: Auto-detect Docker Compose project from current directory (docker-compose.yml / compose.yml)
- P0: Stream logs from all running containers concurrently
- P0: Assign unique colors per container, consistent across restarts
- P0: Display container name prefix on each log line
- P1: Allow specifying project name via `--project` flag
- P1: Allow specifying individual containers via `--containers` flag
- P1: Support `--since` flag to load historical logs on startup

**Acceptance Criteria:**
- Launching `docktail` in a directory with compose.yml streams all container logs
- Each container has a visually distinct color
- New log lines appear in real-time with <100ms latency from Docker event

### F2: Timestamps

Display timestamps for each log line, with the ability to toggle visibility.

**Requirements:**
- P0: Show timestamps in HH:MM:SS.mmm format
- P0: Toggle timestamps on/off with `t` key
- P1: Support multiple timestamp formats via `--time-format` flag
- P1: Show relative timestamps option ("2s ago")

**Acceptance Criteria:**
- Pressing `t` toggles timestamp column on and off
- Timestamps reflect the actual Docker log timestamp, not arrival time

### F3: Freeze, Navigate, Select, Copy

Freeze the log stream to enable cursor navigation, line selection, and clipboard copy.

**Requirements:**
- P0: `f` key freezes/unfreezes the log stream
- P0: When frozen, show line numbers and cursor
- P0: Arrow keys / j/k navigate the cursor
- P0: `Space` toggles selection on current line
- P0: `Shift+Arrow` for range selection
- P0: `y` or `c` copies selected lines to clipboard
- P0: `g`/`G` jump to top/bottom
- P0: `PgUp`/`PgDn` for page navigation
- P0: Mouse click on a log line moves cursor and auto-freezes
- P0: Mouse click+drag or Shift+click for range selection
- P0: Ctrl+click for multi-select individual lines
- P1: `Esc` clears selection

**Acceptance Criteria:**
- Pressing `f` stops log scrolling and shows a cursor on the last line
- Selected lines are visually highlighted
- Copied text includes timestamp (if visible), container name, and message
- Mouse clicks on log lines move the cursor and freeze if not already frozen

### F4: Log Level Filtering

Filter visible logs by severity level.

**Requirements:**
- P0: `l` key cycles through filter levels: ALL → ERROR → WARN → INFO → DEBUG
- P0: Filtered lines are hidden, not removed from buffer
- P1: Visual indicator in status bar showing active filter
- P2: Custom level definitions via config file

**Acceptance Criteria:**
- Setting filter to ERROR shows only ERROR-level lines
- Changing filter back to ALL restores all lines
- Filter applies to both existing and new incoming logs

### F5: Search (Text and Regex)

Search within log messages with plain text or regex patterns.

**Requirements:**
- P0: `/` enters search mode
- P0: Tab toggles between plain text and regex mode
- P0: Matching text is highlighted inline
- P0: Search filters log view to only matching lines
- P0: Show match count in status bar
- P0: Invalid regex shows error indicator
- P1: `n`/`N` to jump between matches (when frozen)
- P2: Search history (up arrow in search mode)

**Acceptance Criteria:**
- Typing `/error.*timeout` in regex mode shows only lines matching that pattern
- Matches are highlighted in orange within each line
- Invalid regex shows "invalid regex" error without crashing

### F6: Container Sidebar

Left panel showing all project containers with their status, allowing toggle and actions.

**Requirements:**
- P0: Show container name, status icon (running/paused/stopped)
- P0: `Tab` cycles focus between sidebar, logs, shell
- P0: `Space` toggles container log visibility
- P0: `Enter` opens action menu
- P0: `a` selects/deselects all containers
- P0: Click on container name to toggle log visibility
- P0: Right-click on container for action menu
- P0: Click on sidebar area to focus it
- P1: Show container image name on hover/expansion
- P1: Show resource usage (CPU/memory) per container

**Acceptance Criteria:**
- Toggling a container off immediately stops showing its logs
- Action menu appears next to the container with valid actions for its state

### F7: Container Actions

Manage container lifecycle from within the TUI.

**Requirements:**
- P0: Start a stopped container
- P0: Stop a running container
- P0: Restart a running container
- P0: Pause/unpause a container
- P0: Actions available via keyboard (Enter on sidebar) and mouse (right-click)
- P1: Confirmation prompt for destructive actions (stop)
- P1: Show notification toast after action completes

**Acceptance Criteria:**
- Stopping a container changes its status icon and stops its log stream
- Starting a container begins streaming its logs
- Restarting a container briefly shows stopped then running status

### F8: Integrated Shell

Open a shell session into any running container in a panel below the log view.

**Requirements:**
- P0: `s` on sidebar opens shell for focused container
- P0: Shell panel appears below logs with resizable divider
- P0: Shell tab bar shows which container is attached
- P0: `x` key closes shell panel
- P0: Click on shell area to focus it
- P0: Click ✕ button on shell tab to close
- P0: Drag resize handle to adjust shell height
- P0: `Esc` in shell returns focus to logs
- P0: Shell supports basic command input and output
- P1: Multiple shell tabs (one per container)
- P1: Shell command history (up arrow)
- P2: Shell auto-complete

**Acceptance Criteria:**
- Opening shell for a container runs `docker exec -it <container> /bin/sh` (or /bin/bash if available)
- Commands execute and output is displayed in the panel
- Closing shell terminates the exec session
- Mouse drag on resize handle smoothly resizes the panel

### F9: Line Wrapping

Toggle between truncated and wrapped display of long log lines.

**Requirements:**
- P0: `w` key toggles line wrapping
- P0: When off (default), long lines truncate with ellipsis
- P0: When on, lines wrap to fill viewport width
- P1: Wrap indicator in status bar

**Acceptance Criteria:**
- Long log lines that exceed viewport width are truncated by default
- Pressing `w` wraps them, increasing the visual height of those lines

### F10: Mouse Support

Full mouse interaction across all panels.

**Requirements:**
- P0: Click on sidebar to focus sidebar
- P0: Click on log area to focus logs
- P0: Click on shell area to focus shell
- P0: Click on log line to move cursor (auto-freezes)
- P0: Click on container in sidebar to toggle log visibility
- P0: Right-click on container for action menu
- P0: Click ✕ on shell tab to close shell
- P0: Drag shell resize handle
- P0: Shift+click on log line for range selection
- P0: Ctrl/Cmd+click for multi-select
- P0: Scroll wheel in log view and shell
- P1: Double-click a log line to copy it
- P1: Hover effects on interactive elements

**Acceptance Criteria:**
- All focus switches work via mouse click
- Right-click context menu appears at cursor position
- Mouse selection works identically to keyboard selection

### F11: Help Overlay

Show all keyboard shortcuts in a modal overlay.

**Requirements:**
- P0: `?` toggles help overlay
- P0: Grouped by section (General, Sidebar, Logs, Shell)
- P0: `Esc` closes help
- P1: Click outside help to close

**Acceptance Criteria:**
- Help overlay shows all current keybindings organized by context

## CLI Interface

```
docktail [flags]

Flags:
  -p, --project string       Docker Compose project name (default: auto-detect)
  -c, --containers strings   Specific containers to monitor (default: all)
  -s, --since string         Show logs since timestamp (e.g., "1h", "2024-01-01")
  -f, --follow               Follow log output (default: true)
  -t, --timestamps           Show timestamps (default: true)
  -w, --wrap                 Wrap long lines (default: false)
  --no-color                 Disable colors
  --theme string             Color theme (default: "dark")
  -h, --help                 Help for docktail
  -v, --version              Version info
```

## Success Metrics

**Leading indicators:**
- First-run success rate: user can see logs from their project within 5 seconds of launching
- Keyboard shortcut discoverability: user finds and uses freeze+copy within first session

**Lagging indicators:**
- GitHub stars > 500 within 6 months
- Regular usage (daily active users reporting via opt-in telemetry)
- Featured in Docker/Go community newsletters

## Open Questions

1. **[Engineering]** Should we support Docker Swarm services in addition to Compose?
2. **[Engineering]** How to handle containers that produce extremely high log throughput (>10k lines/sec)? Ring buffer size?
3. **[Design]** Should container colors be configurable, or always auto-assigned?
4. **[Engineering]** Support for podman as an alternative container runtime?

## Timeline

**Phase 1 (V1):** Core log streaming, navigation, search, container sidebar, basic shell — this spec.
**Phase 2 (V2):** Multiple shell tabs, log export, custom themes, container resource monitoring.
**Phase 3 (V3):** Remote Docker hosts, podman support, plugin system.
