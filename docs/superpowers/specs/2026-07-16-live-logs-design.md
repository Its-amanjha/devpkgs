# Design Spec: Live Logs and Console Output Panel

This spec details the implementation of a real-time console logs overlay panel for `devpkgs`. It enables users to watch background installation streams (stdout/stderr) live in a centered box.

---

## 1. Requirements & UX
* **Overlay Dialog**: A centered rounded box displayed on top of the dashboard.
* **On-Demand Access**: Opens automatically when an upgrade/removal action begins. Can be toggled open/closed by pressing `l` or closed with `Esc`.
* **State Scope**: Caches only the logs of the *active* or *last executed* task. Starts clean on every new action.
* **Scroll Behavior**: Auto-scrolls to the bottom by default. Using `Up`/`Down` arrow keys scroll the view manually, pausing auto-scroll. Reaching the bottom resumes auto-scroll.

---

## 2. Technical Architecture & Data Flow

### A. State Fields (`internal/app/model.go`)
We append these fields to the `Model` struct:
```go
logOverlay      bool      // True if the logs overlay window is open
logLines        []string  // Captured lines of output from the current/last task
logScrollOffset int       // Current scroll line offset from the bottom
logScrollActive bool      // True if the user manually scrolled, pausing auto-scroll
logActive       bool      // True if a task is actively running and printing output
```

### B. Message Types (`internal/pm/pm.go`)
We define these message structures to communicate output lines:
```go
type LogLineMsg struct {
	Line string
}

type LogFinishMsg struct {
	Err error
}
```

### C. Live Command Streaming (`internal/pm/pm.go`)
Instead of blocking on `exec.CombinedOutput()`, we stream line-by-line using a custom Bubble Tea command generator:
```go
func RunStream(packageName string, action Action, manager string, cmdProgram string, args ...string) (tea.Cmd, chan struct{}) {
    // Returns a tea.Cmd that starts the command, redirects stdout/stderr to a reader,
    // and runs a background goroutine to scan line-by-line and dispatch LogLineMsg.
    // Dispatches LogFinishMsg and ActionMsg on finish.
}
```

---

## 3. Keyboard Event Changes (`internal/app/update.go`)
* **When `logOverlay` is true**:
  * `up` / `down`: Scroll up/down the log lines by adjusting `logScrollOffset`. Set `logScrollActive = true`.
  * `esc` / `l`: Close logs overlay (set `logOverlay = false`).
  * Other keys: Blocked (except Ctrl+C / q to quit).
* **When `logOverlay` is false**:
  * `l`: Open logs overlay.

---

## 4. UI Layout & Rendering (`internal/app/overlays.go`)
We add a rendering method `renderLogOverlay()` which outputs a centered, rounded border box styled with the active theme containing the scroll-padded lines of `logLines`.
