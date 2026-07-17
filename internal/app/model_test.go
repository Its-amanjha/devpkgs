package app

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"devpkgs/internal/pm"
)

func TestOutdatedNpmPackage(t *testing.T) {
	m := New()
	m.states[1].versions = map[string]string{"devpkgs": "1.0.0"}
	m.states[1].NpmDetails = map[string]*pm.NpmDetailData{"devpkgs": {Version: "1.1.0"}}
	if !m.isOutdated(1, "devpkgs") {
		t.Fatal("expected a newer npm version to be marked outdated")
	}
}

func TestLogOverlay(t *testing.T) {
	m := New()
	for i := range m.states {
		m.states[i].loading = false
	}
	m.width = 80
	m.height = 24
	m.logOverlay = true
	m.logActive = true
	m.logLines = []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	
	// Test that rendering works and contains the title and lines
	viewStr := m.View()
	if !strings.Contains(viewStr, "Installation Logs") {
		t.Fatal("expected view to contain 'Installation Logs'")
	}
	if !strings.Contains(viewStr, "line 1") || !strings.Contains(viewStr, "line 5") {
		t.Fatal("expected view to contain the log lines")
	}

	// Test scrolling
	m.logScrollActive = true
	m.logScrollOffset = 1
	// Add many log lines to trigger actual scrolling logic
	for i := 0; i < 30; i++ {
		m.logLines = append(m.logLines, "log line")
	}
	viewStr = m.View()
	if !strings.Contains(viewStr, "Installation Logs") {
		t.Fatal("expected view to contain 'Installation Logs' when scrolling")
	}
}

