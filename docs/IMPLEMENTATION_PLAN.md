# Docktail — Implementation Plan for Claude Code

## Context

This is a Go TUI application for monitoring Docker container logs. The scaffolding is complete — the project compiles, produces a binary (`bin/docktail`), and has the core architecture in place. But much of the logic in `app.go` is a first pass that needs hardening, and several features have stub implementations that need to be wired to real Docker APIs.

Read `CLAUDE.md` for project memory and `docs/SPEC.md` for the full product spec before starting any work.

## Current State

**What works:**
- `go build` produces a binary, `docktail --version` prints `0.1.0`
- CLI flags defined via cobra (`--project`, `--containers`, `--since`, `--timestamps`, `--wrap`)
- Docker client connects, lists containers by Compose project label, streams logs
- Bubbletea app model with full keyboard/mouse event handling framework
- All key bindings defined in `keys.go`
- Theme system with dark theme in `theme/theme.go`

**What's stubbed or incomplete:**
- Shell panel has mock input handling — not wired to real `docker exec`
- Clipboard copy (`copySelected()`) builds the string but doesn't write to system clipboard
- Mouse shift+click and ctrl+click modifiers aren't implemented in `handleMouse()`
- `app.go` is 1231 lines — needs to be split into sub-model files under `internal/ui/`
- No tests exist anywhere
- Log stream restart after container lifecycle actions isn't handled
- Action menu doesn't render as an overlay (it's a state flag but no visual)
- The `strings.Title` call in `executeContainerAction` is deprecated

## Dependencies

```
github.com/charmbracelet/bubbletea v1.2.4
github.com/charmbracelet/bubbles v0.20.0
github.com/charmbracelet/lipgloss v1.0.0
github.com/docker/docker v27.5.1+incompatible
github.com/spf13/cobra v1.10.2
```

Additional dependencies to add during implementation:
- `github.com/atotto/clipboard` — system clipboard access
- `golang.design/x/clipboard` — alternative if atotto doesn't work on target OS

## Implementation Phases

Execute these phases in order. Each phase should end with `go build ./...` passing and manual testing against a real Docker Compose project.

---

### Phase 1: Refactor app.go into UI sub-models

**Goal:** Split the 1231-line `app.go` into focused sub-models that compose together.

**Files to create:**

1. **`internal/ui/sidebar.go`** — Sidebar model
   - Extract sidebar rendering (`renderSidebar`) and sidebar key handling (`handleSidebarKey`)
   - Struct: `SidebarModel` with `containers`, `cursor`, `focused` fields
   - Methods: `Update(msg) (SidebarModel, tea.Cmd)`, `View() string`
   - Handle: container toggling, navigation, action menu trigger

2. **`internal/ui/logview.go`** — Log viewport model
   - Extract log rendering (`renderLogView`) and log key handling (`handleLogKey`)
   - Struct: `LogViewModel` with `logs`, `filteredLogs`, `cursor`, `selectedLines`, `frozen`, `wrapLines`, `showTimestamps` fields
   - Methods: `Update`, `View`, `Refilter`, `CopySelected`
   - Handle: freeze/unfreeze, cursor movement, selection, viewport scrolling

3. **`internal/ui/shell.go`** — Shell panel model
   - Extract shell rendering (`renderShellPanel`) and key handling (`handleShellKey`)
   - Struct: `ShellModel` with `container`, `lines`, `input`, `cmdHistory`, `height`, `execSession` fields
   - Methods: `Update`, `View`, `Open(container)`, `Close()`

4. **`internal/ui/search.go`** — Search bar model
   - Extract search handling (`handleSearchKey`, `updateSearchRegex`)
   - Struct: `SearchModel` with `query`, `regex`, `isRegexMode`, `active`, `error` fields
   - Methods: `Update`, `Matches(msg string) bool`, `HighlightView(text string) string`

5. **`internal/ui/titlebar.go`** — Title bar renderer
   - Extract `renderTitleBar` into a standalone view function

6. **`internal/ui/statusbar.go`** — Status bar renderer
   - Extract `renderStatusBar` into a standalone view function

7. **`internal/ui/help.go`** — Help overlay model
   - Extract `renderHelp` and help key handling
   - Struct: `HelpModel` with `visible` field

8. **`internal/ui/actionmenu.go`** — Action menu overlay model
   - Extract action menu state, rendering, and key handling
   - Struct: `ActionMenuModel` with `open`, `actions`, `cursor` fields
   - Must render as an overlay on top of the sidebar

**Refactored `app.go` should:**
- Compose all sub-models as fields
- Route messages to the focused sub-model
- Be under 300 lines

**Validation:**
```bash
go build ./...
go vet ./...
```

---

### Phase 2: Wire real Docker exec for shell

**Goal:** Shell panel runs actual commands inside the container via `docker exec`.

**Changes to `internal/docker/exec.go`:**
- The `CreateExec` function already creates a hijacked connection — it works
- Add a `ResizeExec(execID, height, width)` method for terminal resize

