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

func (p *PipManager) RunAction(name string, action Action, programChan chan<- tea.Msg) tea.Cmd {
	cmd, prefix := p.resolve()
	if cmd == "" {
		return func() tea.Msg {
			go func() {
				err := fmt.Errorf("pip not found")
				programChan <- LogFinishMsg{Manager: "pip", Err: err}
				programChan <- ActionMsg{PackageName: name, Action: action, Manager: "pip", Err: err}
			}()
			return nil
		}
	}
	var args []string
	if action == Remove {
		args = append(prefix, "uninstall", "-y", name)
	} else if action == Install {
		args = append(prefix, "install", name)
	} else {
		args = append(prefix, "install", "--upgrade", name)
	}
	return RunStream(programChan, name, action, "pip", cmd, args...)
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
