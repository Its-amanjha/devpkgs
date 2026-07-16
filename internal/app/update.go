package app

import (
	"fmt"
	"sort"
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
	if nr < 0 {
		nr = 0
	}
	if ng < 0 {
		ng = 0
	}
	if nb < 0 {
		nb = 0
	}
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
			pruneSelected(st)
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
			pruneSelected(st)
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

	case pm.WingetDetailMsg:
		st := &m.states[3]
		if msg.Err == nil && msg.Data != nil {
			if st.WingetDetails == nil {
				st.WingetDetails = make(map[string]*pm.WingetDetailData)
			}
			st.WingetDetails[msg.PackageID] = msg.Data
		}
		return m, nil

	case pm.RegistrySearchMsg:
		if msg.Err == nil {
			m.searchResults = append(m.searchResults, msg.Results...)
			sort.Slice(m.searchResults, func(i, j int) bool {
				return m.searchResults[i].Name < m.searchResults[j].Name
			})
		}
		m.searchActiveWorkers--
		if m.searchActiveWorkers <= 0 {
			m.searchLoading = false
			m.searchActiveWorkers = 0
		}

	case pm.LogLineMsg:
		m.logLines = append(m.logLines, msg.Line)
		return m, ListenLogs(m.logChan)

	case pm.LogFinishMsg:
		m.logActive = false
		return m, ListenLogs(m.logChan)

	case pm.ActionMsg:
		if msg.Err != nil {
			m.actionStatus = fmt.Sprintf("%s failed for %s: %v", msg.Action, msg.PackageName, msg.Err)
			if len(m.bulkQueue) > 0 {
				m.bulkErrors = append(m.bulkErrors, msg.Err.Error())
			}
			m.logLines = append(m.logLines, fmt.Sprintf("Error: %v", msg.Err))
		} else {
			m.actionStatus = fmt.Sprintf("%s completed for %s", msg.Action, msg.PackageName)
		}

		// If in bulk mode, advance to next package
		if len(m.bulkQueue) > 0 {
			m.bulkIndex++
			return m.startNextBulkAction()
		}

		return m.refreshTab(msg.Manager)

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
		if m.logOverlay {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "l":
				if !m.logActive {
					m.logOverlay = false
				}
				return m, nil
			case "up":
				m.logScrollActive = true
				contentH := min(22, m.height-6) - 4
				if contentH < 4 {
					contentH = 4
				}
				maxScroll := len(m.logLines) - contentH
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.logScrollOffset < maxScroll {
					m.logScrollOffset++
				}
				return m, nil
			case "down":
				if m.logScrollOffset > 0 {
					m.logScrollOffset--
				}
				if m.logScrollOffset == 0 {
					m.logScrollActive = false
				}
				return m, nil
			default:
				return m, nil
			}
		}
		if m.actionOverlay {
			switch msg.String() {
			case "esc", "n":
				m.actionOverlay = false
				m.bulkQueue = nil
				return m, nil
			case "y":
				m.actionOverlay = false
				m.logOverlay = true
				if len(m.bulkQueue) > 0 {
					m.bulkLogs = true
					return m.startNextBulkAction()
				}
				m.logLines = nil
				m.logScrollOffset = 0
				m.logScrollActive = false
				m.logActive = true
				m.logChan = make(chan tea.Msg, 100)
				cmd := m.tabs[m.pendingTab].RunAction(m.pendingPackage, m.pendingAction, m.logChan)
				return m, tea.Batch(cmd, ListenLogs(m.logChan))
			case "enter", "s":
				m.actionOverlay = false
				if len(m.bulkQueue) > 0 {
					m.bulkLogs = false
					return m.startNextBulkAction()
				}
				m.logLines = nil
				m.logScrollOffset = 0
				m.logScrollActive = false
				m.logActive = true
				m.logChan = make(chan tea.Msg, 100)
				cmd := m.tabs[m.pendingTab].RunAction(m.pendingPackage, m.pendingAction, m.logChan)
				return m, tea.Batch(cmd, ListenLogs(m.logChan))
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
				if m.searchTabActive {
					m.searchQuery = ""
					m = m.applyFilter()
					return m, nil
				}
				m.searchActive = false
				m.searchQuery = ""
				m = m.applyFilter()
				return m, nil
			case "enter":
				if m.searchTabActive && m.searchQuery != "" {
					m.searchLoading = true
					m.searchResults = nil
					m.searchResultCursor = 0
					m.searchActiveWorkers = 4 // 4 workers: brew, winget, npm, pip
					cmds := []tea.Cmd{
						pm.SearchNpm(m.searchQuery),
						pm.SearchWinget(m.searchQuery),
						pm.SearchPip(m.searchQuery),
					}
					// Brew search: pass formulae map if available
					var formulaeMap map[string]pm.FormulaData
					if m.states[0].Brew != nil && m.states[0].Brew.FormulaeReady {
						formulaeMap = m.states[0].Brew.FormulaeMap
					}
					cmds = append(cmds, pm.SearchBrew(m.searchQuery, formulaeMap))
					return m, tea.Batch(cmds...)
				}
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
						return m, m.selectPackageCmd()
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
						return m, m.selectPackageCmd()
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

		case "l":
			if len(m.logLines) > 0 && !m.logActive {
				m.logOverlay = true
			}
			return m, nil

		case " ":
			if !m.allMode && !m.searchTabActive {
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
			if !m.searchTabActive {
				m.searchActive = true
				m.searchQuery = ""
			}

		case "r":
			if !m.searchTabActive {
				return m.refresh()
			}

		case "o":
			if !m.searchTabActive {
				m.outdatedOnly = !m.outdatedOnly
				m = m.applyFilter()
			}

		case "u", "x":
			if m.searchTabActive {
				return m, nil
			}
			if !m.allMode && len(m.states[m.activeTab].selected) > 0 {
				st := &m.states[m.activeTab]
				var queue []string
				for _, pkg := range st.displayPackages {
					if st.selected[pkg] {
						queue = append(queue, pkg)
					}
				}
				if len(queue) < len(st.selected) {
					inQueue := make(map[string]bool)
					for _, q := range queue {
						inQueue[q] = true
					}
					var rest []string
					for pkg := range st.selected {
						if !inQueue[pkg] {
							rest = append(rest, pkg)
						}
					}
					sort.Strings(rest)
					queue = append(queue, rest...)
				}
				action := pm.Upgrade
				if msg.String() == "x" {
					action = pm.Remove
				}
				if action == pm.Upgrade {
					var filteredQueue []string
					for _, pkg := range queue {
						if !m.isUpToDate(m.activeTab, pkg) {
							filteredQueue = append(filteredQueue, pkg)
						}
					}
					queue = filteredQueue
					if len(queue) == 0 {
						m.actionStatus = "All selected packages are already up to date"
						return m, nil
					}
				}
				m.bulkQueue = queue
				m.bulkIndex = 0
				m.bulkAction = action
				m.bulkErrors = nil
				m.pendingTab = m.activeTab
				m.actionOverlay = true
				return m, nil
			}

			if tabIndex, packageName, ok := m.selectedPackage(); ok {
				if msg.String() == "u" {
					if m.isUpToDate(tabIndex, packageName) {
						m.actionStatus = fmt.Sprintf("%s is already up to date", packageName)
						return m, nil
					}
					m.pendingAction = pm.Upgrade
				} else {
					m.pendingAction = pm.Remove
				}
				m.bulkQueue = nil
				m.pendingTab = tabIndex
				m.pendingPackage = packageName
				m.actionOverlay = true
			}

		case "left":
			if m.searchTabActive {
				m.searchTabActive = false
				m.activeTab = len(m.tabs) - 1
				m.searchActive = false
				m.searchQuery = ""
				m.searchResults = nil
				m.searchResultCursor = 0
				m.searchLoading = false
				m.searchActiveWorkers = 0
				return m, m.selectPackageCmd()
			}
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
				return m.applyFilter(), m.selectPackageCmd()
			}

		case "right":
			if m.searchTabActive {
				return m, nil
			}
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
			if m.activeTab == len(m.tabs)-1 {
				m.searchTabActive = true
				m.allMode = false
				m.searchActive = true
				m.searchQuery = ""
				return m, nil
			}

		case "up":
			if m.searchTabActive {
				if m.searchResultCursor > 0 {
					m.searchResultCursor--
				}
				return m, nil
			}
			if m.allMode {
				if m.allCursor > 0 {
					m.allCursor--
					return m, m.selectPackageCmd()
				}
			} else {
				st := &m.states[m.activeTab]
				if st.cursor > 0 {
					st.cursor--
					return m, m.selectPackageCmd()
				}
			}

		case "down":
			if m.searchTabActive {
				if m.searchResultCursor < len(m.searchResults)-1 {
					m.searchResultCursor++
				}
				return m, nil
			}
			if m.allMode {
				if m.allCursor < len(m.allDisplayPackages)-1 {
					m.allCursor++
					return m, m.selectPackageCmd()
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

func pruneSelected(st *TabState) {
	if len(st.selected) == 0 {
		return
	}
	valid := make(map[string]bool)
	for _, pkg := range st.packages {
		valid[pkg] = true
	}
	for pkg := range st.selected {
		if !valid[pkg] {
			delete(st.selected, pkg)
		}
	}
}
