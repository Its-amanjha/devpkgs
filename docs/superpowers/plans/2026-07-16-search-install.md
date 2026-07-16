# Search & Install Implementation Plan

**Goal:** Add a dedicated SEARCH tab that queries WinGet (CLI), Brew (local), NPM (HTTP API), and Pip (HTTP API) in parallel, displays unified results with manager badges, shows on-demand package details, and installs packages with live log streaming.

**Architecture:** The SEARCH tab is implemented as a virtual tab (not backed by a `pm.Manager`) that renders its own left/right panels. Search queries dispatch 4 parallel workers. Results merge into a unified list. Install actions reuse the existing `RunStream` + logs overlay infrastructure.

---

### Task 1: Search State, Data Structures & Tab Navigation

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/tabs.go`
- Modify: `internal/app/update.go`
- Modify: `internal/app/view.go`

- [ ] **Step 1: Define `SearchResult` struct and add search state fields to `Model`**
  In `model.go`, add:
  ```go
  type SearchResult struct {
      Name        string
      Manager     string
      Description string
      Version     string
  }
  ```
  Add fields to `Model`:
  ```go
  searchResults      []SearchResult
  searchResultCursor int
  searchLoading      bool
  searchActiveWorkers int
  searchTabActive    bool // true when user is on the SEARCH tab
  ```

- [ ] **Step 2: Add SEARCH tab to `renderTabBar` in `tabs.go`**
  After the `for i, tab := range m.tabs` loop, add a "SEARCH" tab cell:
  ```go
  searchLabel := "SEARCH"
  if m.searchTabActive {
      cells = append(cells, activeStyle.Render(searchLabel))
  } else {
      cells = append(cells, inactiveStyle.Render(searchLabel))
  }
  ```

- [ ] **Step 3: Update left/right arrow tab navigation in `update.go`**
  - `right` key: When `m.activeTab == len(m.tabs)-1` (last installed tab), pressing right enters `searchTabActive = true` instead of stopping.
  - `left` key: When `searchTabActive`, pressing left exits search tab and returns to the last installed tab.
  - Entering the search tab should auto-focus the search bar (`m.searchActive = true`).

- [ ] **Step 4: Update `View()` in `view.go`**
  When `m.searchTabActive`, render search-specific left/right panels instead of the installed package panels.

- [ ] **Step 5: Compile and verify**
  Run: `go build ./...`

- [ ] **Step 6: Commit**

---

### Task 2: Parallel Registry Search Workers

**Files:**
- New: `internal/pm/search.go`
- Modify: `internal/app/update.go`
- Modify: `internal/app/model.go`

- [ ] **Step 1: Create `internal/pm/search.go` with search message types and worker functions**
  Define:
  ```go
  type RegistrySearchMsg struct {
      Manager string
      Results []SearchResult
      Err     error
  }

  type SearchResult struct {
      Name        string
      Manager     string
      Description string
      Version     string
  }
  ```
  Implement 4 search functions, each returning `tea.Cmd`:
  - `SearchBrew(query string, formulaeMap map[string]FormulaData) tea.Cmd` — filter in-memory formulae
  - `SearchWinget(query string) tea.Cmd` — run `winget search --query <query>` and parse output
  - `SearchNpm(query string) tea.Cmd` — HTTP GET to `https://registry.npmjs.org/-/v1/search?text=<query>&size=25`
  - `SearchPip(query string) tea.Cmd` — HTTP GET to `https://pypi.org/search/?q=<query>` or use the simple JSON API

- [ ] **Step 2: Add search dispatch logic in `update.go`**
  When user presses `Enter` while on the search tab:
  - Set `m.searchLoading = true`, `m.searchResults = nil`, `m.searchActiveWorkers = 4`
  - Dispatch all 4 search commands via `tea.Batch`
  - For Brew, pass the local formulae map if available

- [ ] **Step 3: Handle `RegistrySearchMsg` in `update.go`**
  On receiving a `RegistrySearchMsg`:
  - Append results to `m.searchResults`
  - Decrement `m.searchActiveWorkers`
  - If `searchActiveWorkers == 0`, set `searchLoading = false`
  - Sort results alphabetically

- [ ] **Step 4: Add unit tests for search parsing**
  - Test WinGet search output parsing
  - Test Brew local search filtering

