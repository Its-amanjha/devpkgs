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
