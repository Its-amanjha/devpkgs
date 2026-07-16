# Design Spec: devpkgs Modular Rewrite

- **Author**: Aman Kumar Jha
- **Date**: 2026-07-15
- **Status**: Proposed
- **Creative North Star**: "The Monospace Sanctuary"

---

## 1. Goal & Context
The goal is to rewrite the base package managers and main dashboard code of `devpkgs` from scratch to achieve 100% copyright ownership and a cleaner, modular architecture. 

We must:
1. **Preserve User Additions**: Retain the exact behavior and code of WinGet support, filters, and action confirmation overlays.
2. **Modularize Bloat**: Split the 1,600+ line `model.go` into logical files (`model.go`, `update.go`, `view.go`, `panels.go`, `overlays.go`).
3. **Rewrite Upstream Engines**: Implement original versions of the Homebrew, NPM, and Pip package list engines without copying any code from the upstream repository, while ensuring functional compatibility.

---

## 2. Target File Architecture

### 2.1 Package Managers (`internal/pm/`)
- **[pm.go](file:///d:/Github%20repo/devpkgs/internal/pm/pm.go)**: Common interface `Manager`, standard `Action` types (`Upgrade`, `Remove`), and general execution helpers.
- **[winget.go](file:///d:/Github%20repo/devpkgs/internal/pm/winget.go)**: Kept exactly as-is (fully authored by the user).
- **[brew.go](file:///d:/Github%20repo/devpkgs/internal/pm/brew.go)**: Redesigned parser using `brew list --formula --versions` and a JSON decoder for formula definitions, fetching disk sizes via standard filepath calculations.
- **[npm.go](file:///d:/Github%20repo/devpkgs/internal/pm/npm.go)**: Redesigned global module list parser using `npm ls -g --depth=0 --json` and concurrent metadata fetchers using limited goroutines.
- **[pip.go](file:///d:/Github%20repo/devpkgs/internal/pm/pip.go)**: Resolves python/pip commands and parses JSON output from `pip list --format=json`.

### 2.2 View & Controllers (`internal/app/`)
- **[model.go](file:///d:/Github%20repo/devpkgs/internal/app/model.go)**: Contains state structs (`Model`, `TabState`, `BrewState`) and constructor functions.
- **`update.go` [NEW]**: Implements `Update(msg tea.Msg) (tea.Model, tea.Cmd)` handling:
  - Tab navigation (left/right arrow keys).
  - Search queries and package list updates.
  - Action confirmations and CLI execution triggers.
- **`view.go` [NEW]**: Implements `View() string` assembling panels, margins, headers, and footers.
- **`panels.go` [NEW]**: Contains functions to render the left pane package lists (with visual tags) and the right pane package details.
- **`overlays.go` [NEW]**: Contains functions to render the center confirmation overlay box and the theme selection menu overlay.

---

## 3. Key Design Decisions

### 3.1 Splitting State & Controller
Instead of having all UI rendering and event mapping in one file, `model.go` will be purely state-declarative. Event handlers will live in `update.go` and UI components in `view.go`/`panels.go`.

### 3.2 Thread-Safe Concurrent Metadata Fetching
We will use safe channel-based semaphores for concurrent HTTP requests in the NPM and Pip detail retrievers.

---

## 4. Verification Plan

### Automated Tests
1. Run `go test ./...` to verify WinGet parsing and Npm outdated logic.
2. Compile the binary using `go build ./...` to verify type safety.

### Manual Verification
- Launch the application using `go run .`.
- Verify searching, filtering, switching tabs, selecting packages, updating themes, and trigger a safe Upgrade/Removal action to verify overlay confirmation dialogs.
