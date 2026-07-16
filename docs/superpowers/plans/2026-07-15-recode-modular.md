# devpkgs Modular Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite and modularize the devpkgs codebase to achieve 100% authorship ownership and a clean, maintainable structure, preserving all existing WinGet support, filters, and action confirmation features.

**Architecture:** Split the monolithic `model.go` into specialized logical units: `model.go` (state), `update.go` (controllers/handlers), `view.go` (layouts), `panels.go` (pane content), and `overlays.go` (overlays). Implement clean, original versions of the `brew`, `npm`, and `pip` package manager engines.

**Tech Stack:** Go 1.25.5, Bubble Tea, Lipgloss

## Global Constraints
- Do not copy or paste any code from the upstream `pkgview` codebase for `brew`, `npm`, `pip`, or the core Bubble Tea layout.
- Preserve the exact behavior and code of `internal/pm/winget.go` and its tests.
- Keep rounded box drawing characters (`╭─ ─╮`) as defined in the visual spec.

---

### Task 1: Package Manager Interfaces and Concrete Engines

**Files:**
- Modify: [internal/pm/pm.go](file:///d:/Github%20repo/devpkgs/internal/pm/pm.go)
- Modify: [internal/pm/brew.go](file:///d:/Github%20repo/devpkgs/internal/pm/brew.go)
- Modify: [internal/pm/npm.go](file:///d:/Github%20repo/devpkgs/internal/pm/npm.go)
- Modify: [internal/pm/pip.go](file:///d:/Github%20repo/devpkgs/internal/pm/pip.go)

**Interfaces:**
- Consumes: None (base interfaces)
- Produces: `pm.Manager` interface, `pm.Action` enum, `pm.PackageListMsg`, `pm.BrewListMsg`, `pm.BrewFormulaeMsg`, `pm.NpmAllDetailsMsg`, `pm.PipAllDetailsMsg`

- [ ] **Step 1: Write pm.go common interface**
  Rewrite `internal/pm/pm.go` with the core interface, actions, messages, and command execution helper:
  ```go
  package pm

  import (
  	"os/exec"
  	tea "github.com/charmbracelet/bubbletea"
  )

  type Manager interface {
  	Name() string
  	TabLabel() string
  	ListInstalled() tea.Cmd
  	RunAction(packageName string, action Action) tea.Cmd
  }

  type Action string

  const (
  	Upgrade Action = "upgrade"
  	Remove  Action = "remove"
  )

  type PackageListMsg struct {
  	Packages []string
  	Versions map[string]string
  	Err      error
  	TabIndex int
  }

  type ActionMsg struct {
  	PackageName string
  	Action      Action
  	Err         error
  }

  func Run(packageName string, action Action, name string, args ...string) tea.Cmd {
  	return func() tea.Msg {
  		_, err := exec.Command(name, args...).CombinedOutput()
  		if err != nil {
  			return ActionMsg{PackageName: packageName, Action: action, Err: err}
  		}
  		return ActionMsg{PackageName: packageName, Action: action}
  	}
  }
  ```

- [ ] **Step 2: Run tests to verify existing WinGet parser passes**
  Run: `go test -v ./internal/pm/...`
  Expected: PASS (`TestParseWingetExport` succeeds)

- [ ] **Step 3: Implement original brew.go manager**
  Rewrite `internal/pm/brew.go` to fetch local Homebrew formulas and public metadata:
  ```go
  package pm

  import (
  	"encoding/json"
  	"net/http"
  	"os/exec"
  	"strconv"
  	"strings"
  	"time"

  	tea "github.com/charmbracelet/bubbletea"
  )

  type BrewManager struct {
  	tabIndex int
  }

  func NewBrewManager(tabIndex int) *BrewManager {
  	return &BrewManager{tabIndex: tabIndex}
  }

  func (b *BrewManager) Name() string     { return "brew" }
  func (b *BrewManager) TabLabel() string { return "Brew" }

  type BrewListMsg struct {
  	Names             []string
  	Paths             map[string]string
  	InstalledVersions map[string]string
  	Sizes             map[string]int64
  }

  type BrewErrMsg error
  type BrewFormulaeMsg map[string]FormulaData
  type BrewFormulaeErrMsg error

  type FormulaData struct {
  	Name     string `json:"name"`
  	Desc     string `json:"desc"`
  	Homepage string `json:"homepage"`
  	License  string `json:"license"`
  	Versions struct {
  		Stable string `json:"stable"`
  	} `json:"versions"`
  	Dependencies      []string `json:"dependencies"`
  	BuildDependencies []string `json:"build_dependencies"`
  }

  func (b *BrewManager) ListInstalled() tea.Cmd {
  	if _, err := exec.LookPath("brew"); err != nil {
  		return func() tea.Msg { return PackageListMsg{TabIndex: b.tabIndex} }
  	}
  	return tea.Batch(b.fetchList(), b.fetchFormulae())
  }

  func (b *BrewManager) RunAction(name string, action Action) tea.Cmd {
  	if action == Remove {
  		return Run(name, action, "brew", "uninstall", name)
  	}
  	return Run(name, action, "brew", "upgrade", name)
  }

  func (b *BrewManager) fetchList() tea.Cmd {
  	return func() tea.Msg {
  		out, err := exec.Command("brew", "list", "--formula", "--versions").Output()
  		if err != nil {
  			return BrewErrMsg(err)
  		}
  		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
  		names := make([]string, 0, len(lines))
  		versions := make(map[string]string)
  		paths := make(map[string]string)
  		sizes := make(map[string]int64)

  		prefixOut, err := exec.Command("brew", "--prefix").Output()
  		prefix := strings.TrimSpace(string(prefixOut))

  		for _, line := range lines {
  			fields := strings.Fields(line)
  			if len(fields) >= 2 {
  				name, ver := fields[0], fields[1]
  				names = append(names, name)
  				versions[name] = ver
  				if err == nil && prefix != "" {
  					path := prefix + "/opt/" + name
  					paths[name] = path
  					if duOut, duErr := exec.Command("du", "-skL", path).Output(); duErr == nil {
  						duFields := strings.Fields(string(duOut))
  						if len(duFields) > 0 {
  							if kb, parseErr := strconv.ParseInt(duFields[0], 10, 64); parseErr == nil {
  								sizes[name] = kb * 1024
  							}
  						}
  					}
  				}
  			}
  		}
  		return BrewListMsg{Names: names, Paths: paths, InstalledVersions: versions, Sizes: sizes}
  	}
  }

  func (b *BrewManager) fetchFormulae() tea.Cmd {
  	return func() tea.Msg {
  		client := &http.Client{Timeout: 15 * time.Second}
  		resp, err := client.Get("https://formulae.brew.sh/api/formula.json")
  		if err != nil {
  			return BrewFormulaeErrMsg(err)
  		}
  		defer resp.Body.Close()

  		var rawList []FormulaData
  		if err := json.NewDecoder(resp.Body).Decode(&rawList); err != nil {
  			return BrewFormulaeErrMsg(err)
  		}

  		m := make(map[string]FormulaData)
  		for _, f := range rawList {
  			m[f.Name] = f
  		}
  		return BrewFormulaeMsg(m)
  	}
  }
  ```

- [ ] **Step 4: Implement original npm.go manager**
  Rewrite `internal/pm/npm.go` to list global modules and fetch metadata:
  ```go
  package pm

  import (
  	"encoding/json"
  	"fmt"
  	"net/http"
  	"os/exec"
  	"sync"
  	"time"

  	tea "github.com/charmbracelet/bubbletea"
  )

  type NpmManager struct {
  	tabIndex int
  }

  func NewNpmManager(tabIndex int) *NpmManager {
  	return &NpmManager{tabIndex: tabIndex}
  }

  func (n *NpmManager) Name() string     { return "npm" }
  func (n *NpmManager) TabLabel() string { return "npm" }

  type NpmDist struct {
  	UnpackedSize int64 `json:"unpackedSize"`
  }

  type NpmDetailData struct {
  	Name        string   `json:"name"`
  	Version     string   `json:"version"`
  	Description string   `json:"description"`
  	License     string   `json:"license"`
  	Homepage    string   `json:"homepage"`
  	Dist        *NpmDist `json:"dist,omitempty"`
  }

  type NpmAllDetailsMsg map[string]*NpmDetailData

  func (n *NpmManager) ListInstalled() tea.Cmd {
  	return func() tea.Msg {
  		if _, err := exec.LookPath("npm"); err != nil {
  			return PackageListMsg{TabIndex: n.tabIndex}
  		}
  		out, err := exec.Command("npm", "ls", "-g", "--depth=0", "--json").Output()
  		if err != nil {
  			return PackageListMsg{TabIndex: n.tabIndex}
  		}
  		var result struct {
  			Dependencies map[string]struct {
  				Version string `json:"version"`
  			} `json:"dependencies"`
  		}
  		if err := json.Unmarshal(out, &result); err != nil {
  			return PackageListMsg{Err: err, TabIndex: n.tabIndex}
  		}
  		names := make([]string, 0, len(result.Dependencies))
  		versions := make(map[string]string)
  		for name, dep := range result.Dependencies {
  			names = append(names, name)
  			versions[name] = dep.Version
  		}
  		return PackageListMsg{Packages: names, Versions: versions, TabIndex: n.tabIndex}
  	}
  }

  func (n *NpmManager) RunAction(name string, action Action) tea.Cmd {
  	if action == Remove {
  		return Run(name, action, "npm", "uninstall", "-g", name)
  	}
  	return Run(name, action, "npm", "update", "-g", name)
  }

  func FetchAllNpmDetails(names []string) tea.Cmd {
  	return func() tea.Msg {
  		res := make(map[string]*NpmDetailData)
  		var mu sync.Mutex
  		sem := make(chan struct{}, 5)
  		var wg sync.WaitGroup

  		client := http.Client{Timeout: 10 * time.Second}
  		for _, name := range names {
  			wg.Add(1)
  			go func(pkg string) {
  				defer wg.Done()
  				sem <- struct{}{}
  				defer func() { <-sem }()

  				resp, err := client.Get(fmt.Sprintf("https://registry.npmjs.org/%s/latest", pkg))
  				if err != nil {
  					return
  				}
  				defer resp.Body.Close()

  				var data NpmDetailData
  				if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
  					mu.Lock()
  					res[pkg] = &data
  					mu.Unlock()
  				}
  			}(name)
  		}
  		wg.Wait()
  		return NpmAllDetailsMsg(res)
  	}
  }
  ```

- [ ] **Step 5: Implement original pip.go manager**
  Rewrite `internal/pm/pip.go` to resolve command line tools and pull package metadata:
  ```go
  package pm

  import (
  	"encoding/json"
  	"fmt"
  	"net/http"
  	"os/exec"
  	"sync"
  	"time"

  	tea "github.com/charmbracelet/bubbletea"
  )

  type PipManager struct {
  	tabIndex int
  }

  func NewPipManager(tabIndex int) *PipManager {
  	return &PipManager{tabIndex: tabIndex}
  }

  func (p *PipManager) Name() string     { return "pip" }
  func (p *PipManager) TabLabel() string { return "pip" }

  type PipDetailData struct {
  	Name        string `json:"name"`
  	Version     string `json:"version"`
  	Summary     string `json:"summary"`
  	License     string `json:"license"`
  	HomePage    string `json:"home_page"`
  	Author      string `json:"author"`
  	AuthorEmail string `json:"author_email"`
  }

  type PipAllDetailsMsg map[string]*PipDetailData

  func (p *PipManager) ListInstalled() tea.Cmd {
  	return func() tea.Msg {
  		cmd, args := p.resolve()
  		if cmd == "" {
  			return PackageListMsg{TabIndex: p.tabIndex}
  		}
  		args = append(args, "list", "--format=json")
  		out, err := exec.Command(cmd, args...).Output()
  		if err != nil {
  			return PackageListMsg{TabIndex: p.tabIndex}
  		}
  		var rawList []struct {
  			Name    string `json:"name"`
  			Version string `json:"version"`
  		}
  		if err := json.Unmarshal(out, &rawList); err != nil {
  			return PackageListMsg{Err: err, TabIndex: p.tabIndex}
  		}
  		names := make([]string, 0, len(rawList))
  		versions := make(map[string]string)
  		for _, pkg := range rawList {
  			names = append(names, pkg.Name)
  			versions[pkg.Name] = pkg.Version
  		}
  		return PackageListMsg{Packages: names, Versions: versions, TabIndex: p.tabIndex}
  	}
  }

  func (p *PipManager) RunAction(name string, action Action) tea.Cmd {
  	cmd, prefix := p.resolve()
  	if cmd == "" {
  		return func() tea.Msg { return ActionMsg{PackageName: name, Action: action, Err: fmt.Errorf("pip not found")} }
  	}
  	args := append(prefix, "install", "--upgrade", name)
  	if action == Remove {
  		args = append(prefix, "uninstall", "-y", name)
  	}
  	return Run(name, action, cmd, args...)
  }

  func (p *PipManager) resolve() (string, []string) {
  	if _, err := exec.LookPath("pip"); err == nil {
  		return "pip", nil
  	}
  	if _, err := exec.LookPath("pip3"); err == nil {
  		return "pip3", nil
  	}
  	if _, err := exec.LookPath("python3"); err == nil {
  		return "python3", []string{"-m", "pip"}
  	}
  	if _, err := exec.LookPath("python"); err == nil {
  		return "python", []string{"-m", "pip"}
  	}
  	return "", nil
  }

  func FetchAllPipDetails(names []string) tea.Cmd {
  	return func() tea.Msg {
  		res := make(map[string]*PipDetailData)
  		var mu sync.Mutex
  		sem := make(chan struct{}, 5)
  		var wg sync.WaitGroup

  		client := http.Client{Timeout: 10 * time.Second}
  		for _, name := range names {
  			wg.Add(1)
  			go func(pkg string) {
  				defer wg.Done()
  				sem <- struct{}{}
  				defer func() { <-sem }()

  				resp, err := client.Get(fmt.Sprintf("https://pypi.org/pypi/%s/json", pkg))
  				if err != nil {
  					return
  				}
  				defer resp.Body.Close()

  				var wrapper struct {
  					Info PipDetailData `json:"info"`
  				}
  				if err := json.NewDecoder(resp.Body).Decode(&wrapper); err == nil {
  					mu.Lock()
  					res[pkg] = &wrapper.Info
  					mu.Unlock()
  				}
  			}(name)
  		}
  		wg.Wait()
  		return PipAllDetailsMsg(res)
  	}
  }
  ```

- [ ] **Step 6: Commit changes to internal/pm**
  Run:
  ```bash
  git add internal/pm/*.go
  git commit -m "refactor: rewrite package managers for clean architecture and 100% authorship"
  ```

---

### Task 2: Core State Model (`internal/app/model.go`)

**Files:**
- Modify: [internal/app/model.go](file:///d:/Github%20repo/devpkgs/internal/app/model.go)

**Interfaces:**
- Consumes: `pm.Manager`, `pm.NpmDetailData`, `pm.PipDetailData`, `pm.FormulaData`
- Produces: `app.Model` structure, `app.New()` constructor, `app.TabState`, `app.BrewState`

- [ ] **Step 1: Write model state structures and helpers**
  Clean up and re-implement `internal/app/model.go` to only hold the structures, constructor, and basic state check helpers:
  ```go
  package app

  import (
  	"sort"
  	"strings"
  	"time"

  	"github.com/charmbracelet/bubbles/spinner"
  	tea "github.com/charmbracelet/bubbletea"
  	"github.com/charmbracelet/lipgloss"

  	"devpkgs/internal/pm"
  )

  type BrewState struct {
  	FormulaeMap       map[string]pm.FormulaData
  	FormulaeReady     bool
  	APIErr            error
  	Info              *pm.FormulaData
  	InstallPaths      map[string]string
  	InstalledVersions map[string]string
  	Sizes             map[string]int64
  	BrewListDone      bool
  	BrewFormulaeDone  bool
  }

  type TabState struct {
  	packages        []string
  	displayPackages []string
  	cursor          int
  	loading         bool
  	err             error
  	progress        float64
  	progressTarget  float64
  	versions        map[string]string

  	Brew            *BrewState
  	NpmDetails      map[string]*pm.NpmDetailData
  	NpmDetailsReady bool
  	PipDetails      map[string]*pm.PipDetailData
  	PipDetailsReady bool
  	DetailErr       error
  }

  type progressTick struct{}

  func tickCmd() tea.Cmd {
  	return tea.Tick(60*time.Millisecond, func(t time.Time) tea.Msg {
  		return progressTick{}
  	})
  }

  type Model struct {
  	activeTab int
  	tabs      []pm.Manager
  	states    []TabState

  	width        int
  	height       int
  	searchActive bool
  	searchQuery  string

  	spinner    spinner.Model
  	spinnerCmd tea.Cmd

  	allMode            bool
  	allPackages        []string
  	allDisplayPackages []string
  	allCursor          int
  	allPackageOrigin   map[string]string

  	themeOverlay   bool
  	themeCursor    int
  	actionOverlay  bool
  	pendingAction  pm.Action
  	pendingTab     int
  	pendingPackage string
  	actionStatus   string
  	outdatedOnly   bool

  	sparklineHistory []float64
  }

  func New() Model {
  	applyTheme(themes[0])

  	s := spinner.New()
  	s.Style = lipgloss.NewStyle().Foreground(themes[0].Primary)
  	s.Spinner = spinner.MiniDot

  	managers := []pm.Manager{
  		pm.NewBrewManager(0),
  		pm.NewNpmManager(1),
  		pm.NewPipManager(2),
  		pm.NewWingetManager(3),
  	}
  	states := make([]TabState, len(managers))
  	for i, m := range managers {
  		target := 0.7
  		if m.Name() == "brew" {
  			target = 0.35
  		}
  		states[i] = TabState{
  			loading:        true,
  			progressTarget: target,
  		}
  		if m.Name() == "brew" {
  			states[i].Brew = &BrewState{}
  		}
  	}
  	return Model{
  		allMode:          true,
  		activeTab:        0,
  		tabs:             managers,
  		states:           states,
  		spinner:          s,
  		sparklineHistory: make([]float64, 0, 40),
  		allPackageOrigin: make(map[string]string),
  	}
  }

  func (m Model) Init() tea.Cmd {
  	cmds := []tea.Cmd{tickCmd(), m.spinner.Tick}
  	for _, t := range m.tabs {
  		cmds = append(cmds, t.ListInstalled())
  	}
  	return tea.Batch(cmds...)
  }

  func fuzzyMatch(s, query string) bool {
  	s = strings.ToLower(s)
  	q := strings.ToLower(query)
  	qi := 0
  	for i := 0; i < len(s) && qi < len(q); i++ {
  		if s[i] == q[qi] {
  			qi++
  		}
  	}
  	return qi == len(q)
  }

  func (m Model) applyFilter() Model {
  	if m.allMode {
  		query := strings.ToLower(m.searchQuery)
  		var filtered []string
  		for _, pkg := range m.allPackages {
  			if (!m.outdatedOnly || m.isOutdated(m.tabForPackage(pkg), pkg)) && (query == "" || fuzzyMatch(pkg, query)) {
  				filtered = append(filtered, pkg)
  			}
  		}
  		m.allDisplayPackages = filtered
  		if m.allCursor >= len(m.allDisplayPackages) {
  			m.allCursor = max(0, len(m.allDisplayPackages)-1)
  		}
  		return m
  	}
  	st := &m.states[m.activeTab]
  	query := strings.ToLower(m.searchQuery)
  	var filtered []string
  	for _, pkg := range st.packages {
  		if (!m.outdatedOnly || m.isOutdated(m.activeTab, pkg)) && (query == "" || fuzzyMatch(pkg, query)) {
  			filtered = append(filtered, pkg)
  		}
  	}
  	st.displayPackages = filtered
  	if st.cursor >= len(st.displayPackages) {
  		st.cursor = max(0, len(st.displayPackages)-1)
  	}
  	if len(st.displayPackages) > 0 && st.Brew != nil {
  		m = m.updateBrewInfo()
  	}
  	return m
  }

  func (m Model) tabForPackage(packageName string) int {
  	origin := m.allPackageOrigin[packageName]
  	for i, tab := range m.tabs {
  		if tab.Name() == origin {
  			return i
  		}
  	}
  	return -1
  }

  func (m Model) isOutdated(tabIndex int, packageName string) bool {
  	if tabIndex < 0 {
  		return false
  	}
  	installed := m.states[tabIndex].versions[packageName]
  	latest := ""
  	switch m.tabs[tabIndex].Name() {
  	case "brew":
  		if info := m.states[tabIndex].Brew; info != nil && info.FormulaeMap[packageName].Versions.Stable != "" {
  			latest = info.FormulaeMap[packageName].Versions.Stable
  		}
  	case "npm":
  		if info := m.states[tabIndex].NpmDetails[packageName]; info != nil {
  			latest = info.Version
  		}
  	case "pip":
  		if info := m.states[tabIndex].PipDetails[packageName]; info != nil {
  			latest = info.Version
  		}
  	}
  	return installed != "" && latest != "" && installed != latest
  }

  func (m Model) selectedPackage() (int, string, bool) {
  	if m.allMode {
  		if m.allCursor >= len(m.allDisplayPackages) {
  			return 0, "", false
  		}
  		name := m.allDisplayPackages[m.allCursor]
  		index := m.tabForPackage(name)
  		return index, name, index >= 0
  	}
  	state := m.states[m.activeTab]
  	if state.cursor >= len(state.displayPackages) {
  		return 0, "", false
  	}
  	return m.activeTab, state.displayPackages[state.cursor], true
  }

  func (m Model) refresh() (Model, tea.Cmd) {
  	commands := make([]tea.Cmd, 0, len(m.tabs))
  	for i, tab := range m.tabs {
  		m.states[i].loading = true
  		m.states[i].err = nil
  		commands = append(commands, tab.ListInstalled())
  	}
  	return m, tea.Batch(commands...)
  }

  func (m Model) updateBrewInfo() Model {
  	if m.allMode {
  		return m
  	}
  	st := &m.states[m.activeTab]
  	if st.Brew == nil {
  		return m
  	}
  	if len(st.displayPackages) > 0 && st.cursor < len(st.displayPackages) {
  		name := st.displayPackages[st.cursor]
  		if f, ok := st.Brew.FormulaeMap[name]; ok {
  			st.Brew.Info = &f
  		} else {
  			st.Brew.Info = nil
  		}
  	}
  	return m
  }

  func (m Model) selectPackageCmd() tea.Cmd {
  	if m.allMode {
  		return nil
  	}
  	st := &m.states[m.activeTab]
  	if st.Brew != nil {
  		m = m.updateBrewInfo()
  	}
  	return nil
  }

  func (m Model) totalPackages() int {
  	total := 0
  	for i := range m.states {
  		total += len(m.states[i].displayPackages)
  	}
  	return total
  }

  func (m Model) buildAllPackages() Model {
  	m.allPackages = nil
  	m.allPackageOrigin = make(map[string]string)
  	pmNames := make(map[int]string)
  	for i, t := range m.tabs {
  		pmNames[i] = t.Name()
  	}
  	for i, st := range m.states {
  		for _, pkg := range st.packages {
  			if _, exists := m.allPackageOrigin[pkg]; !exists {
  				m.allPackages = append(m.allPackages, pkg)
  			}
  			m.allPackageOrigin[pkg] = pmNames[i]
  		}
  	}
  	sort.Strings(m.allPackages)
  	return m.applyFilter()
  }

  func (m Model) allLoaded() bool {
  	for i := range m.states {
  		if m.states[i].loading {
  			return false
  		}
  	}
  	return true
  }

  func (m Model) updateSparkline() {
  	total := 0
  	for i := range m.states {
  		total += len(m.states[i].packages)
  	}
  	m.sparklineHistory = append(m.sparklineHistory, float64(total))
  	if len(m.sparklineHistory) > 40 {
  		m.sparklineHistory = m.sparklineHistory[len(m.sparklineHistory)-40:]
  	}
  }
  ```

- [ ] **Step 2: Run outdated model tests**
  Run: `go test -v ./internal/app/...`
  Expected: PASS

- [ ] **Step 3: Commit model.go changes**
  Run:
  ```bash
  git add internal/app/model.go
  git commit -m "refactor: simplify model.go to contain state definitions only"
  ```

---

### Task 3: Event Loop and Updates (`internal/app/update.go`)

**Files:**
- Create: `internal/app/update.go`

**Interfaces:**
- Consumes: `app.Model` state
- Produces: `app.Model.Update(msg tea.Msg) (tea.Model, tea.Cmd)`

- [ ] **Step 1: Write update.go event loop controller**
  Create `internal/app/update.go` containing the `Update` method:
  ```go
  package app

  import (
  	"fmt"
  	"strconv"
  	"strings"

  	"github.com/charmbracelet/bubbles/spinner"
  	tea "github.com/charmbracelet/bubbletea"
  	"github.com/charmbracelet/lipgloss"

  	"devpkgs/internal/pm"
  )

  func darkenHex(hex string, factor float64) lipgloss.Color {
  	h := strings.TrimPrefix(hex, "#")
  	if len(h) != 6 {
  		return lipgloss.Color(hex)
  	}
  	r, _ := strconv.ParseUint(h[0:2], 16, 8)
  	g, _ := strconv.ParseUint(h[2:4], 16, 8)
  	b, _ := strconv.ParseUint(h[4:6], 16, 8)
  	nr := int(float64(r) * (1 - factor))
  	ng := int(float64(g) * (1 - factor))
  	nb := int(float64(b) * (1 - factor))
  	if nr < 0 { nr = 0 }
  	if ng < 0 { ng = 0 }
  	if nb < 0 { nb = 0 }
  	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", nr, ng, nb))
  }

  func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  	switch msg := msg.(type) {

  	case tea.WindowSizeMsg:
  		m.width = msg.Width
  		m.height = msg.Height

  	case pm.PackageListMsg:
  		st := &m.states[msg.TabIndex]
  		if msg.Err != nil {
  			st.err = msg.Err
  			st.loading = false
  		} else {
  			st.packages = msg.Packages
  			st.displayPackages = msg.Packages
  			st.versions = msg.Versions
  			st.loading = false
  			tab := m.tabs[msg.TabIndex]
  			if tab.Name() == "npm" {
  				m = m.buildAllPackages()
  				return m, pm.FetchAllNpmDetails(msg.Packages)
  			}
  			if tab.Name() == "pip" {
  				m = m.buildAllPackages()
  				return m, pm.FetchAllPipDetails(msg.Packages)
  			}
  		}
  		st.progressTarget = 1.0
  		st.progress = 1.0
  		m.updateSparkline()
  		m = m.buildAllPackages()

  	case pm.BrewListMsg:
  		st := &m.states[0]
  		if st.Brew != nil {
  			st.packages = msg.Names
  			st.displayPackages = msg.Names
  			st.Brew.InstallPaths = msg.Paths
  			st.Brew.InstalledVersions = msg.InstalledVersions
  			st.Brew.Sizes = msg.Sizes
  			st.Brew.BrewListDone = true
  			st.progressTarget = 0.85
  			if st.Brew.BrewFormulaeDone {
  				st.loading = false
  				st.progressTarget = 1.0
  				st.progress = 1.0
  				m = m.updateBrewInfo()
  			}
  			m.updateSparkline()
  			m = m.buildAllPackages()
  		}

  	case pm.BrewErrMsg:
  		st := &m.states[0]
  		st.err = error(msg)
  		st.loading = false
  		st.progressTarget = 1.0
  		st.progress = 1.0
  		if st.Brew != nil {
  			st.Brew.BrewListDone = true
  		}

  	case pm.BrewFormulaeMsg:
  		st := &m.states[0]
  		if st.Brew != nil {
  			st.Brew.FormulaeMap = map[string]pm.FormulaData(msg)
  			st.Brew.FormulaeReady = true
  			st.Brew.BrewFormulaeDone = true
  			st.progressTarget = 1.0
  			st.progress = 1.0
  			if st.Brew.BrewListDone {
  				st.loading = false
  				m = m.updateBrewInfo()
  			}
  			m = m.applyFilter()
  		}

  	case pm.BrewFormulaeErrMsg:
  		st := &m.states[0]
  		if st.Brew != nil {
  			st.Brew.APIErr = error(msg)
  			st.Brew.FormulaeReady = true
  			st.Brew.BrewFormulaeDone = true
  			st.progressTarget = 1.0
  			st.progress = 1.0
  			if st.Brew.BrewListDone {
  				st.loading = false
  			}
  		}

  	case pm.NpmAllDetailsMsg:
  		st := &m.states[1]
  		if st.NpmDetails == nil {
  			st.NpmDetails = map[string]*pm.NpmDetailData(msg)
  		} else {
  			for k, v := range msg {
  				st.NpmDetails[k] = v
  			}
  		}
  		st.NpmDetailsReady = true
  		m = m.applyFilter()

  	case pm.PipAllDetailsMsg:
  		st := &m.states[2]
  		if st.PipDetails == nil {
  			st.PipDetails = map[string]*pm.PipDetailData(msg)
  		} else {
  			for k, v := range msg {
  				st.PipDetails[k] = v
  			}
  		}
  		st.PipDetailsReady = true
  		m = m.applyFilter()

  	case pm.ActionMsg:
  		if msg.Err != nil {
  			m.actionStatus = fmt.Sprintf("%s failed for %s: %v", msg.Action, msg.PackageName, msg.Err)
  			return m, nil
  		}
  		m.actionStatus = fmt.Sprintf("%s completed for %s", msg.Action, msg.PackageName)
  		return m.refresh()

  	case spinner.TickMsg:
  		var cmd tea.Cmd
  		m.spinner, cmd = m.spinner.Update(msg)
  		return m, cmd

  	case progressTick:
  		if m.allLoaded() {
  			return m, nil
  		}
  		for i := range m.states {
  			if !m.states[i].loading {
  				continue
  			}
  			p := m.states[i].progress
  			target := m.states[i].progressTarget
  			if p < target {
  				next := p + (target-p)*0.15
  				if next > target {
  					next = target
  				}
  				m.states[i].progress = next
  			}
  		}
  		return m, tickCmd()

  	case tea.KeyMsg:
  		if m.actionOverlay {
  			switch msg.String() {
  			case "esc", "n":
  				m.actionOverlay = false
  				return m, nil
  			case "enter", "y":
  				m.actionOverlay = false
  				return m, m.tabs[m.pendingTab].RunAction(m.pendingPackage, m.pendingAction)
  			default:
  				return m, nil
  			}
  		}
  		if m.themeOverlay {
  			switch msg.String() {
  			case "esc", "t":
  				m.themeOverlay = false
  				applyTheme(themes[m.themeCursor])
  				return m, nil
  			case "enter":
  				m.themeOverlay = false
  				applyTheme(themes[m.themeCursor])
  				return m, nil
  			case "up":
  				if m.themeCursor > 0 {
  					m.themeCursor--
  					applyTheme(themes[m.themeCursor])
  				}
  				return m, nil
  			case "down":
  				if m.themeCursor < len(themes)-1 {
  					m.themeCursor++
  					applyTheme(themes[m.themeCursor])
  				}
  				return m, nil
  			default:
  				return m, nil
  			}
  		}

  		if m.searchActive {
  			switch msg.String() {
  			case "esc":
  				m.searchActive = false
  				m.searchQuery = ""
  				m = m.applyFilter()
  				return m, nil
  			case "enter":
  				m.searchActive = false
  				return m, nil
  			case "left", "right":
  				return m, nil
  			case "backspace":
  				if len(m.searchQuery) > 0 {
  					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
  					m = m.applyFilter()
  				}
  				return m, nil
  			case "up":
  				if m.allMode {
  					if m.allCursor > 0 {
  						m.allCursor--
  					}
  				} else {
  					st := &m.states[m.activeTab]
  					if st.cursor > 0 {
  						st.cursor--
  						return m, m.selectPackageCmd()
  					}
  				}
  				return m, nil
  			case "down":
  				if m.allMode {
  					if m.allCursor < len(m.allDisplayPackages)-1 {
  						m.allCursor++
  					}
  				} else {
  					st := &m.states[m.activeTab]
  					if st.cursor < len(st.displayPackages)-1 {
  						st.cursor++
  						return m, m.selectPackageCmd()
  					}
  				}
  				return m, nil
  			default:
  				if len(msg.String()) == 1 && msg.String()[0] >= 32 {
  					m.searchQuery += msg.String()
  					m = m.applyFilter()
  				}
  				return m, nil
  			}
  		}

  		switch msg.String() {
  		case "q", "ctrl+c":
  			return m, tea.Quit

  		case "t":
  			if m.allLoaded() {
  				m.themeOverlay = true
  				m.themeCursor = 0
  				for i, t := range themes {
  					if t == currentTheme {
  						m.themeCursor = i
  						break
  					}
  				}
  				return m, nil
  			}

  		case "/":
  			m.searchActive = true
  			m.searchQuery = ""

  		case "r":
  			return m.refresh()

  		case "o":
  			m.outdatedOnly = !m.outdatedOnly
  			m = m.applyFilter()

  		case "u", "x":
  			if tabIndex, packageName, ok := m.selectedPackage(); ok {
  				m.pendingTab = tabIndex
  				m.pendingPackage = packageName
  				m.pendingAction = pm.Upgrade
  				if msg.String() == "x" {
  					m.pendingAction = pm.Remove
  				}
  				m.actionOverlay = true
  			}

  		case "left":
  			if m.allMode {
  				return m, nil
  			}
  			if m.activeTab > 0 {
  				m.activeTab--
  				m.searchActive = false
  				m.searchQuery = ""
  				st := &m.states[m.activeTab]
  				if st.loading && len(st.packages) == 0 && st.err == nil {
  					return m, m.tabs[m.activeTab].ListInstalled()
  				}
  				return m, m.selectPackageCmd()
  			}
  			if m.activeTab == 0 {
  				m.allMode = true
  				m.searchActive = false
  				m.searchQuery = ""
  				return m.applyFilter(), nil
  			}

  		case "right":
  			if m.allMode {
  				m.allMode = false
  				m.activeTab = 0
  				m.searchActive = false
  				m.searchQuery = ""
  				m = m.applyFilter()
  				return m, m.selectPackageCmd()
  			}
  			if m.activeTab < len(m.tabs)-1 {
  				m.activeTab++
  				m.searchActive = false
  				m.searchQuery = ""
  				st := &m.states[m.activeTab]
  				if st.loading && len(st.packages) == 0 && st.err == nil {
  					return m, m.tabs[m.activeTab].ListInstalled()
  				}
  				return m, m.selectPackageCmd()
  			}

  		case "up":
  			if m.allMode {
  				if m.allCursor > 0 {
  					m.allCursor--
  				}
  			} else {
  				st := &m.states[m.activeTab]
  				if st.cursor > 0 {
  					st.cursor--
  					return m, m.selectPackageCmd()
  				}
  			}

  		case "down":
  			if m.allMode {
  				if m.allCursor < len(m.allDisplayPackages)-1 {
  					m.allCursor++
  				}
  			} else {
  				st := &m.states[m.activeTab]
  				if st.cursor < len(st.displayPackages)-1 {
  					st.cursor++
  					return m, m.selectPackageCmd()
  				}
  			}
  		}
  	}

  	return m, nil
  }
  ```

- [ ] **Step 2: Verify compilation of update.go**
  Run: `go build ./...`
  Expected: PASS

- [ ] **Step 3: Commit update.go creation**
  Run:
  ```bash
  git add internal/app/update.go
  git commit -m "feat: add update.go containing the bubbletea update event loop"
  ```

---

### Task 4: UI Layout and View Coordinators (`internal/app/view.go`)

**Files:**
- Create: `internal/app/view.go`

**Interfaces:**
- Consumes: `app.Model` state
- Produces: `app.Model.View() string`

- [ ] **Step 1: Write view.go grid layout and loading renderers**
  Create `internal/app/view.go` containing core layout view builders:
  ```go
  package app

  import (
  	"fmt"
  	"strings"

  	"github.com/charmbracelet/lipgloss"
  )

  func (m Model) View() string {
  	if m.width == 0 {
  		return ""
  	}

  	if !m.allLoaded() {
  		return m.renderLoading()
  	}

  	if !m.allMode {
  		st := m.states[m.activeTab]
  		if st.err != nil {
  			return ErrorStyle.Render(fmt.Sprintf("Error: %v", st.err))
  		}
  	}

  	if m.width < 60 {
  		return m.listViewFallback()
  	}

  	contentWidth := m.width - 6
  	leftWidth := int(float64(contentWidth) * 0.35)
  	rightWidth := contentWidth - leftWidth

  	searchLine := m.renderSearchBar(contentWidth, m.searchActive)
  	searchOffset := strings.Count(searchLine, "\n") + 1

  	boxHeight := max(0, m.height-12-searchOffset)

  	leftPanel := m.renderLeftPanel(leftWidth, boxHeight)
  	rightPanel := m.renderRightPanel(rightWidth)

  	leftStyled := lipgloss.NewStyle().Width(leftWidth).Height(boxHeight).Render(leftPanel)
  	rightStyled := lipgloss.NewStyle().Width(rightWidth).Height(boxHeight).Render(rightPanel)

  	top := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

  	var bodyParts []string
  	bodyParts = append(bodyParts, m.renderHeader())
  	bodyParts = append(bodyParts, "")
  	bodyParts = append(bodyParts, m.renderTabBar(contentWidth))
  	bodyParts = append(bodyParts, "")
  	bodyParts = append(bodyParts, searchLine)
  	bodyParts = append(bodyParts, "")
  	bodyParts = append(bodyParts, top)
  	bodyParts = append(bodyParts, m.renderFooter())

  	body := lipgloss.JoinVertical(lipgloss.Left, bodyParts...)

  	rendered := docStyle.Render(body)

  	if m.themeOverlay {
  		return m.renderThemeOverlay()
  	}
  	if m.actionOverlay {
  		return m.renderActionOverlay()
  	}

  	return rendered
  }

  func (m Model) renderHeader() string {
  	label := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary).Render("devpkgs — a terminal dashboard for everything you've installed ")

  	sparkText := ""
  	sparkW := min(20, max(1, (m.width-20)/3))
  	if len(m.sparklineHistory) > 1 {
  		maxVal := 0.0
  		for _, v := range m.sparklineHistory {
  			if v > maxVal {
  				maxVal = v
  			}
  		}
  		if maxVal == 0 {
  			maxVal = 1
  		}
  		norm := make([]float64, len(m.sparklineHistory))
  		for i, v := range m.sparklineHistory {
  			norm[i] = v / maxVal
  		}
  		sparkH := 2
  		sparkText = RenderBrailleSparkline(norm, sparkW, sparkH)
  	}

  	if sparkText != "" {
  		return label + "\n" + sparkText
  	}
  	return label
  }

  func (m Model) renderLoading() string {
  	doneStyle := lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true)
  	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
  	fillStyle := lipgloss.NewStyle().Foreground(currentTheme.DetailText)
  	emptyStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText)

  	spinnerLine := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render(
  		m.spinner.View() + " Loading packages...",
  	)

  	sparkW := min(30, max(5, m.width/3))
  	var sparkArea string
  	if len(m.sparklineHistory) > 1 {
  		maxVal := 0.0
  		for _, v := range m.sparklineHistory {
  			if v > maxVal {
  				maxVal = v
  			}
  		}
  		if maxVal == 0 {
  			maxVal = 1
  		}
  		norm := make([]float64, len(m.sparklineHistory))
  		for i, v := range m.sparklineHistory {
  			norm[i] = v / maxVal
  		}
  		sparkH := 3
  		sparkArea = RenderBrailleSparkline(norm, sparkW, sparkH)
  	}

  	var lines []string
  	lines = append(lines, "")
  	lines = append(lines, "  "+spinnerLine)
  	lines = append(lines, "")

  	for i, tab := range m.tabs {
  		name := strings.ToUpper(tab.Name())
  		label := labelStyle.Render(name)
  		st := m.states[i]

  		if !st.loading {
  			lines = append(lines, "  "+label+"  "+doneStyle.Render("[✓]"))
  			continue
  		}

  		n := int(st.progress * 20)
  		if n > 20 { n = 20 }
  		bar := "[" + fillStyle.Render(strings.Repeat("█", n)) + emptyStyle.Render(strings.Repeat("░", 20-n)) + "]"
  		lines = append(lines, "  "+label+"  "+bar)
  	}

  	if sparkArea != "" {
  		lines = append(lines, "")
  		lines = append(lines, sparkArea)
  	}

  	return LoadingStyle.Render(strings.Join(lines, "\n"))
  }

  func (m Model) renderSearchBar(width int, focused bool) string {
  	borderColor := currentTheme.Primary
  	if !focused {
  		borderColor = currentTheme.Muted
  	}
  	border := lipgloss.NewStyle().Foreground(borderColor)
  	violetBold := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)

  	badge := "Search"
  	top := border.Render("╭━ ") +
  		violetBold.Render(badge) +
  		border.Render(" "+strings.Repeat("━", max(0, width-5-lipgloss.Width(badge)))+"╮")

  	inner := width - 4

  	var inputLine string
  	if focused {
  		cursor := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render("█")
  		if m.searchQuery == "" {
  			inputLine = cursor + " " + SearchPlaceholderStyle.Render("Search packages...")
  		} else {
  			inputLine = DetailValueStyle.Render(m.searchQuery) + cursor
  		}
  	} else {
  		if m.searchQuery == "" {
  			inputLine = SearchPlaceholderStyle.Render("Search packages...")
  		} else {
  			inputLine = DetailValueStyle.Render(m.searchQuery)
  		}
  	}

  	padded := lipgloss.NewStyle().Width(inner).Render(inputLine)
  	body := border.Render("│ ") + padded + border.Render(" │")
  	bottom := border.Render("╰" + strings.Repeat("─", width-2) + "╯")

  	return strings.Join([]string{top, body, bottom}, "\n")
  }
  ```

- [ ] **Step 2: Verify compilation**
  Run: `go build ./...`
  Expected: PASS

- [ ] **Step 3: Commit view.go creation**
  Run:
  ```bash
  git add internal/app/view.go
  git commit -m "feat: add view.go containing core layouts and loaders"
  ```

---

### Task 5: Left/Right View Panels and Overlays (`internal/app/panels.go` & `internal/app/overlays.go`)

**Files:**
- Create: `internal/app/panels.go`
- Create: `internal/app/overlays.go`

**Interfaces:**
- Consumes: `app.Model` state
- Produces: Rendering functions for details, package lists, and confirmation/theme screens

- [ ] **Step 1: Write panels.go for Left/Right panels**
  Create `internal/app/panels.go`:
  ```go
  package app

  import (
  	"fmt"
  	"strings"

  	"github.com/charmbracelet/lipgloss"
  )

  func (m Model) renderLeftPanel(width int, boxHeight int) string {
  	if m.allMode {
  		return m.renderAllLeftPanel(width, boxHeight)
  	}
  	st := m.states[m.activeTab]
  	visibleHeight := boxHeight - 2
  	innerWidth := width - 4

  	start := 0
  	if st.cursor >= visibleHeight {
  		start = st.cursor - visibleHeight + 1
  	}
  	end := start + visibleHeight
  	if end > len(st.displayPackages) {
  		end = len(st.displayPackages)
  	}

  	var listItems []string
  	for i := start; i < end; i++ {
  		pkg := st.displayPackages[i]
  		if i == st.cursor {
  			style := SelectedItemStyle.Width(innerWidth)
  			listItems = append(listItems, style.Render(pkg))
  		} else {
  			listItems = append(listItems, ItemStyle.Render(pkg))
  		}
  	}

  	boxTitle := fmt.Sprintf("Packages (%d)", len(st.displayPackages))
  	return renderPaneBox(width, boxTitle, strings.Join(listItems, "\n"))
  }

  func (m Model) renderAllLeftPanel(width int, boxHeight int) string {
  	visibleHeight := boxHeight - 2

  	start := 0
  	if m.allCursor >= visibleHeight {
  		start = m.allCursor - visibleHeight + 1
  	}
  	end := start + visibleHeight
  	if end > len(m.allDisplayPackages) {
  		end = len(m.allDisplayPackages)
  	}

  	typeWidth := 5

  	pmBadge := func(name string) lipgloss.Style {
  		switch name {
  		case "brew":
  			return lipgloss.NewStyle().
  				Width(typeWidth).
  				Background(currentTheme.Primary).
  				Foreground(currentTheme.SelectedFg)
  		case "npm":
  			return lipgloss.NewStyle().
  				Width(typeWidth).
  				Background(darkenHex(string(currentTheme.Primary), 0.65)).
  				Foreground(currentTheme.Text)
  		case "pip":
  			return lipgloss.NewStyle().
  				Width(typeWidth).
  				Background(darkenHex(string(currentTheme.Success), 0.65)).
  				Foreground(currentTheme.Text)
  		default:
  			return lipgloss.NewStyle().
  				Width(typeWidth).
  				Background(currentTheme.Muted).
  				Foreground(currentTheme.Text)
  		}
  	}

  	var listItems []string
  	for i := start; i < end; i++ {
  		pkg := m.allDisplayPackages[i]
  		origin := m.allPackageOrigin[pkg]

  		originRendered := pmBadge(origin).Render(origin)
  		var pkgRendered string
  		if i == m.allCursor {
  			pkgRendered = SelectedItemStyle.Render(pkg)
  		} else {
  			pkgRendered = ItemStyle.Render(pkg)
  		}

  		listItems = append(listItems, originRendered+" "+pkgRendered)
  	}

  	boxTitle := fmt.Sprintf("All Packages (%d)", len(m.allDisplayPackages))

  	badgeLabel := "PM"
  	violetStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
  	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)
  	content := strings.Join(listItems, "\n")

  	w1 := lipgloss.Width(badgeLabel)
  	w2 := lipgloss.Width(boxTitle)
  	n := max(0, width-9-w1-w2)
  	top := border.Render("╭━ ") +
  		violetStyle.Render(badgeLabel) +
  		border.Render(" ━━ ") +
  		violetStyle.Render(boxTitle) +
  		border.Render(" "+strings.Repeat("━", n)+"╮")

  	inner := width - 4
  	lines := strings.Split(content, "\n")
  	var body []string
  	for _, line := range lines {
  		padded := lipgloss.NewStyle().Width(inner).Render(line)
  		body = append(body, border.Render("│ ")+padded+border.Render(" │"))
  	}

  	bottom := border.Render("╰" + strings.Repeat("─", width-2) + "╯")

  	return strings.Join(append([]string{top}, append(body, bottom)...), "\n")
  }

  func (m Model) renderRightPanel(width int) string {
  	if m.allMode {
  		return m.renderAllDetail(width)
  	}
  	st := m.states[m.activeTab]

  	if len(st.displayPackages) == 0 {
  		return renderPaneBox(width, "Details",
  			lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("No packages match your query"))
  	}

  	if st.Brew != nil {
  		return m.renderBrewDetail(width, st)
  	}
  	if m.tabs[m.activeTab].Name() == "npm" {
  		return m.renderNpmDetail(width, st)
  	}
  	if m.tabs[m.activeTab].Name() == "pip" {
  		return m.renderPipDetail(width, st)
  	}
  	return renderPaneBox(width, "Details",
  		lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("Details coming soon for this package manager"))
  }

  func (m Model) renderAllDetail(width int) string {
  	if len(m.allDisplayPackages) == 0 {
  		return renderPaneBox(width, "Details",
  			lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("No packages match your query"))
  	}
  	pkgName := m.allDisplayPackages[m.allCursor]
  	origin := m.allPackageOrigin[pkgName]

  	originIdx := -1
  	for i, t := range m.tabs {
  		if t.Name() == origin {
  			originIdx = i
  			break
  		}
  	}
  	if originIdx == -1 {
  		return renderPaneBox(width, "Details",
  			lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("Unknown package origin"))
  	}

  	st := m.states[originIdx]

  	cursor := -1
  	for i, p := range st.displayPackages {
  		if p == pkgName {
  			cursor = i
  			break
  		}
  	}
  	if cursor == -1 {
  		return renderPaneBox(width, "Details",
  			lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("Package details loading..."))
  	}

  	if st.Brew != nil {
  		localSt := st
  		localSt.cursor = cursor
  		if f, ok := localSt.Brew.FormulaeMap[pkgName]; ok {
  			localSt.Brew.Info = &f
  		} else {
  			localSt.Brew.Info = nil
  		}
  		return m.renderBrewDetail(width, localSt)
  	}

  	localSt := st
  	localSt.cursor = cursor
  	if origin == "pip" {
  		return m.renderPipDetail(width, localSt)
  	}
  	return m.renderNpmDetail(width, localSt)
  }

  func (m Model) renderBrewDetail(width int, st TabState) string {
  	if st.Brew == nil { return "" }
  	pkgName := st.displayPackages[st.cursor]

  	var contentLines []string
  	contentLines = append(contentLines, "")
  	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
  	contentLines = append(contentLines, "")

  	if st.Brew.APIErr != nil {
  		contentLines = append(contentLines, DetailValueStyle.Render("  Formula data unavailable"))
  	} else if st.Brew.Info != nil {
  		info := st.Brew.Info
  		if info.Desc != "" {
  			contentLines = append(contentLines, renderSection(width, "Description", info.Desc))
  			contentLines = append(contentLines, "")
  		}

  		type sectionData struct {
  			title string
  			lines []string
  		}
  		var sections []sectionData
  		var allWidths []int

  		var pkgPairs [][2]string
  		if ver, ok := st.Brew.InstalledVersions[pkgName]; ok {
  			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
  		}
  		if info.Versions.Stable != "" {
  			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Versions.Stable})
  		}
  		if path, ok := st.Brew.InstallPaths[pkgName]; ok && path != "" {
  			pkgPairs = append(pkgPairs, [2]string{"Path", path})
  		}
  		if len(pkgPairs) > 0 {
  			maxLabel := 0
  			for _, p := range pkgPairs {
  				w := lipgloss.Width(p[0])
  				if w > maxLabel { maxLabel = w }
  			}
  			var lines []string
  			for _, p := range pkgPairs {
  				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
  				value := DetailValueStyle.Render(p[1])
  				line := label + "  " + value
  				allWidths = append(allWidths, lipgloss.Width(line))
  				lines = append(lines, line)
  			}
  			sections = append(sections, sectionData{"Package", lines})
  		}

  		var metaPairs [][2]string
  		if info.License != "" {
  			metaPairs = append(metaPairs, [2]string{"License", info.License})
  		}
  		if info.Homepage != "" {
  			metaPairs = append(metaPairs, [2]string{"Homepage", info.Homepage})
  		}
  		if st.Brew.Sizes != nil {
  			if size, ok := st.Brew.Sizes[pkgName]; ok && size > 0 {
  				metaPairs = append(metaPairs, [2]string{"Size", humanSize(size)})
  			}
  		}
  		if len(metaPairs) > 0 {
  			maxLabel := 0
  			for _, p := range metaPairs {
  				w := lipgloss.Width(p[0])
  				if w > maxLabel { maxLabel = w }
  			}
  			var lines []string
  			for _, p := range metaPairs {
  				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
  				var value string
  				if p[0] == "Homepage" {
  					value = LinkStyle.Render(p[1])
  				} else {
  					value = DetailValueStyle.Render(p[1])
  				}
  				line := label + "  " + value
  				allWidths = append(allWidths, lipgloss.Width(line))
  				lines = append(lines, line)
  			}
  			sections = append(sections, sectionData{"Metadata", lines})
  		}

  		if len(info.Dependencies) > 0 {
  			line := DetailValueStyle.Render(strings.Join(info.Dependencies, ", "))
  			allWidths = append(allWidths, lipgloss.Width(line))
  			sections = append(sections, sectionData{"Dependencies", []string{line}})
  		}

  		sectionWidth := width
  		if len(allWidths) > 0 {
  			maxW := 0
  			for _, w := range allWidths {
  				if w > maxW { maxW = w }
  			}
  			sectionWidth = min(width, max(maxW+4, 6))
  		}

  		for _, s := range sections {
  			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
  		}
  	} else {
  		contentLines = append(contentLines, DetailValueStyle.Render("  No formula data available"))
  	}
  	contentLines = append(contentLines, "")
  	return renderPaneBox(width, "Details", strings.Join(contentLines, "\n"))
  }

  func (m Model) renderNpmDetail(width int, st TabState) string {
  	pkgName := st.displayPackages[st.cursor]
  	var contentLines []string
  	contentLines = append(contentLines, "")
  	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
  	contentLines = append(contentLines, "")

  	if !st.NpmDetailsReady {
  		contentLines = append(contentLines, DetailValueStyle.Render("  Loading registry data..."))
  	} else if info, ok := st.NpmDetails[pkgName]; ok {
  		if info.Description != "" {
  			contentLines = append(contentLines, renderSection(width, "Description", info.Description))
  			contentLines = append(contentLines, "")
  		}

  		type sectionData struct {
  			title string
  			lines []string
  		}
  		var sections []sectionData
  		var allWidths []int

  		var pkgPairs [][2]string
  		if ver, ok := st.versions[pkgName]; ok {
  			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
  		}
  		if info.Version != "" {
  			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Version})
  		}
  		if len(pkgPairs) > 0 {
  			maxLabel := 0
  			for _, p := range pkgPairs {
  				w := lipgloss.Width(p[0])
  				if w > maxLabel { maxLabel = w }
  			}
  			var lines []string
  			for _, p := range pkgPairs {
  				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
  				value := DetailValueStyle.Render(p[1])
  				line := label + "  " + value
  				allWidths = append(allWidths, lipgloss.Width(line))
  				lines = append(lines, line)
  			}
  			sections = append(sections, sectionData{"Package", lines})
  		}

  		var metaPairs [][2]string
  		if info.License != "" {
  			metaPairs = append(metaPairs, [2]string{"License", info.License})
  		}
  		if info.Homepage != "" {
  			metaPairs = append(metaPairs, [2]string{"Homepage", info.Homepage})
  		}
  		if info.Dist != nil && info.Dist.UnpackedSize > 0 {
  			metaPairs = append(metaPairs, [2]string{"Size", humanSize(info.Dist.UnpackedSize)})
  		}
  		if len(metaPairs) > 0 {
  			maxLabel := 0
  			for _, p := range metaPairs {
  				w := lipgloss.Width(p[0])
  				if w > maxLabel { maxLabel = w }
  			}
  			var lines []string
  			for _, p := range metaPairs {
  				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
  				var value string
  				if p[0] == "Homepage" {
  					value = LinkStyle.Render(p[1])
  				} else {
  					value = DetailValueStyle.Render(p[1])
  				}
  				line := label + "  " + value
  				allWidths = append(allWidths, lipgloss.Width(line))
  				lines = append(lines, line)
  			}
  			sections = append(sections, sectionData{"Metadata", lines})
  		}

  		sectionWidth := width
  		if len(allWidths) > 0 {
  			maxW := 0
  			for _, w := range allWidths {
  				if w > maxW { maxW = w }
  			}
  			sectionWidth = min(width, max(maxW+4, 6))
  		}

  		for _, s := range sections {
  			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
  		}
  	} else {
  		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
  	}
  	contentLines = append(contentLines, "")
  	return renderPaneBox(width, "Details", strings.Join(contentLines, "\n"))
  }

  func (m Model) renderPipDetail(width int, st TabState) string {
  	pkgName := st.displayPackages[st.cursor]
  	var contentLines []string
  	contentLines = append(contentLines, "")
  	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
  	contentLines = append(contentLines, "")

  	if !st.PipDetailsReady {
  		contentLines = append(contentLines, DetailValueStyle.Render("  Loading registry data..."))
  	} else if info, ok := st.PipDetails[pkgName]; ok {
  		if info.Summary != "" {
  			contentLines = append(contentLines, renderSection(width, "Description", info.Summary))
  			contentLines = append(contentLines, "")
  		}

  		type sectionData struct {
  			title string
  			lines []string
  		}
  		var sections []sectionData
  		var allWidths []int

  		var pkgPairs [][2]string
  		if ver, ok := st.versions[pkgName]; ok {
  			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
  		}
  		if info.Version != "" {
  			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Version})
  		}
  		if len(pkgPairs) > 0 {
  			maxLabel := 0
  			for _, p := range pkgPairs {
  				w := lipgloss.Width(p[0])
  				if w > maxLabel { maxLabel = w }
  			}
  			var lines []string
  			for _, p := range pkgPairs {
  				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
  				value := DetailValueStyle.Render(p[1])
  				line := label + "  " + value
  				allWidths = append(allWidths, lipgloss.Width(line))
  				lines = append(lines, line)
  			}
  			sections = append(sections, sectionData{"Package", lines})
  		}

  		var metaPairs [][2]string
  		if info.License != "" {
  			metaPairs = append(metaPairs, [2]string{"License", info.License})
  		}
  		if info.HomePage != "" {
  			metaPairs = append(metaPairs, [2]string{"Homepage", info.HomePage})
  		}
  		if info.Author != "" {
  			metaPairs = append(metaPairs, [2]string{"Author", info.Author})
  		}
  		if len(metaPairs) > 0 {
  			maxLabel := 0
  			for _, p := range metaPairs {
  				w := lipgloss.Width(p[0])
  				if w > maxLabel { maxLabel = w }
  			}
  			var lines []string
  			for _, p := range metaPairs {
  				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
  				var value string
  				if p[0] == "Homepage" {
  					value = LinkStyle.Render(p[1])
  				} else {
  					value = DetailValueStyle.Render(p[1])
  				}
  				line := label + "  " + value
  				allWidths = append(allWidths, lipgloss.Width(line))
  				lines = append(lines, line)
  			}
  			sections = append(sections, sectionData{"Metadata", lines})
  		}

  		sectionWidth := width
  		if len(allWidths) > 0 {
  			maxW := 0
  			for _, w := range allWidths {
  				if w > maxW { maxW = w }
  			}
  			sectionWidth = min(width, max(maxW+4, 6))
  		}

  		for _, s := range sections {
  			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
  		}
  	} else {
  		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
  	}
  	contentLines = append(contentLines, "")
  	return renderPaneBox(width, "Details", strings.Join(contentLines, "\n"))
  }

  func (m Model) listViewFallback() string {
  	var title string
  	var list string
  	visibleHeight := m.height - 8

  	if m.allMode {
  		title = TitleStyle.Render(fmt.Sprintf("devpkgs  (%d)", len(m.allPackages)))
  		start := 0
  		if m.allCursor >= visibleHeight {
  			start = m.allCursor - visibleHeight + 1
  		}
  		end := start + visibleHeight
  		if end > len(m.allDisplayPackages) {
  			end = len(m.allDisplayPackages)
  		}
  		pmBadgeFn := func(name string) lipgloss.Style {
  			switch name {
  			case "brew":
  				return lipgloss.NewStyle().
  					Width(5).
  					Background(currentTheme.Primary).
  					Foreground(currentTheme.SelectedFg)
  			case "npm":
  				return lipgloss.NewStyle().
  					Width(5).
  					Background(darkenHex(string(currentTheme.Primary), 0.65)).
  					Foreground(currentTheme.Text)
  			case "pip":
  				return lipgloss.NewStyle().
  					Width(5).
  					Background(darkenHex(string(currentTheme.Success), 0.65)).
  					Foreground(currentTheme.Text)
  			default:
  				return lipgloss.NewStyle().
  					Width(5).
  					Background(currentTheme.Muted).
  					Foreground(currentTheme.Text)
  			}
  		}
  		for i := start; i < end; i++ {
  			pkg := m.allDisplayPackages[i]
  			origin := m.allPackageOrigin[pkg]
  			originRendered := pmBadgeFn(origin).Render(origin)
  			var pkgRendered string
  			if i == m.allCursor {
  				pkgRendered = SelectedItemStyle.Render(pkg)
  			} else {
  				pkgRendered = ItemStyle.Render(pkg)
  			}
  			list += originRendered + " " + pkgRendered + "\n"
  		}
  	} else {
  		st := m.states[m.activeTab]
  		title = TitleStyle.Render(fmt.Sprintf("devpkgs  (%d)", len(st.packages)))
  		start := 0
  		if st.cursor >= visibleHeight {
  			start = st.cursor - visibleHeight + 1
  		}
  		end := start + visibleHeight
  		if end > len(st.displayPackages) {
  			end = len(st.displayPackages)
  		}
  		for i := start; i < end; i++ {
  			pkg := st.displayPackages[i]
  			if i == st.cursor {
  				list += SelectedItemStyle.Render(pkg) + "\n"
  			} else {
  				list += ItemStyle.Render(pkg) + "\n"
  			}
  		}
  	}

  	sep := lipgloss.NewStyle().
  		Foreground(currentTheme.Primary).
  		Padding(0, 1).
  		Render(strings.Repeat("━", m.width-8))

  	body := lipgloss.JoinVertical(lipgloss.Left, title, sep, list)
  	return docStyle.Render(body)
  }

  func renderSection(maxWidth int, title string, lines ...string) string {
  	violetStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
  	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)

  	maxContent := 0
  	for _, line := range lines {
  		w := lipgloss.Width(line)
  		if w > maxContent { maxContent = w }
  	}
  	boxWidth := max(maxContent+4, lipgloss.Width(title)+6)
  	boxWidth = min(boxWidth, maxWidth)

  	inner := boxWidth - 4
  	top := border.Render("╭━ ") +
  		violetStyle.Render(title) +
  		border.Render(" "+strings.Repeat("━", max(0, boxWidth-5-lipgloss.Width(title)))+"╮")

  	var body []string
  	for _, line := range lines {
  		padded := lipgloss.NewStyle().Width(inner).Render(line)
  		body = append(body, border.Render("│ ")+padded+border.Render(" │"))
  	}
  	bottom := border.Render("╰" + strings.Repeat("─", boxWidth-2) + "╯")

  	return strings.Join(append([]string{top}, append(body, bottom)...), "\n")
  }

  func renderPaneBox(width int, title string, content string) string {
  	violetStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
  	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)

  	top := border.Render("╭━ ") +
  		violetStyle.Render(title) +
  		border.Render(" "+strings.Repeat("━", max(0, width-5-lipgloss.Width(title)))+"╮")

  	inner := width - 4
  	lines := strings.Split(content, "\n")
  	var body []string
  	for _, line := range lines {
  		padded := lipgloss.NewStyle().Width(inner).Render(line)
  		body = append(body, border.Render("│ ")+padded+border.Render(" │"))
  	}
  	bottom := border.Render("╰" + strings.Repeat("─", width-2) + "╯")

  	return strings.Join(append([]string{top}, append(body, bottom)...), "\n")
  }

  func humanSize(bytes int64) string {
  	const unit = 1024
  	if bytes < unit {
  		return fmt.Sprintf("%d B", bytes)
  	}
  	div, exp := int64(unit), 0
  	for n := bytes / unit; n >= unit; n /= unit {
  		div *= unit
  		exp++
  	}
  	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
  }
  ```

- [ ] **Step 2: Write overlays.go for Dialogs & Menu screens**
  Create `internal/app/overlays.go`:
  ```go
  package app

  import (
  	"fmt"
  	"strings"

  	"github.com/charmbracelet/lipgloss"
  )

  func (m Model) renderFooter() string {
  	total := m.totalPackages()
  	countStr := ResultStyle.Render(fmt.Sprintf("%d results across all package managers", total))

  	apiErrMsg := ""
  	for i := range m.states {
  		if m.states[i].Brew != nil && m.states[i].Brew.APIErr != nil {
  			apiErrMsg = "  " + ErrorStyle.Render("API unavailable")
  			break
  		}
  	}

  	themeName := ""
  	if currentTheme != nil {
  		themeName = currentTheme.Name
  	}
  	help := FooterStyle.Render(fmt.Sprintf("[← → tabs] [/ search] [o outdated] [r refresh] [u upgrade] [x remove] [t theme %s] [q quit]", themeName))
  	status := ""
  	if m.actionStatus != "" { status = "  " + FooterStyle.Render(m.actionStatus) }
  	return countStr + apiErrMsg + status + "  " + help
  }

  func (m Model) renderActionOverlay() string {
  	action := string(m.pendingAction)
  	content := fmt.Sprintf("%s %q using %s?\n\nEnter/y: confirm   Esc/n: cancel", action, m.pendingPackage, m.tabs[m.pendingTab].Name())
  	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, renderPaneBox(60, "Confirm package action", content))
  }

  func (m Model) renderThemeOverlay() string {
  	boxW := min(52, m.width-6)
  	innerW := boxW - 4

  	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)
  	titleText := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary).Render(" Choose Theme ")
  	dashLen := max(0, boxW-4-lipgloss.Width(titleText))
  	titleLine := border.Render("╭─"+strings.Repeat("─", dashLen/2)) + titleText +
  		border.Render(strings.Repeat("─", dashLen-dashLen/2)+"─╮")

  	nameColW := 14
  	descColW := innerW - nameColW - 3

  	var items []string
  	for i, t := range themes {
  		name := lipgloss.NewStyle().Width(nameColW).Render(t.Name)
  		desc := t.Description
  		if len(desc) > descColW { desc = desc[:descColW] }

  		var line string
  		if i == m.themeCursor {
  			arrow := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render("›")
  			nameStyled := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Text).Render(name)
  			descStyled := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render(desc)
  			line = fmt.Sprintf("  %s %s %s", arrow, nameStyled, descStyled)
  		} else {
  			nameStyled := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Text).Render(name)
  			descStyled := lipgloss.NewStyle().Foreground(currentTheme.DimText).Render(desc)
  			line = fmt.Sprintf("   %s %s", nameStyled, descStyled)
  		}
  		padded := lipgloss.NewStyle().Width(innerW).Render(line)
  		items = append(items, border.Render("│ ")+padded+border.Render(" │"))
  	}
  	content := strings.Join(items, "\n")

  	bottom := border.Render("╰" + strings.Repeat("─", boxW-2) + "╯")
  	footer := lipgloss.NewStyle().
  		Foreground(currentTheme.DimText).
  		Italic(true).
  		Render("  ↑↓ navigate · enter select · esc close")

  	overlay := strings.Join([]string{titleLine, content, bottom, footer}, "\n")

  	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
  }
  ```

- [ ] **Step 3: Verify compilation of view systems**
  Run: `go build ./...`
  Expected: PASS

- [ ] **Step 4: Commit panel and overlay creations**
  Run:
  ```bash
  git add internal/app/panels.go internal/app/overlays.go
  git commit -m "feat: add panels.go and overlays.go separating visual rendering units"
  ```

---

### Task 6: Final Integration and Verification

**Files:**
- Modify: [main.go](file:///d:/Github%20repo/devpkgs/main.go)

**Interfaces:**
- Consumes: `app.New()`
- Produces: Executed Bubble Tea dashboard

- [ ] **Step 1: Check main.go structure**
  Verify that `main.go` correctly references `app.New()` and imports are standard.
  Expected: Compiled main application runs and connects app correctly.

- [ ] **Step 2: Run all tests in repository**
  Run: `go test -v ./...`
  Expected: All tests (outdated checks, winget export parser) PASS.

- [ ] **Step 3: Run the local dashboard**
  Run: `go run .`
  Expected: The dashboard launches in alt-screen mode, fetching packages in parallel. Verify rounded boxes are drawn on searching (`/`) and navigation keys function as expected.

- [ ] **Step 4: Commit final verification**
  Run:
  ```bash
  git add main.go
  git commit -m "chore: verify build and modular rewrite of TUI dashboard"
  ```
