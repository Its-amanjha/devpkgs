package pm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

func (b *BrewManager) RunAction(name string, action Action, programChan chan<- tea.Msg) tea.Cmd {
	if action == Remove {
		return RunStream(programChan, name, action, "brew", "brew", "uninstall", name)
	}
	if action == Install {
		return RunStream(programChan, name, action, "brew", "brew", "install", name)
	}
	return RunStream(programChan, name, action, "brew", "brew", "upgrade", name)
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
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				name, ver := fields[0], fields[1]
				names = append(names, name)
				versions[name] = ver
				if err == nil && prefix != "" {
					path := prefix + "/opt/" + name
					paths[name] = path
					sizes[name] = getDirSize(path)
				}
			}
		}
		return BrewListMsg{Names: names, Paths: paths, InstalledVersions: versions, Sizes: sizes}
	}
}

func (b *BrewManager) fetchFormulae() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("brew", "info", "--json=v2", "--installed").Output()
		if err != nil {
			return BrewFormulaeErrMsg(err)
		}

		var info struct {
			Formulae []FormulaData `json:"formulae"`
		}
		if err := json.Unmarshal(out, &info); err != nil {
			return BrewFormulaeErrMsg(err)
		}

		m := make(map[string]FormulaData)
		for _, f := range info.Formulae {
			m[f.Name] = f
		}
		return BrewFormulaeMsg(m)
	}
}

func getDirSize(path string) int64 {
	var size int64
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}

