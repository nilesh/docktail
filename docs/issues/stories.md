# Stories

## E1: Core Log Streaming

### S1.1: Auto-detect Docker Compose project
**Priority:** P0
Detect compose.yml / docker-compose.yml in the current directory and use the directory name as the project name. Support `--project` flag override.
**Acceptance:** Running `docktail` in a dir with compose.yml starts streaming. Running with `--project foo` uses "foo".

### S1.2: Stream logs from all running containers
**Priority:** P0
Use Docker SDK `ContainerLogs` with `Follow: true` to stream logs from all running containers concurrently. Merge into a single time-ordered view.
**Acceptance:** Logs from all containers appear interleaved. New lines show up within 100ms.

### S1.3: Color-code container names
**Priority:** P0
Assign persistent colors to each container from a 10-color palette. Display the container name in its assigned color on each log line.
**Acceptance:** Each container has a distinct color that doesn't change during the session.

### S1.4: Parse and display timestamps
**Priority:** P0
Extract Docker log timestamps (RFC3339Nano). Display in HH:MM:SS.mmm format. Toggle with `t` key.
**Acceptance:** Timestamps show real Docker log times. Pressing `t` hides/shows them.

### S1.5: Support --since flag for historical logs
**Priority:** P1
Pass `--since` value to Docker `ContainerLogs` API. Support durations ("1h", "30m") and dates.
**Acceptance:** `docktail --since 1h` shows logs from the last hour on startup.

### S1.6: Support --containers flag for filtering
**Priority:** P1
Filter container list to only those specified in `--containers` flag.
**Acceptance:** `docktail --containers web,api` only shows logs from web and api containers.

---

## E2: Navigation & Selection

### S2.1: Freeze/unfreeze log stream
**Priority:** P0
`f` key toggles freeze. When frozen, log scrolling stops, line numbers appear, and a cursor is placed on the last line.
**Acceptance:** Pressing `f` stops scrolling, shows cursor. Pressing again resumes.

### S2.2: Keyboard cursor navigation
**Priority:** P0
When frozen: `↑/↓` or `j/k` move cursor. `g`/`G` jump to top/bottom. `PgUp`/`PgDn` scroll by 20 lines.
**Acceptance:** All navigation keys move the cursor and keep it in viewport.

### S2.3: Line selection (space, shift+arrow)
**Priority:** P0
`Space` toggles selection on the current line. `Shift+↑/↓` extends a range selection from an anchor point.
**Acceptance:** Selected lines are highlighted. Multiple non-contiguous lines can be selected.

### S2.4: Copy selected lines
**Priority:** P0
`y` or `c` copies selected lines to system clipboard. Include timestamp (if visible) and container name.
**Acceptance:** Copied text appears in system clipboard. A "✓ copied" indicator shows briefly.

### S2.5: Mouse click to move cursor
**Priority:** P0
Clicking a log line auto-freezes (if not frozen) and moves cursor to that line.
**Acceptance:** Click on any log line moves the cursor there.

### S2.6: Mouse shift+click range selection
**Priority:** P0
Shift+click selects a range from the current cursor to the clicked line.
**Acceptance:** Shift+clicking a line selects all lines between cursor and click target.

### S2.7: Mouse ctrl+click multi-select
**Priority:** P0
Ctrl+click toggles selection on the clicked line without affecting other selections.
**Acceptance:** Ctrl+clicking adds/removes individual lines from selection.

### S2.8: Scroll wheel support
**Priority:** P0
Mouse wheel scrolls the log view (when frozen) and shell panel.
**Acceptance:** Scrolling moves the viewport up/down.

---

## E3: Search & Filtering

### S3.1: Text search
**Priority:** P0
`/` enters search mode. Typed text filters log view to only matching lines. Matches highlighted inline.
**Acceptance:** Typing `/error` shows only lines containing "error", with "error" highlighted.

### S3.2: Regex search
**Priority:** P0
`Tab` in search mode toggles regex mode. Regex patterns are applied case-insensitive. Invalid regex shows error.
**Acceptance:** `/error.*timeout` in regex mode matches lines with "error" followed by "timeout".

