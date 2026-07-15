package pm

import (
	"encoding/json"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type WingetManager struct{ tabIndex int }

func NewWingetManager(tabIndex int) *WingetManager { return &WingetManager{tabIndex: tabIndex} }
func (w *WingetManager) Name() string              { return "winget" }
func (w *WingetManager) TabLabel() string          { return "WinGet" }

func (w *WingetManager) ListInstalled() tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("winget"); err != nil {
			return PackageListMsg{TabIndex: w.tabIndex}
		}
		file, err := os.CreateTemp("", "devpkgs-winget-*.json")
		if err != nil { return PackageListMsg{Err: err, TabIndex: w.tabIndex} }
		path := file.Name()
		file.Close()
		os.Remove(path)
		defer os.Remove(path)
		if err := exec.Command("winget", "export", "--output", path, "--include-versions", "--accept-source-agreements", "--disable-interactivity").Run(); err != nil {
			return PackageListMsg{Err: err, TabIndex: w.tabIndex}
		}
		data, err := os.ReadFile(path)
		if err != nil { return PackageListMsg{Err: err, TabIndex: w.tabIndex} }
		names, versions, err := parseWingetExport(data)
		return PackageListMsg{Packages: names, Versions: versions, Err: err, TabIndex: w.tabIndex}
	}
}

func (w *WingetManager) RunAction(packageName string, action Action) tea.Cmd {
	args := []string{"upgrade", "--id", packageName, "--exact", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	if action == Remove {
		args = []string{"uninstall", "--id", packageName, "--exact", "--disable-interactivity"}
	}
	return Run(packageName, action, "winget", args...)
}

func parseWingetExport(data []byte) ([]string, map[string]string, error) {
	var export struct {
		Sources []struct {
			Packages []struct {
				ID      string `json:"PackageIdentifier"`
				Version string `json:"Version"`
			} `json:"Packages"`
		} `json:"Sources"`
	}
	if err := json.Unmarshal(data, &export); err != nil { return nil, nil, err }
	versions := map[string]string{}
	var names []string
	for _, source := range export.Sources {
		for _, pkg := range source.Packages {
			if pkg.ID == "" { continue }
			if _, exists := versions[pkg.ID]; !exists { names = append(names, pkg.ID) }
			versions[pkg.ID] = pkg.Version
		}
	}
	return names, versions, nil
}
