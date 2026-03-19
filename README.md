# docktail

A TUI for monitoring Docker container logs.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

Docktail gives you a single-pane view of all container logs in your Docker Compose project, with search, filtering, selection, and an integrated shell.

## Features

- Stream logs from all containers in a Compose project
- Color-coded container names
- Freeze, navigate, select, and copy log lines
- Regex and text search with inline highlighting
- Log level filtering (ERROR, WARN, INFO, DEBUG)
- Toggle timestamps and line wrapping
- Container lifecycle management (start/stop/restart/pause)
- Integrated shell panel for running commands inside containers
- Full keyboard and mouse support

## Install

```bash
go install github.com/nilesh/docktail@latest
```

Or build from source:

```bash
git clone https://github.com/nilesh/docktail.git
cd docktail
make build
```

## Usage

```bash
# Auto-detect project from current directory
cd my-compose-project
docktail

# Specify project
docktail --project myapp

# Monitor specific containers
docktail --containers web,api,db

# Show logs from last hour
docktail --since 1h
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `f` | Freeze/unfreeze logs |
| `t` | Toggle timestamps |
| `w` | Toggle line wrap |
| `l` | Cycle log level filter |
| `/` | Search (Tab for regex) |
| `Tab` | Cycle focus (sidebar/logs/shell) |
| `?` | Help overlay |
| `q` | Quit |

**Sidebar:** `↑/↓` navigate, `Space` toggle, `Enter` actions, `s` shell, `a` all

**Logs (frozen):** `↑/↓` cursor, `Space` select, `Shift+↑/↓` range, `y` copy

## License

MIT
