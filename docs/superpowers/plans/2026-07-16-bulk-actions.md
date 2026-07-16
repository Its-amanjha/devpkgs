# Bulk Actions & Multi-Select Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-select with `Space` key and sequential bulk upgrade/remove across tagged packages.

**Architecture:** Add a `selected` map per tab state, a bulk queue in the Model, and a queue-advancing handler in the update loop that chains `RunAction` calls sequentially.

**Tech Stack:** Go, bubbletea, lipgloss.

## Global Constraints
- Selection scoped to current tab only.
- Sequential execution (never parallel).
- Preserve single-item behavior when nothing is selected.
- Skip up-to-date packages in bulk upgrade.

---

### Task 1: Selection State & Toggle

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/update.go`

**Interfaces:**
- Produces: `selected map[string]bool` field in `TabState`, `Space` key handler in `update.go`.

- [ ] **Step 1: Add `selected` field to `TabState` in `model.go`**
  ```go
  // In TabState struct, add:
  selected map[string]bool
  ```
  Initialize it in `New()` for each tab state:
  ```go
  selected: make(map[string]bool),
  ```

- [ ] **Step 2: Add `Space` key handler in `update.go`**
  In the `tea.KeyMsg` handler (after overlay checks, in the normal key handling section), add:
  ```go
  case " ":
      if !m.allMode {
          st := &m.states[m.activeTab]
          if st.cursor < len(st.displayPackages) {
              pkg := st.displayPackages[st.cursor]
              if st.selected == nil {
                  st.selected = make(map[string]bool)
              }
              if st.selected[pkg] {
                  delete(st.selected, pkg)
              } else {
                  st.selected[pkg] = true
              }
          }
      }
      return m, nil
  ```

- [ ] **Step 3: Compile and verify**
  Run: `go build ./...`
  Expected: SUCCESS

- [ ] **Step 4: Commit**

---

### Task 2: Checkbox Rendering in Package Lists

**Files:**
- Modify: `internal/app/panels.go`

**Interfaces:**
- Consumes: `TabState.selected` map from Task 1.

- [ ] **Step 1: Modify list item rendering to show checkboxes**
  In `panels.go`, find where individual package names are rendered in the left panel list. Prepend `[✓] ` or `[ ] ` based on `st.selected[pkg]`. Only show checkboxes when at least one package is selected in that tab (to keep the UI clean when not in multi-select mode).
  ```go
  showCheckboxes := len(st.selected) > 0
  prefix := ""
  if showCheckboxes {
      if st.selected[pkg] {
          prefix = "[✓] "
      } else {
          prefix = "[ ] "
      }
  }
  ```
  Adjust `innerWidth` calculations to account for the 4-character prefix when checkboxes are visible.

- [ ] **Step 2: Update footer to show selection count**
  In `overlays.go` or wherever the footer is rendered, when `len(st.selected) > 0`, append the count:
  ```go
  if !m.allMode && len(m.states[m.activeTab].selected) > 0 {
      countStr += fmt.Sprintf("  %d selected", len(m.states[m.activeTab].selected))
  }
  ```

- [ ] **Step 3: Compile and verify**
  Run: `go build ./...`

- [ ] **Step 4: Commit**

---

### Task 3: Bulk Queue State & Confirmation Dialog

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/overlays.go`
- Modify: `internal/app/update.go`

**Interfaces:**
- Produces: `bulkQueue`, `bulkIndex`, `bulkAction`, `bulkLogs` fields in Model, modified confirmation dialog.

- [ ] **Step 1: Add bulk queue fields to `Model` in `model.go`**
  ```go
  bulkQueue  []string
  bulkIndex  int
  bulkAction pm.Action
  bulkLogs   bool
  ```

