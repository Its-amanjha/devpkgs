package pm

import (
	"bufio"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type Manager interface {
	Name() string
	TabLabel() string
	ListInstalled() tea.Cmd
	RunAction(packageName string, action Action, programChan chan<- tea.Msg) tea.Cmd
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
	Manager     string
	Err         error
}

func Run(packageName string, action Action, manager string, name string, args ...string) tea.Cmd {
	return func() tea.Msg {
		_, err := exec.Command(name, args...).CombinedOutput()
		if err != nil {
			return ActionMsg{PackageName: packageName, Action: action, Manager: manager, Err: err}
		}
		return ActionMsg{PackageName: packageName, Action: action, Manager: manager}
	}
}

type LogLineMsg struct {
	Line string
}

type LogFinishMsg struct {
	Manager string
	Err     error
}

func RunStream(
	programChan chan<- tea.Msg,
	packageName string,
	action Action,
	manager string,
	cmdName string,
	args ...string,
) tea.Cmd {
	return func() tea.Msg {
		go func() {
			c := exec.Command(cmdName, args...)
			stdout, err := c.StdoutPipe()
			if err != nil {
				programChan <- LogFinishMsg{Manager: manager, Err: err}
				programChan <- ActionMsg{PackageName: packageName, Action: action, Manager: manager, Err: err}
				return
			}
			c.Stderr = c.Stdout
			if err := c.Start(); err != nil {
				programChan <- LogFinishMsg{Manager: manager, Err: err}
				programChan <- ActionMsg{PackageName: packageName, Action: action, Manager: manager, Err: err}
				return
			}

			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				programChan <- LogLineMsg{Line: scanner.Text()}
			}

			var finishErr error
			if scanErr := scanner.Err(); scanErr != nil {
				finishErr = scanErr
			}

			err = c.Wait()
			if finishErr == nil {
				finishErr = err
			}

			programChan <- LogFinishMsg{Manager: manager, Err: finishErr}
			programChan <- ActionMsg{PackageName: packageName, Action: action, Manager: manager, Err: err}
		}()
		return nil
	}
}
