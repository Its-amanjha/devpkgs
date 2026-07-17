package pm

import (
	"os/exec"
	"strings"

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
		out, err := exec.Command("winget", "list", "--accept-source-agreements", "--disable-interactivity").Output()
		if err != nil {
			return PackageListMsg{Err: err, TabIndex: w.tabIndex}
		}
		names, versions := parseWingetList(string(out))
		return PackageListMsg{Packages: names, Versions: versions, TabIndex: w.tabIndex}
	}
}

func (w *WingetManager) RunAction(packageName string, action Action, programChan chan<- tea.Msg) tea.Cmd {
	var args []string
	if action == Remove {
		args = []string{"uninstall", "--id", packageName, "--exact", "--disable-interactivity"}
	} else if action == Install {
		args = []string{"install", "--id", packageName, "--exact", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	} else {
		args = []string{"upgrade", "--id", packageName, "--exact", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	}
	return RunStream(programChan, packageName, action, "winget", "winget", args...)
}

func parseWingetList(output string) ([]string, map[string]string) {
	var names []string
	versions := make(map[string]string)
	lines := strings.Split(output, "\n")
	
	// Find the separator line (all dashes)
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
	
	// Parse header to find column positions
	header := lines[sepIdx-1]
	idIdx := strings.Index(header, "Id")
	if idIdx < 0 {
		idIdx = strings.Index(header, "ID")
	}
	versionIdx := strings.Index(header, "Version")
	if idIdx < 0 || versionIdx < 0 {
		return nil, nil
	}
	
	// Parse data rows
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

type WingetDetailData struct {
	ID          string
	Version     string
	Publisher   string
	Homepage    string
	License     string
	Description string
}

type WingetDetailMsg struct {
	PackageID string
	Data      *WingetDetailData
	Err       error
}

func ParseWingetShow(id string, output string) *WingetDetailData {
	data := &WingetDetailData{ID: id}
	lines := strings.Split(output, "\n")
	inDesc := false
	var descLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(line, "Version:") {
			inDesc = false
			data.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Publisher:") {
			inDesc = false
			data.Publisher = strings.TrimSpace(strings.TrimPrefix(line, "Publisher:"))
		} else if strings.HasPrefix(line, "Homepage:") {
			inDesc = false
			data.Homepage = strings.TrimSpace(strings.TrimPrefix(line, "Homepage:"))
		} else if strings.HasPrefix(line, "License:") {
			inDesc = false
			data.License = strings.TrimSpace(strings.TrimPrefix(line, "License:"))
		} else if strings.HasPrefix(line, "Description:") {
			inDesc = true
		} else if inDesc {
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				descLines = append(descLines, trimmed)
			} else {
				inDesc = false
			}
		}
	}
	data.Description = strings.Join(descLines, " ")
	return data
}

func FetchWingetDetails(id string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("winget", "show", "--id", id, "--accept-source-agreements", "--disable-interactivity").Output()
		if err != nil {
			return WingetDetailMsg{PackageID: id, Err: err}
		}
		return WingetDetailMsg{PackageID: id, Data: ParseWingetShow(id, string(out))}
	}
}

