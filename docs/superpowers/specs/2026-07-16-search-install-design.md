# Design Spec: Search & Install (Registry Queries)

This spec details the implementation of a dedicated "SEARCH" tab in `devpkgs` which queries online package registries (WinGet, Brew, NPM, Pip) in parallel and allows users to install new packages.

---

## 1. Requirements & UX
* **Dedicated Tab**: A tab named `SEARCH` positioned at index 4 (right after WinGet). Can be navigated to using Arrow keys.
* **Unified Results**: Typing a query and pressing `Enter` queries all active managers concurrently, combining results into a single list with origin badges (e.g., `[npm]`, `[winget]`).
* **On-Demand Details**: Highlighting a result in the list queries that package's registry API in the background to show Description, Homepage, License, and Publisher in the right details panel.
* **Logs Integration**: Pressing `i` on a search result opens the logs overlay popup and streams the installer stdout/stderr in real-time. On success, it automatically switches back to the corresponding manager's tab and reloads the installed list.

---

## 2. Technical Architecture & Polymorphism
We will implement a virtual `SearchManager` struct satisfying the `pm.Manager` interface to slot naturally into the existing tab system without modifying the overall layout rendering logic.

### A. SearchManager Struct (`internal/pm/search.go` [NEW])
```go
package pm

import tea "github.com/charmbracelet/bubbletea"

type SearchManager struct {
	tabIndex int
}

func NewSearchManager(index int) *SearchManager {
	return &SearchManager{tabIndex: index}
}

func (s *SearchManager) Name() string     { return "search" }
func (s *SearchManager) TabLabel() string { return "Search" }
func (s *SearchManager) ListInstalled() tea.Cmd {
	return nil // Does not list installed packages
}
```

We add `Install Action = "install"` to `pm.go`:
```go
const (
	Upgrade Action = "upgrade"
	Remove  Action = "remove"
	Install Action = "install"
)
```

And `RunAction` on `SearchManager` will trigger the installer command depending on the highlighted package's origin manager.

---

## 3. Data Structures & State Fields (`internal/app/model.go`)
We append these fields to the `Model` struct:
```go
searchQueryResult   []SearchResult // Unified online search results list
searchQueryLoading  bool           // True if background query workers are active
searchQueryCursor   int            // Cursor index for the search results list
searchActiveWorkers int            // Count of active search goroutines
```

Where `SearchResult` is:
```go
type SearchResult struct {
	Name        string
	Manager     string
	Description string
	Version     string
}
```

---

## 4. Parallel Query Workers (`internal/pm/search.go`)
When search is submitted, we trigger a parallel search dispatcher:
- **Brew**: Checks matching formula locally from `BrewState.FormulaeMap`.
- **WinGet**: Runs `winget search <query>`.
- **NPM**: Fetches JSON from `https://registry.npmjs.org/-/v1/search?text=<query>&size=25`.
- **Pip**: Fetches JSON from `https://pypi.org/pypi/<query>/json` or index.

Each worker sends a `RegistrySearchMsg`:
```go
type RegistrySearchMsg struct {
	Manager string
	Results []SearchResult
	Err     error
}
```
The update loop combines the results, sorts them, and decrements `searchActiveWorkers`. Once `searchActiveWorkers == 0`, `searchQueryLoading` is set to `false`.

---

## 5. UI Layout & Rendering
- **Left Panel**: When active tab is index 4 (Search), render search input bar at the top, list results under it, and spinner if `searchQueryLoading` is true.
- **Right Panel**: Queries full description and metadata for the selected `SearchResult` on-demand using custom background APIs.