**Changes to `internal/ui/shell.go`:**
- On `Open(container)`, call `docker.CreateExec()` to get an `ExecSession`
- Start a goroutine that reads from `ExecSession.Reader()` and sends lines as Bubbletea messages
- On key input, write to `ExecSession.Write()` instead of mock handling
- On `Close()`, call `ExecSession.Close()` to terminate the exec
- Handle raw terminal mode: send each keypress as a byte, not as a whole command
- Define a `ShellOutputMsg` Bubbletea message type for async output

**Key design point:** The shell must operate in raw mode since we're attached to a PTY. Each keypress (including Enter, Ctrl+C, arrow keys) must be translated to the correct byte sequence and written to the exec connection. Do NOT accumulate input as a string and send on Enter — that's the mock behavior.

**Testing:**
- Start a `docker compose up` project
- Launch docktail, press `s` on a container
- Type `ls`, `ps aux`, `cat /etc/hostname` and verify output
- Press `Ctrl+C` in shell — should not kill docktail
- Close shell with `x` — verify exec session is cleaned up

---

### Phase 3: Clipboard integration

**Goal:** `y` key copies selected log lines to system clipboard.

**Add dependency:**
```bash
go get github.com/atotto/clipboard
```

**Changes to `internal/ui/logview.go`:**
- In `CopySelected()`, after building the text string, call `clipboard.WriteAll(text)`
- Handle the error (some headless environments don't have clipboard access)
- If `clipboard.WriteAll` fails, try OSC52 escape sequence as fallback:
  ```go
  // OSC52: \033]52;c;<base64-encoded-text>\007
  fmt.Fprintf(os.Stdout, "\033]52;c;%s\007", base64.StdEncoding.EncodeToString([]byte(text)))
  ```

**Validation:** Copy lines, paste into another terminal or editor.

---

### Phase 4: Complete mouse interaction

**Goal:** Mouse shift+click, ctrl+click, right-click, resize drag all work.

**Changes to `internal/app/app.go` (or a new `internal/app/mouse.go`):**

The current `handleMouse` handles basic left-click and right-click. Extend it:

1. **Shift+click range selection on log lines:**
   - Bubbletea v1 `tea.MouseMsg` doesn't directly expose modifier keys via the Msg struct. However, `tea.MouseMsg` has `Shift`, `Alt`, `Ctrl` boolean fields in newer versions.
   - Check `msg.Shift` — if true and `msg.Button == tea.MouseButtonLeft`, calculate range from current cursor to clicked line index, set `selectedLines` for all lines in range.

2. **Ctrl+click multi-select:**
   - Check `msg.Ctrl` — if true, toggle the clicked line in `selectedLines` without clearing existing selection.

3. **Shell resize drag:**
   - On `tea.MouseButtonLeft` press (not release) on the resize handle row, set a `resizing` flag
   - On `tea.MouseMotion` while `resizing` is true, update `shellHeight` based on `msg.Y`
   - On `tea.MouseButtonLeft` release, clear `resizing` flag
   - The resize handle is the 1px row between log view and shell tab bar

4. **Right-click context menu positioning:**
   - Currently opens action menu with `actionMenuOpen = true`. Need to also store click coordinates so the menu renders near the click position.

5. **Double-click to copy single line:**
   - Detect double-click (two left-click releases within 300ms on same line)
   - Copy that single line to clipboard

**Validation:** Test all interactions with a mouse in a real terminal.

---

### Phase 5: Fix log stream lifecycle

**Goal:** When a container is started/restarted, automatically begin streaming its logs. When stopped, clean up.

**Current problem:** `startLogStreams()` is called once at startup. After a container action (stop/start/restart), the log stream for that container is stale.

**Solution:**

1. **Track per-container stream contexts:**
   ```go
   type containerStream struct {
       cancel context.CancelFunc
       active bool
   }
   ```
   Store a `map[string]*containerStream` keyed by container ID.

2. **On container start/restart:**
   - Create a new context for that container
   - Start `StreamLogs` with that context
   - Feed into the shared `logCh`

3. **On container stop:**
   - Cancel the context for that container's stream
   - Mark stream as inactive

4. **On container pause:**
   - Logs stop arriving naturally (Docker pauses the process), no need to cancel stream
   - But UI should indicate paused state

5. **Refactor `startLogStreams` → `startStreamForContainer(c *model.Container)`:**
   - Called per-container instead of bulk
   - Returns a `tea.Cmd` that feeds log messages

**Files changed:** `internal/app/app.go`, `internal/docker/logs.go`

---

### Phase 6: Action menu overlay rendering

**Goal:** The action menu renders visually as a floating popup next to the sidebar container.

**Current problem:** `actionMenuOpen` is a boolean flag but `View()` doesn't render the menu overlay.

**Changes to `internal/ui/actionmenu.go`:**
- Render a bordered box with action items
- Position it at column `sidebarWidth + 1`, row `titleBarHeight + 1 + sidebarCursor`
- Use lipgloss `Place` or manual string manipulation to overlay on top of the log view
- Highlight the currently focused action item
- Show keyboard hints (Enter to select, Esc to close)

**Rendering approach:** After composing the main layout string, use ANSI cursor positioning to draw the menu on top. Or, use lipgloss overlay technique:
```go
// Convert rendered view to a 2D grid, stamp the menu onto it, re-serialize
```

This is the trickiest visual piece. Bubbletea doesn't have a built-in overlay system, so you'll need to:
1. Render the full layout as a string
2. Split it into lines
3. Replace characters at the menu position with the menu content
4. Re-join and return

---

### Phase 7: Fix deprecations and edge cases

1. **Replace `strings.Title`** in `executeContainerAction` with `cases.Title(language.English).String()` from `golang.org/x/text`

2. **Handle terminal resize** — when `tea.WindowSizeMsg` arrives, recalculate shell height bounds and clamp

3. **Handle Docker daemon disconnect** — if Docker goes away, show an error in the status bar instead of crashing

4. **Handle empty project** — if no containers match, show a helpful empty state instead of crashing

5. **Log buffer memory** — the buffer is capped at 5000 but `refilter()` is called on every new log, rebuilding `filteredLogs` from scratch. For performance:
   - Only append to `filteredLogs` if the new entry passes the filter
   - Full rebuild only when filter criteria change

6. **Fix go.mod toolchain** — the module says `go 1.25.0` because of transitive deps. Pin `toolchain go1.22.1` explicitly to avoid forcing users to have Go 1.25.

---

### Phase 8: Tests

**Goal:** Core logic has test coverage.

**Files to create:**

1. **`internal/model/log_test.go`**
   - Test `ParseLevel` with various log formats (structured, unstructured, mixed case)
   - Test edge cases: empty string, very short strings, binary content

2. **`internal/model/container_test.go`**
   - Test `AssignColor` wraps around after 10 containers

3. **`internal/ui/search_test.go`**
   - Test text search matching (case insensitive)
   - Test regex search (valid patterns, invalid patterns)
   - Test `HighlightView` produces correct segments

4. **`internal/ui/logview_test.go`**
   - Test `Refilter` with level filter + search + visibility combinations
   - Test `CopySelected` output format

5. **`internal/docker/logs_test.go`**
   - Test `parseLine` with Docker timestamp format
   - Test with and without timestamps
   - Mock reader for `StreamLogs`

**Run:** `go test ./... -v -count=1`

---

### Phase 9: CI and release

1. **`.github/workflows/ci.yml`**
   ```yaml
   name: CI
   on: [push, pull_request]
   jobs:
     test:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-go@v5
           with:
             go-version: '1.22'
         - run: go build ./...
         - run: go test ./... -v
         - run: go vet ./...
   ```

2. **`.github/workflows/release.yml`** with goreleaser on tag push

3. **`.goreleaser.yml`** for multi-platform builds (linux/mac/windows × amd64/arm64)

---

## Order of execution

| Phase | Effort | Depends on |
|-------|--------|-----------|
| 1. Refactor app.go | Large (main structural change) | Nothing |
| 2. Wire docker exec | Medium | Phase 1 (shell sub-model) |
| 3. Clipboard | Small | Phase 1 (logview sub-model) |
| 4. Mouse interaction | Medium | Phase 1 |
| 5. Log stream lifecycle | Medium | Phase 1 |
| 6. Action menu overlay | Medium | Phase 1 |
| 7. Fix deprecations | Small | Phase 1 |
| 8. Tests | Medium | Phases 1-7 |
| 9. CI and release | Small | Phase 8 |

Phases 2-7 can be done in parallel after Phase 1 is complete. Phase 8 should come after all feature work. Phase 9 last.

## Testing approach

There's no Docker available in CI for integration tests. Structure tests as:

- **Unit tests** — test model logic, parsing, filtering without Docker
- **Integration tests** — gated behind `//go:build integration` tag, require a running Docker daemon
- **Manual testing** — use a real Docker Compose project. A `docker-compose.yml` in `testdata/` with simple services (nginx, alpine) for manual testing.

Create `testdata/docker-compose.yml`:
```yaml
services:
  web:
    image: nginx:alpine
    ports: ["8080:80"]
  worker:
    image: alpine
    command: sh -c 'while true; do echo "[INFO] heartbeat $(date)"; sleep 2; done'
  failing:
    image: alpine
    command: sh -c 'while true; do echo "[ERROR] something broke" >&2; sleep 5; done'
```

## Key files reference

| File | Purpose | Lines |
|------|---------|-------|
| `cmd/root.go` | CLI entry, flags, bootstrap | 90 |
| `internal/app/app.go` | Main model (to be refactored) | 1231 |
| `internal/app/keys.go` | All keybinding definitions | ~100 |
| `internal/docker/client.go` | Docker API wrapper | 165 |
| `internal/docker/logs.go` | Log streaming | 107 |
| `internal/docker/exec.go` | Container exec | 69 |
| `internal/model/log.go` | LogEntry, ParseLevel | 69 |
| `internal/model/container.go` | Container struct, colors | 40 |
| `internal/theme/theme.go` | Theme and styles | ~100 |
