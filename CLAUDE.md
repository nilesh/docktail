# Docktail — Project Memory

## What Is This

Docktail is a TUI application for monitoring Docker container and Kubernetes pod logs. It targets developers running multi-container Docker Compose setups or Kubernetes clusters who need a better alternative to `docker compose logs -f` or `kubectl logs -f`.

## Tech Stack

- **Go 1.22+** — language
- **Bubbletea v1** — TUI framework (Elm architecture)
- **Lipgloss** — TUI styling
- **Bubbles** — pre-built TUI components (key bindings, viewports)
- **Docker SDK (v27)** — `github.com/docker/docker` for container API
- **client-go** — `k8s.io/client-go` for Kubernetes pod/log/exec API
- **Cobra** — CLI argument parsing

## Project Structure

```
cmd/root.go            — CLI entry point, flag parsing, app bootstrap
internal/app/app.go    — Main Bubbletea model (Update/View/Init)
internal/app/keys.go   — All keybinding definitions
internal/backend/      — Backend interface (Docker/K8s abstraction)
internal/docker/       — Docker client wrapper (logs, exec, lifecycle)
internal/kube/         — Kubernetes client (pods, logs, exec, lifecycle)
internal/model/        — Data types (LogEntry, Container)
internal/theme/        — Color theme and lipgloss styles
internal/ui/           — UI components (sidebar, logview, shell, etc.)
```

## Build & Run

```bash
make build           # builds to bin/docktail
make run             # builds and runs
go build -o bin/docktail .  # direct build
./bin/docktail --version
```

## Key Design Decisions

1. **Freeze-first model** — log selection only works when the stream is frozen (`f` key). This avoids UX issues with selecting while logs scroll.

2. **Single-file app model** — The main Bubbletea model is in `app.go` with all Update/View logic. As it grows, UI components should be extracted to `internal/ui/` as sub-models.

3. **Docker SDK v27** — Using v27 (not v28+) because v28 changed to moby module paths which caused import issues. Pin to v27 for stability.

6. **Backend abstraction** — `internal/backend/backend.go` defines a `Backend` interface implemented by both Docker and Kubernetes clients. The app layer (`app.go`) only depends on this interface, so adding new backends requires zero UI changes.

4. **Log level detection** — Simple heuristic-based (scans first 50 chars for ERROR/WARN/INFO/DEBUG). Not structured log aware.

5. **Color assignment** — Containers get colors based on their index in the list. Colors are stable across the session but not persisted.

## Development Guidelines

- Keep the Bubbletea model clean. Each Update case should be short. Extract complex logic to helper methods.
- All keyboard shortcuts must be defined in `keys.go`, never hardcoded in Update.
- Mouse events go through `handleMouse()` in `app.go`.
- Docker operations that can block (start/stop/restart) must be done in Bubbletea Cmds (async), never in Update directly.
- Log buffer is capped at 5000 entries. Old entries are dropped from the front.
- Test with `docker compose up` in a real project. The app auto-detects the Compose project from the current directory.

## Current Status (v0.1.0)

Core architecture is in place:
- CLI with cobra, all flags defined
- Docker client with log streaming, exec, lifecycle management
- Bubbletea app with full keyboard handling, mouse support
- Sidebar, log view, shell panel, help overlay, search, level filtering
- Builds and produces a working binary

### What Needs Work

- [ ] Extract UI components from app.go into internal/ui/ sub-models
- [ ] Wire up real Docker exec for shell panel (currently mock)
- [ ] Add clipboard integration (atotto/clipboard or OSC52)
- [ ] Add shift+click range selection in mouse handler
- [ ] Add `--since` flag support to log streaming
- [ ] Add container resource usage display
- [ ] Add tests
- [ ] CI/CD with GitHub Actions
- [ ] Release workflow with goreleaser

## Spec

Full product spec is in `docs/SPEC.md`.

## Issues

Epic and story breakdowns are in `docs/issues/`.