func TestSelectionAndToggle(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0

	m.states[0].packages = []string{"pkgA", "pkgB"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB"}
	m.states[0].cursor = 0

	if m.states[0].selected == nil {
		t.Fatal("expected selected map to be initialized")
	}

	// Simulating space key press to toggle pkgA
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd != nil {
		t.Fatal("expected no command on space toggle")
	}
	m = updatedModel.(Model)

	if !m.states[0].selected["pkgA"] {
		t.Fatal("expected pkgA to be selected after space key press")
	}

	// Press space again to toggle off
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updatedModel.(Model)
	if m.states[0].selected["pkgA"] {
		t.Fatal("expected pkgA to be deselected after second space key press")
	}
}

func TestCheckboxAndFooterRendering(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0
	m.width = 80
	m.height = 24

	m.states[0].packages = []string{"pkgA", "pkgB"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB"}
	m.states[0].cursor = 0
	m.states[0].loading = false

	// Initially no checkboxes should be rendered
	leftPanelStr := m.renderLeftPanel(40, 20)
	if strings.Contains(leftPanelStr, "[ ]") || strings.Contains(leftPanelStr, "[✓]") {
		t.Fatal("expected no checkboxes to render when selected map is empty")
	}

	// Select pkgA
	m.states[0].selected["pkgA"] = true

	// Now checkboxes should be rendered
	leftPanelStr = m.renderLeftPanel(40, 20)
	if !strings.Contains(leftPanelStr, "[✓] pkgA") {
		t.Fatal("expected selected package pkgA to render with [✓] prefix")
	}
	if !strings.Contains(leftPanelStr, "[ ] pkgB") {
		t.Fatal("expected unselected package pkgB to render with [ ] prefix")
	}

	// Footer should show the selection count
	footerStr := m.renderFooter()
	if !strings.Contains(footerStr, "1 selected") {
		t.Fatal("expected footer to contain '1 selected'")
	}
}

func TestBulkQueueAndConfirmation(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0
	m.width = 80
	m.height = 24
	for i := range m.states {
		m.states[i].loading = false
	}

	m.states[0].packages = []string{"pkgA", "pkgB", "pkgC"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB", "pkgC"}
	m.states[0].cursor = 0

	// Select pkgC and pkgA
	m.states[0].selected["pkgC"] = true
	m.states[0].selected["pkgA"] = true

	// Simulate pressing "u" key
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Fatal("expected no command on bulk action trigger")
	}
	m2 := updatedModel.(Model)

	if !m2.actionOverlay {
		t.Fatal("expected actionOverlay to be true")
	}
	if m2.bulkAction != pm.Upgrade {
		t.Fatalf("expected bulkAction to be Upgrade, got %v", m2.bulkAction)
	}
	if m2.pendingTab != 0 {
		t.Fatalf("expected pendingTab to be 0, got %d", m2.pendingTab)
	}
	if len(m2.bulkQueue) != 2 {
		t.Fatalf("expected bulkQueue to have 2 packages, got %d", len(m2.bulkQueue))
	}
	// order should match displayPackages: pkgA then pkgC
	if m2.bulkQueue[0] != "pkgA" || m2.bulkQueue[1] != "pkgC" {
		t.Fatalf("expected bulkQueue to be [pkgA, pkgC], got %v", m2.bulkQueue)
	}

	// Test the confirmation rendering
	viewStr := m2.View()
	if !strings.Contains(viewStr, "Confirm bulk action") {
		t.Fatal("expected action overlay to show 'Confirm bulk action' title")
	}
	if !strings.Contains(viewStr, "Upgrade 2 packages using brew?") {
		t.Fatal("expected action overlay to show bulk execution text")
	}

	// Now test "x" trigger
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m3 := updatedModel.(Model)
	if m3.bulkAction != pm.Remove {
		t.Fatalf("expected bulkAction to be Remove, got %v", m3.bulkAction)
	}

	// Now test single package fallback (empty selection)
	m.states[0].selected = make(map[string]bool)
	m.states[0].versions = map[string]string{"pkgA": "1.0.0"}
	m.states[0].Brew.FormulaeMap = map[string]pm.FormulaData{
		"pkgA": {
			Versions: struct {
				Stable string `json:"stable"`
			}{
				Stable: "1.1.0",
			},
		},
	}
	m.states[0].cursor = 0 // points to pkgA

	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m4 := updatedModel.(Model)
	if len(m4.bulkQueue) != 0 {
		t.Fatalf("expected bulkQueue to be empty, got %v", m4.bulkQueue)
	}
	if m4.pendingPackage != "pkgA" {
		t.Fatalf("expected pendingPackage to be pkgA, got %s", m4.pendingPackage)
	}
}

func TestBulkUpgradeSkipUpToDate(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0
	for i := range m.states {
		m.states[i].loading = false
	}

	m.states[0].packages = []string{"pkgA", "pkgB"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB"}
	m.states[0].versions = map[string]string{"pkgA": "1.0.0", "pkgB": "1.0.0"}
	m.states[0].Brew.FormulaeMap = map[string]pm.FormulaData{
		"pkgA": {
			Versions: struct {
				Stable string `json:"stable"`
			}{
				Stable: "1.0.0", // Up-to-date
			},
		},
		"pkgB": {
			Versions: struct {
				Stable string `json:"stable"`
			}{
				Stable: "1.1.0", // Outdated (needs upgrade)
			},
		},
	}

	// Case 1: Select both pkgA (up-to-date) and pkgB (outdated).
	// Only pkgB should end up in the bulk upgrade queue.
	m.states[0].selected = map[string]bool{
		"pkgA": true,
		"pkgB": true,
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m2 := updatedModel.(Model)

	if !m2.actionOverlay {
		t.Fatal("expected actionOverlay to be true when there is at least one package to upgrade")
	}
	if len(m2.bulkQueue) != 1 {
		t.Fatalf("expected bulkQueue to have exactly 1 package, got %d (%v)", len(m2.bulkQueue), m2.bulkQueue)
	}
	if m2.bulkQueue[0] != "pkgB" {
		t.Fatalf("expected bulkQueue to contain only 'pkgB', got %v", m2.bulkQueue)
	}

	// Test cancellation clears bulk queue (Finding 2)
	cancelledModel, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m3 := cancelledModel.(Model)
	if m3.actionOverlay {
		t.Fatal("expected actionOverlay to be false after cancel")
	}
	if m3.bulkQueue != nil {
		t.Fatalf("expected bulkQueue to be nil after cancel, got %v", m3.bulkQueue)
	}

	// Case 2: Select only pkgA (up-to-date).
	// The queue should be empty after filtering, overlay shouldn't open,
	// and actionStatus should indicate they are already up to date.
	m.states[0].selected = map[string]bool{
		"pkgA": true,
	}

	updatedModel2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m4 := updatedModel2.(Model)

	if m4.actionOverlay {
		t.Fatal("expected actionOverlay to be false when all selected packages are up-to-date")
	}
	if m4.actionStatus != "All selected packages are already up to date" {
		t.Fatalf("expected actionStatus 'All selected packages are already up to date', got '%s'", m4.actionStatus)
	}
}

func TestSequentialBulkActionExecution(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0
	for i := range m.states {
		m.states[i].loading = false
	}

	m.states[0].packages = []string{"pkgA", "pkgB"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB"}
	m.states[0].cursor = 0

	// Select pkgA and pkgB
	m.states[0].selected["pkgA"] = true
	m.states[0].selected["pkgB"] = true

	// Simulate pressing "x" key to uninstall bulk
	mModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = mModel.(Model)

	if !m.actionOverlay {
		t.Fatal("expected actionOverlay to be true")
	}
	if len(m.bulkQueue) != 2 {
		t.Fatalf("expected 2 packages in bulkQueue, got %d", len(m.bulkQueue))
	}

	// Confirm with "y" (logs enabled)
	mModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = mModel.(Model)

	if m.actionOverlay {
		t.Fatal("expected actionOverlay to be false after confirmation")
	}
	if !m.logOverlay {
		t.Fatal("expected logOverlay to be true since we pressed 'y'")
	}
	if !m.bulkLogs {
		t.Fatal("expected bulkLogs to be true")
	}
	if m.bulkIndex != 0 {
		t.Fatalf("expected bulkIndex to be 0, got %d", m.bulkIndex)
	}
	if cmd == nil {
		t.Fatal("expected a non-nil batch command to start run")
	}

	// Simulate completion of the first package (pkgA)
	actionMsg := pm.ActionMsg{
		Manager:     "brew",
		PackageName: "pkgA",
		Action:      pm.Remove,
		Err:         nil,
	}
	mModel, cmd = m.Update(actionMsg)
	m = mModel.(Model)

	// Since there is a second package (pkgB), bulkIndex should advance to 1, and start next action
	if m.bulkIndex != 1 {
		t.Fatalf("expected bulkIndex to advance to 1, got %d", m.bulkIndex)
	}
	if len(m.bulkQueue) != 2 {
		t.Fatal("expected bulkQueue to still have 2 items")
	}
	if cmd == nil {
		t.Fatal("expected a non-nil batch command to run next bulk action")
	}

	// Simulate completion of the second package (pkgB)
	actionMsg2 := pm.ActionMsg{
		Manager:     "brew",
		PackageName: "pkgB",
		Action:      pm.Remove,
		Err:         nil,
	}
	mModel, cmd = m.Update(actionMsg2)
	m = mModel.(Model)

	// Bulk queue is completed: bulkQueue should be nil, selections cleared
	if m.bulkQueue != nil {
		t.Fatalf("expected bulkQueue to be nil after completion, got %v", m.bulkQueue)
	}
	if len(m.states[0].selected) != 0 {
		t.Fatalf("expected selections to be cleared, got %v", m.states[0].selected)
	}
	if !strings.Contains(m.actionStatus, "Bulk remove completed") {
		t.Fatalf("expected action status to say bulk completed, got '%s'", m.actionStatus)
	}
	if cmd == nil {
		t.Fatal("expected tab list refresh command upon bulk completion")
	}
}

func TestSequentialBulkActionExecutionSilent(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0
	for i := range m.states {
		m.states[i].loading = false
	}

	m.states[0].packages = []string{"pkgA", "pkgB"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB"}
	m.states[0].cursor = 0

	// Select pkgA and pkgB
	m.states[0].selected["pkgA"] = true
	m.states[0].selected["pkgB"] = true

	// Simulate pressing "x" key to uninstall bulk
	mModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = mModel.(Model)

	// Confirm with "enter" / "s" (logs disabled)
	mModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mModel.(Model)

	if m.actionOverlay {
		t.Fatal("expected actionOverlay to be false after confirmation")
	}
	if m.logOverlay {
		t.Fatal("expected logOverlay to be false since we pressed 'enter'")
	}
	if m.bulkLogs {
		t.Fatal("expected bulkLogs to be false")
	}
	if cmd == nil {
		t.Fatal("expected a non-nil batch command to start run")
	}
}

func TestBulkActionWithErrors(t *testing.T) {
	m := New()
	m.allMode = false
	m.activeTab = 0
	for i := range m.states {
		m.states[i].loading = false
	}

	m.states[0].packages = []string{"pkgA", "pkgB"}
	m.states[0].displayPackages = []string{"pkgA", "pkgB"}
	m.states[0].cursor = 0

	// Select pkgA and pkgB
	m.states[0].selected["pkgA"] = true
	m.states[0].selected["pkgB"] = true

	// Simulate pressing "x" key to uninstall bulk
	mModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = mModel.(Model)

	// Confirm with "enter" / "s"
	mModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mModel.(Model)
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	// Simulate completion of first package (pkgA) without error
	actionMsg1 := pm.ActionMsg{
		Manager:     "brew",
		PackageName: "pkgA",
		Action:      pm.Remove,
		Err:         nil,
	}
	mModel, cmd = m.Update(actionMsg1)
	m = mModel.(Model)

	if len(m.bulkErrors) != 0 {
		t.Fatalf("expected 0 errors in bulkErrors, got %v", m.bulkErrors)
	}

	// Simulate completion of second package (pkgB) with error
	errPkgB := fmt.Errorf("permission denied")
	actionMsg2 := pm.ActionMsg{
		Manager:     "brew",
		PackageName: "pkgB",
		Action:      pm.Remove,
		Err:         errPkgB,
	}
	mModel, cmd = m.Update(actionMsg2)
	m = mModel.(Model)

	// Bulk queue is completed: bulkQueue should be nil, selections cleared
	if m.bulkQueue != nil {
		t.Fatalf("expected bulkQueue to be nil after completion, got %v", m.bulkQueue)
	}
	if len(m.states[0].selected) != 0 {
		t.Fatalf("expected selections to be cleared, got %v", m.states[0].selected)
	}
	if !strings.Contains(m.actionStatus, "finished with 1 errors") {
		t.Fatalf("expected action status to report 1 error, got '%s'", m.actionStatus)
	}

	if len(m.bulkErrors) != 1 || m.bulkErrors[0] != errPkgB.Error() {
		t.Fatalf("expected 1 error in bulkErrors, got %v", m.bulkErrors)
	}

	// Check if error was appended to logLines
	hasErrorInLogs := false
	for _, log := range m.logLines {
		if strings.Contains(log, "Error: permission denied") {
			hasErrorInLogs = true
			break
		}
	}
	if !hasErrorInLogs {
		t.Fatal("expected logLines to contain the error description")
	}
}

func TestSearchUI(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 24
	m.searchTabActive = true

	// Initially empty search results
	m.searchResults = nil
	m.searchLoading = false
	leftView := m.renderSearchLeftPanel(60, 20)
	if !strings.Contains(leftView, "Type a package name and press Enter to search") {
		t.Fatal("expected prompt when no search results are present")
	}

	// Simulated loading
	m.searchLoading = true
	leftView = m.renderSearchLeftPanel(60, 20)
	if !strings.Contains(leftView, "Searching registries") {
		t.Fatal("expected loading spinner prompt when search is loading")
	}

	// Simulated results
	m.searchLoading = false
	m.searchResults = []pm.SearchResult{
		{Name: "wget", Manager: "brew", Description: "Internet retriever", Version: "1.21.4"},
		{Name: "express", Manager: "npm", Description: "Fast framework", Version: "4.18.2"},
	}
	m.searchResultCursor = 0

	leftView = m.renderSearchLeftPanel(60, 20)
	if !strings.Contains(strings.ToLower(leftView), "wget") || !strings.Contains(strings.ToLower(leftView), "express") {
		t.Fatal("expected search results to be rendered in left panel")
	}

	rightView := m.renderSearchRightPanel(50, 20)
	if !strings.Contains(rightView, "wget") || !strings.Contains(rightView, "Internet retriever") {
		t.Fatal("expected wget details to be rendered in right panel")
	}

	// Cursor navigation
	// Down key
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updatedModel.(Model)
	if cmd != nil {
		t.Fatal("expected nil command on down navigation")
	}
	if m.searchResultCursor != 1 {
		t.Fatalf("expected cursor to be 1, got %d", m.searchResultCursor)
	}

	rightView = m.renderSearchRightPanel(50, 20)
	if !strings.Contains(rightView, "express") || !strings.Contains(rightView, "Fast framework") {
		t.Fatal("expected express details to be rendered in right panel after navigation")
	}

	// Up key
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updatedModel.(Model)
	if m.searchResultCursor != 0 {
		t.Fatalf("expected cursor to return to 0, got %d", m.searchResultCursor)
	}

	// List view fallback
	m.width = 40 // small width to trigger list view fallback
	fallbackView := m.listViewFallback()
	if !strings.Contains(fallbackView, "[brew] wget") {
		t.Fatal("expected fallback view to render search results correctly")
	}
}

func TestSearchInstallFlow(t *testing.T) {
	// Initialize Model
	m := New()

	m.searchTabActive = true
	m.searchActive = false
	m.searchResults = []pm.SearchResult{
		{
			Name:        "git",
			Manager:     "winget",
			Description: "Fast distributed version control system",
			Version:     "2.40.0",
		},
	}
	m.searchResultCursor = 0

	// Pressing "i" should trigger actionOverlay
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updatedModel.(Model)
	if cmd != nil {
		t.Fatal("expected nil command on action overlay trigger")
	}
	if !m.actionOverlay {
		t.Fatal("expected action overlay to be active")
	}
	if m.pendingAction != pm.Install {
		t.Fatalf("expected pending action to be Install, got %v", m.pendingAction)
	}
	if m.pendingPackage != "git" {
		t.Fatalf("expected pending package to be git, got %s", m.pendingPackage)
	}

	// Pressing "y" on actionOverlay should start the installation action and open logs overlay
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updatedModel.(Model)
	if cmd == nil {
		t.Fatal("expected non-nil command for installation run")
	}
	if m.actionOverlay {
		t.Fatal("expected action overlay to be closed")
	}
	if !m.logOverlay {
		t.Fatal("expected log overlay to be active")
	}
	if !m.logActive {
		t.Fatal("expected logs to be active")
	}

	// Simulate ActionMsg completion of pm.Install
	msg := pm.ActionMsg{
		Action:      pm.Install,
		PackageName: "git",
		Manager:     "winget",
		Err:         nil,
	}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	
	// Assert searchTabActive is closed and activeTab is switched to winget (index 3)
	if m.searchTabActive {
		t.Fatal("expected searchTabActive to be false after successful install")
	}
	if m.activeTab != 3 {
		t.Fatalf("expected activeTab to be 3 (winget), got %d", m.activeTab)
	}
}

func TestSearchBackspaceHandling(t *testing.T) {
	m := New()
	m.searchTabActive = true
	m.searchActive = false
	m.searchQuery = "golang"

	// 1. Press backspace when unfocused (should focus search and delete last character)
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updatedModel.(Model)
	if !m.searchActive {
		t.Fatal("expected searchActive to be true after backspace on active search query")
	}
	if m.searchQuery != "golan" {
		t.Fatalf("expected searchQuery to be 'golan', got %s", m.searchQuery)
	}

	// 2. Press ctrl+h when focused (should delete last character)
	m.searchActive = true
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updatedModel.(Model)
	if m.searchQuery != "gola" {
		t.Fatalf("expected searchQuery to be 'gola' after ctrl+h, got %s", m.searchQuery)
	}
}

func TestWingetDetailErrorCaching(t *testing.T) {
	m := New()
	
	// Send WingetDetailMsg with error
	errPkg := "MSIX\\System.Package"
	msg := pm.WingetDetailMsg{
		PackageID: errPkg,
		Err:       fmt.Errorf("no package found"),
	}
	
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)
	
	st := m.states[3] // winget tab
	if st.WingetDetails == nil {
		t.Fatal("expected WingetDetails map to be initialized")
	}
	
	detail, exists := st.WingetDetails[errPkg]
	if !exists {
		t.Fatal("expected error package details to be cached")
	}
	if !strings.Contains(detail.Description, "Details unavailable") {
		t.Fatalf("expected placeholder error description, got: %q", detail.Description)
	}
}

func TestModelBuildDebug(t *testing.T) {
	content, err := os.ReadFile("C:/Users/amanj/.gemini/antigravity/brain/bf324dac-9e54-4317-b1bf-c6d283255bae/.system_generated/tasks/task-2218.log")
	if err != nil {
		t.Logf("Skipping TestModelBuildDebug as log file is not readable: %v", err)
		return
	}

	names, versions := parseWingetListForTest(string(content))
	t.Logf("Parsed %d names from winget log", len(names))

	m := New()
	m.activeTab = 3 // winget tab
	
	msg3 := pm.PackageListMsg{
		TabIndex: 3,
		Packages: names,
		Versions: versions,
	}
	model, _ := m.Update(msg3)
	m = model.(Model)

	msg2 := pm.PackageListMsg{
		TabIndex: 2,
		Packages: []string{"aiohttp", "requests", "urllib3"},
		Versions: map[string]string{"aiohttp": "3.8.1", "requests": "2.28.1", "urllib3": "1.26.12"},
	}
	model, _ = m.Update(msg2)
	m = model.(Model)

	msg1 := pm.PackageListMsg{
		TabIndex: 1,
		Packages: []string{"npm", "typescript"},
		Versions: map[string]string{"npm": "8.19.2", "typescript": "4.8.4"},
	}
	model, _ = m.Update(msg1)
	m = model.(Model)

	m.activeTab = 0 // ALL tab
	m.allMode = true
	m = m.applyFilter()
}

func parseWingetListForTest(output string) ([]string, map[string]string) {
	var names []string
	versions := make(map[string]string)
	lines := strings.Split(output, "\n")
	
	sepIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 && strings.Count(trimmed, "-") > len(trimmed)/2 {
			sepIdx = i
			break
		}
	}
	if sepIdx < 1 {
		return nil, nil
	}
	
	header := lines[sepIdx-1]
	idIdx := strings.Index(header, "Id")
	if idIdx < 0 {
		idIdx = strings.Index(header, "ID")
	}
	versionIdx := strings.Index(header, "Version")
	if idIdx < 0 || versionIdx < 0 {
		return nil, nil
	}
	
	for i := sepIdx + 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		if len(line) <= idIdx {
			continue
		}
		
		idEnd := versionIdx
		if idEnd > len(line) {
			idEnd = len(line)
		}
		id := strings.TrimSpace(line[idIdx:idEnd])
		if id == "" {
			continue
		}
		
		version := ""
		if len(line) > versionIdx {
			availableIdx := strings.Index(header, "Available")
			if availableIdx > versionIdx && availableIdx < len(line) {
				version = strings.TrimSpace(line[versionIdx:availableIdx])
			} else {
				fields := strings.Fields(line[versionIdx:])
				if len(fields) > 0 {
					version = fields[0]
				}
			}
		}
		
		if _, exists := versions[id]; !exists {
			names = append(names, id)
		}
		versions[id] = version
	}
	return names, versions
}

func TestRenderMSIX(t *testing.T) {
	m := New()
	m.activeTab = 3
	m.allMode = true
	
	// Create some packages including MSIX
	pkgs := []string{
		"ARP\\Machine\\X64\\O365HomePremRetail - en-gb",
		"Google.Antigravity",
		"MSIX\\5319275A.WhatsAppDesktop_2.2625.101.0_x64__cv1g1gvanyjgm",
		"MSIX\\Microsoft.AV1VideoExtension_2.0.7.0_x64__8wekyb3d8bbwe",
	}
	
	msg := pm.PackageListMsg{
		TabIndex: 3,
		Packages: pkgs,
		Versions: map[string]string{
			pkgs[0]: "1.0",
			pkgs[1]: "2.3",
			pkgs[2]: "2.26",
			pkgs[3]: "2.0",
		},
	}
	
	model, _ := m.Update(msg)
	m = model.(Model)
	m.activeTab = 0
	m.allMode = true
	m = m.applyFilter()
	
	t.Logf("allDisplayPackages: %v", m.allDisplayPackages)
	for i, pkg := range m.allDisplayPackages {
		origin := m.allPackageOrigin[pkg]
		maxPkgLen := 50 - 4 - 6
		displayPkg := truncateString(pkg, maxPkgLen)
		pkgRendered := ItemStyle.Render(displayPkg)
		t.Logf("Item %d: pkg=%q, origin=%q, maxPkgLen=%d, displayPkg=%q (len=%d), pkgRendered=%q (len=%d)",
			i, pkg, origin, maxPkgLen, displayPkg, len(displayPkg), pkgRendered, len(pkgRendered))
	}
	
	panel := m.renderAllLeftPanel(50, 10)
	t.Logf("Rendered panel:\n%s", panel)
}





