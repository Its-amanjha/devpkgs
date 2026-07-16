package pm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type SearchResult struct {
	Name        string
	Manager     string
	Description string
	Version     string
}

type RegistrySearchMsg struct {
	Manager string
	Results []SearchResult
	Err     error
}

func SearchBrew(query string, formulaeMap map[string]FormulaData) tea.Cmd {
	return func() tea.Msg {
		query = strings.ToLower(query)
		var results []SearchResult
		for name, data := range formulaeMap {
			if strings.Contains(strings.ToLower(name), query) ||
				strings.Contains(strings.ToLower(data.Desc), query) {
				results = append(results, SearchResult{
					Name:        name,
					Manager:     "brew",
					Description: data.Desc,
					Version:     data.Versions.Stable,
				})
			}
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})
		if len(results) > 25 {
			results = results[:25]
		}
		return RegistrySearchMsg{Manager: "brew", Results: results}
	}
}

func SearchWinget(query string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("winget", "search", "--query", query, "--count", "25", "--accept-source-agreements").CombinedOutput()
		if err != nil {
			return RegistrySearchMsg{Manager: "winget", Err: err}
		}
		// Parse the tabular output
		var results []SearchResult
		lines := strings.Split(string(out), "\n")
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
			return RegistrySearchMsg{Manager: "winget", Results: nil}
		}
		// Parse header to find column positions
		header := lines[sepIdx-1]
		nameIdx := 0
		idIdx := strings.Index(header, "Id")
		if idIdx < 0 {
			idIdx = strings.Index(header, "ID")
		}
		versionIdx := strings.Index(header, "Version")
		if idIdx < 0 || versionIdx < 0 {
			return RegistrySearchMsg{Manager: "winget", Results: nil}
		}
		// Parse data rows
		for i := sepIdx + 1; i < len(lines); i++ {
			line := lines[i]
			if len(line) < versionIdx {
				continue
			}
			id := strings.TrimSpace(line[idIdx:min(versionIdx, len(line))])
			version := ""
			if versionIdx < len(line) {
				version = strings.TrimSpace(line[versionIdx:])
			}
			name := strings.TrimSpace(line[nameIdx:min(idIdx, len(line))])
			if id == "" {
				continue
			}
			results = append(results, SearchResult{
				Name:        id,
				Manager:     "winget",
				Description: name,
				Version:     version,
			})
		}
		return RegistrySearchMsg{Manager: "winget", Results: results}
	}
}

func SearchNpm(query string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		url := fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=25", query)
		resp, err := client.Get(url)
		if err != nil {
			return RegistrySearchMsg{Manager: "npm", Err: err}
		}
		defer resp.Body.Close()

		var data struct {
			Objects []struct {
				Package struct {
					Name        string `json:"name"`
					Version     string `json:"version"`
					Description string `json:"description"`
				} `json:"package"`
			} `json:"objects"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return RegistrySearchMsg{Manager: "npm", Err: err}
		}
		var results []SearchResult
		for _, obj := range data.Objects {
			results = append(results, SearchResult{
				Name:        obj.Package.Name,
				Manager:     "npm",
				Description: obj.Package.Description,
				Version:     obj.Package.Version,
			})
		}
		return RegistrySearchMsg{Manager: "npm", Results: results}
	}
}

func SearchPip(query string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		// Exact match first
		url := fmt.Sprintf("https://pypi.org/pypi/%s/json", query)
		resp, err := client.Get(url)
		if err != nil {
			return RegistrySearchMsg{Manager: "pip", Results: nil}
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			var data struct {
				Info struct {
					Name    string `json:"name"`
					Version string `json:"version"`
					Summary string `json:"summary"`
				} `json:"info"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.Info.Name != "" {
				return RegistrySearchMsg{
					Manager: "pip",
					Results: []SearchResult{{
						Name:        data.Info.Name,
						Manager:     "pip",
						Description: data.Info.Summary,
						Version:     data.Info.Version,
					}},
				}
			}
		}
		return RegistrySearchMsg{Manager: "pip", Results: nil}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