- [ ] **Step 5: Compile and verify**

- [ ] **Step 6: Commit**

---

### Task 3: Search Results UI Rendering

**Files:**
- Modify: `internal/app/panels.go`
- Modify: `internal/app/view.go`
- Modify: `internal/app/update.go`

- [ ] **Step 1: Create `renderSearchLeftPanel` in `panels.go`**
  Renders the search results list with manager badges (like the ALL tab does):
  - Show spinner if `m.searchLoading`
  - Show "Type a package name and press Enter to search" if no results and not loading
  - Each result shows: `[npm] express  — Fast, unopinionated...`
  - Highlight cursor item

- [ ] **Step 2: Create `renderSearchRightPanel` in `panels.go`**
  When a search result is highlighted, show available details:
  - Name, Manager, Version, Description
  - (On-demand detail fetching will be added in Task 5)

- [ ] **Step 3: Wire panels into `View()` in `view.go`**
  When `searchTabActive`, call `renderSearchLeftPanel` and `renderSearchRightPanel` instead of the regular panels.

- [ ] **Step 4: Handle up/down cursor navigation for search results in `update.go`**
  When on the search tab, up/down keys control `m.searchResultCursor`.

- [ ] **Step 5: Compile and verify**

- [ ] **Step 6: Commit**

---

### Task 4: Install Action & Logs Integration

**Files:**
- Modify: `internal/pm/pm.go`
- Modify: `internal/app/update.go`
- Modify: `internal/app/model.go`

- [ ] **Step 1: Add `Install` action constant to `pm.go`**
  ```go
  Install Action = "install"
  ```

- [ ] **Step 2: Implement install command builder**
  Create a helper that returns the correct install command based on manager name:
  - `winget`: `winget install --id <name> --exact --accept-package-agreements --accept-source-agreements --disable-interactivity`
  - `brew`: `brew install <name>`
  - `npm`: `npm install -g <name>`
  - `pip`: `pip install <name>`

- [ ] **Step 3: Handle `i` key press on search results in `update.go`**
  When user presses `i` on a highlighted search result:
  - Open confirmation dialog showing the package name and manager
  - On confirm (`Enter` or `y`), run the install command via `RunStream` with the logs overlay
  - On success, auto-switch to the manager's installed tab and refresh its package list

- [ ] **Step 4: Handle auto-tab-switch after successful install**
  In the `ActionMsg` handler, when `msg.Action == pm.Install`:
  - Close logs overlay
  - Find the tab index for `msg.Manager`
  - Set `m.searchTabActive = false`, `m.activeTab = tabIndex`
  - Refresh that tab

- [ ] **Step 5: Compile and run full test suite**

- [ ] **Step 6: Commit**

---

### Task 5: On-Demand Search Result Details

**Files:**
- Modify: `internal/pm/search.go`
- Modify: `internal/app/update.go`
- Modify: `internal/app/panels.go`
- Modify: `internal/app/model.go`

- [ ] **Step 1: Add detail fetching functions to `search.go`**
  - `FetchSearchDetail(result SearchResult) tea.Cmd` — dispatches the right API call based on manager:
    - Brew: lookup from local formulae map
    - NPM: `https://registry.npmjs.org/<name>`
    - Pip: `https://pypi.org/pypi/<name>/json`
    - WinGet: `winget show --id <name> --exact`

- [ ] **Step 2: Add `SearchDetailMsg` type and `searchDetailCache` to `model.go`**
  ```go
  type SearchDetailMsg struct {
      Name        string
      Manager     string
      Description string
      Homepage    string
      License     string
      Publisher   string
      Err         error
  }

  // In Model:
  searchDetailCache map[string]*SearchDetailMsg
  ```

- [ ] **Step 3: Trigger detail fetch on cursor move in `update.go`**
  When cursor moves to a new search result, check cache. If not cached, dispatch `FetchSearchDetail`.

- [ ] **Step 4: Render details in `renderSearchRightPanel` in `panels.go`**
  Display Description, Homepage, License, Publisher from cache. Show "Loading..." spinner if not yet cached.

- [ ] **Step 5: Compile and run full test suite**

- [ ] **Step 6: Commit**