- [ ] **Step 2: Modify `u` / `x` key handlers in `update.go`**
  When the user presses `u` or `x`, check if there are selected packages:
  ```go
  case "u":
      if !m.allMode {
          st := &m.states[m.activeTab]
          if len(st.selected) > 0 {
              // Bulk mode
              var queue []string
              for _, pkg := range st.displayPackages {
                  if st.selected[pkg] {
                      queue = append(queue, pkg)
                  }
              }
              m.bulkQueue = queue
              m.bulkIndex = 0
              m.bulkAction = pm.Upgrade
              m.pendingTab = m.activeTab
              m.actionOverlay = true
              return m, nil
          }
          // Single item (existing behavior)
          ...
      }
  ```

- [ ] **Step 3: Update confirmation dialog text for bulk mode**
  In `renderActionOverlay()` in `overlays.go`:
  ```go
  if len(m.bulkQueue) > 0 {
      content = fmt.Sprintf("%s %d packages using %s?\n\nEnter: confirm   y: with logs   Esc/n: cancel",
          string(m.bulkAction), len(m.bulkQueue), m.tabs[m.pendingTab].Name())
  }
  ```

- [ ] **Step 4: Compile and verify**
  Run: `go build ./...`

- [ ] **Step 5: Commit**

---

### Task 4: Sequential Queue Execution

**Files:**
- Modify: `internal/app/update.go`
- Modify: `internal/app/model.go`

**Interfaces:**
- Consumes: `bulkQueue`, `bulkIndex`, `bulkAction`, `bulkLogs` from Task 3.

- [ ] **Step 1: Modify `enter` / `y` confirmation handler for bulk mode**
  In the `actionOverlay` key handler in `update.go`, when bulk queue is populated:
  ```go
  case "enter", "y":
      m.actionOverlay = false
      m.bulkLogs = msg.String() == "y"
      if len(m.bulkQueue) > 0 {
          // Start bulk execution
          return m.startNextBulkAction()
      }
      // Single action (existing behavior)
      ...
  ```

- [ ] **Step 2: Create `startNextBulkAction()` helper in `model.go`**
  ```go
  func (m Model) startNextBulkAction() (Model, tea.Cmd) {
      if m.bulkIndex >= len(m.bulkQueue) {
          // All done — clear selections, refresh tab
          m.states[m.pendingTab].selected = make(map[string]bool)
          m.bulkQueue = nil
          m.actionStatus = fmt.Sprintf("Bulk %s completed for %d packages", m.bulkAction, m.bulkIndex)
          return m.refreshTab(m.tabs[m.pendingTab].Name())
      }

      pkg := m.bulkQueue[m.bulkIndex]

      // Skip up-to-date packages during upgrade
      if m.bulkAction == pm.Upgrade && m.isUpToDate(m.pendingTab, pkg) {
          m.bulkIndex++
          return m.startNextBulkAction()
      }

      m.actionStatus = fmt.Sprintf("Upgrading %d/%d: %s...", m.bulkIndex+1, len(m.bulkQueue), pkg)
      m.logLines = nil
      m.logScrollOffset = 0
      m.logScrollActive = false
      m.logActive = true
      m.logChan = make(chan tea.Msg, 100)

      if m.bulkLogs {
          m.logOverlay = true
      }

      cmd := m.tabs[m.pendingTab].RunAction(pkg, m.bulkAction, m.logChan)
      return m, tea.Batch(cmd, ListenLogs(m.logChan))
  }
  ```

- [ ] **Step 3: Modify `ActionMsg` handler to advance bulk queue**
  In the `pm.ActionMsg` case in `update.go`:
  ```go
  case pm.ActionMsg:
      if msg.Err != nil {
          m.actionStatus = fmt.Sprintf("%s failed for %s: %v", msg.Action, msg.PackageName, msg.Err)
      } else {
          m.actionStatus = fmt.Sprintf("%s completed for %s", msg.Action, msg.PackageName)
      }

      // If in bulk mode, advance to next package
      if len(m.bulkQueue) > 0 {
          m.bulkIndex++
          return m.startNextBulkAction()
      }

      return m.refreshTab(msg.Manager)
  ```

- [ ] **Step 4: Compile and run full test suite**
  Run: `go test -v ./...`
  Expected: PASS

- [ ] **Step 5: Commit**
