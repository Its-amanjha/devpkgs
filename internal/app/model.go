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
	WingetDetails   map[string]*pm.WingetDetailData
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
		if m.Name() == "winget" {
			states[i].WingetDetails = make(map[string]*pm.WingetDetailData)
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
	case "winget":
		if info := m.states[tabIndex].WingetDetails[packageName]; info != nil {
			latest = info.Version
		}
	}
	return installed != "" && latest != "" && installed != latest
}

func (m Model) isUpToDate(tabIndex int, packageName string) bool {
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
	case "winget":
		if info := m.states[tabIndex].WingetDetails[packageName]; info != nil {
			latest = info.Version
		}
	}
	return installed != "" && latest != "" && installed == latest
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
	tabIndex, pkgName, ok := m.selectedPackage()
	if !ok {
		return nil
	}
	if m.tabs[tabIndex].Name() == "brew" {
		m = m.updateBrewInfo()
		return nil
	}
	if m.tabs[tabIndex].Name() == "winget" {
		st := &m.states[tabIndex]
		if st.WingetDetails == nil {
			st.WingetDetails = make(map[string]*pm.WingetDetailData)
		}
		if _, cached := st.WingetDetails[pkgName]; !cached {
			return pm.FetchWingetDetails(pkgName)
		}
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