### S3.3: Log level filtering
**Priority:** P0
`l` key cycles: ALL → ERROR → WARN → INFO → DEBUG. Filter applies to existing and new logs.
**Acceptance:** Setting filter to ERROR shows only error lines. Status bar shows active filter.

### S3.4: Jump between search matches
**Priority:** P1
When frozen with active search: `n` jumps to next match, `N` to previous.
**Acceptance:** Pressing `n` moves cursor to the next matching line.

---

## E4: Container Management

### S4.1: Container sidebar with status
**Priority:** P0
Left sidebar shows all containers with name, visibility indicator, and status icon (▸ running, ⏸ paused, ■ stopped).
**Acceptance:** Sidebar shows all project containers with correct status.

### S4.2: Toggle container visibility
**Priority:** P0
`Space` on sidebar toggles container log visibility. `a` toggles all. Click on container also toggles.
**Acceptance:** Toggling off a container immediately hides its logs from the view.

### S4.3: Container action menu
**Priority:** P0
`Enter` on sidebar opens a context menu with actions (start/stop/restart/pause depending on state). Right-click also opens it.
**Acceptance:** Menu shows valid actions for the container's current state. Selecting an action executes it.

### S4.4: Execute container actions
**Priority:** P0
Implement start, stop, restart, pause, unpause via Docker SDK. Show notification after action.
**Acceptance:** Stopping a container changes its icon and stops its log stream. Notification shown.

### S4.5: Focus cycling with Tab
**Priority:** P0
`Tab` cycles focus between sidebar, logs, and shell (if open). Active panel is visually indicated.
**Acceptance:** Tab moves focus. Active panel border/header changes color.

### S4.6: Click to focus panels
**Priority:** P0
Clicking on sidebar, log area, or shell panel sets focus to that panel.
**Acceptance:** Clicking any panel focuses it and updates visual indicator.

---

## E5: Integrated Shell

### S5.1: Open shell for a container
**Priority:** P0
`s` on sidebar or "Shell" from action menu opens a shell panel below logs. Uses `docker exec` with /bin/bash (fallback /bin/sh).
**Acceptance:** Shell panel appears with a working prompt for the selected container.

### S5.2: Shell input and output
**Priority:** P0
Type commands and see output. Support command history with ↑/↓.
**Acceptance:** Running `ls` in shell shows container filesystem. History navigable.

### S5.3: Shell tab bar and close
**Priority:** P0
Tab bar shows container name. `✕` button and `x` key close the shell. `Esc` returns focus to logs.
**Acceptance:** Shell tab shows which container. Clicking ✕ or pressing x closes it.

### S5.4: Resizable shell panel
**Priority:** P0
Drag the resize handle between logs and shell to adjust heights.
**Acceptance:** Mouse drag smoothly resizes the split.

---

## E6: Polish & Release

### S6.1: Clipboard integration
**Priority:** P0
Wire up system clipboard using `atotto/clipboard` or OSC52 escape sequences.
**Acceptance:** Copying lines in docktail makes them available in system paste.

### S6.2: Line wrap toggle
**Priority:** P0
`w` key toggles between truncated (with ellipsis) and wrapped display of long lines.
**Acceptance:** Long lines wrap when enabled. Status bar shows wrap state.

### S6.3: Help overlay
**Priority:** P0
`?` shows a modal with all keybindings grouped by context. `Esc` or `?` closes it.
**Acceptance:** Help shows all current shortcuts, organized clearly.

### S6.4: Add unit tests
**Priority:** P1
Test log parsing, level detection, container color assignment, search/filter logic.
**Acceptance:** `go test ./...` passes with >80% coverage on model and docker packages.

### S6.5: GitHub Actions CI
**Priority:** P1
Run tests, lint (golangci-lint), and build on push/PR.
**Acceptance:** CI badge in README. PRs blocked if tests fail.

### S6.6: Release with goreleaser
**Priority:** P1
Multi-platform binary releases (linux/mac/windows, amd64/arm64) via goreleaser on tag push.
**Acceptance:** `go install github.com/nilesh/docktail@latest` works. GitHub releases have binaries.

### S6.7: Homebrew formula
**Priority:** P2
Publish a Homebrew tap for `brew install docktail`.
**Acceptance:** `brew install nilesh/tap/docktail` installs the latest version.
