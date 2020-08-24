package modules

import (
	"github.com/pnegahdar/venvy/manager"
	"github.com/pnegahdar/venvy/util"
	logger "github.com/sirupsen/logrus"
	"os/exec"

	"fmt"
	"strings"
)

type Pane struct {
	Root     string
	Commands []string
}

var pf = fmt.Sprintf

type TmuxWindowConfig struct {
	DisableDestroyExisting bool   `json:"disable_destroy_existing"`
	Name                   string `validate:"required"`
	Panes                  []Pane
	Layout                 string // even-horizontal, even-vertical, main-horizontal, main-vertical, tiled
}

// TODO: tmux when zsh plugin is not enabled
type TmuxWindow struct {
	manager *venvy.ProjectManager
	config  *TmuxWindowConfig
}

func (tx *TmuxWindow) ShellActivateCommands() ([]string, error) {
	currentTmuxSession, err := tmuxCurrentSession()
	if err != nil {
		return nil, err
	}
	if currentTmuxSession == "" {
		return nil, fmt.Errorf("no last tmux session and use_session not passed in config")
	}
	targetWindowName := fmt.Sprintf("%s-%s", tx.manager.Project.Name, tx.config.Name)
	currentWindow, _ := tmuxCurrentWindow()
	// Deactivating current window wont work as the rest of the venvy execution will get aborted.
	if targetWindowName == currentWindow {
		logger.Warnf("window target %s is currently active. not going to reactivate tmux -- switch first to restart the window", currentWindow)
		return nil, nil
	}
	existingWindows, err := tmuxListWindows()
	if err != nil {
		return nil, fmt.Errorf("could not list current tmux windows with err %s", err)
	}
	commands := []string{}
	for _, window := range existingWindows {
		if targetWindowName == window.Name {
			if tx.config.DisableDestroyExisting {
				return nil, fmt.Errorf("window with name %s already exists and disable_destory_existing set to true", targetWindowName)
			} else {
				// Kill the existing window
				commands = append(commands, pf("tmux kill-window -t %s", window.ID))
			}
		}
	}
	// Note tmux allows us to send all these commands as a single chained command. This less prone to syntax errors.
	for i, pane := range tx.config.Panes {
		paneRootDir := tx.manager.RootPath(pane.Root)
		if i == 0 {
			commands = append(commands,
				pf("lastWindow=$(tmux new-window -P -c %s -n %s)", paneRootDir, targetWindowName),
			)
		} else {
			commands = append(commands,
				pf(`lastWindow=$(tmux split-window -P -c %s -t "${lastWindow}")`, paneRootDir),
			)
		}
		if len(pane.Commands) > 0 {
			paneCommand := strings.Join(pane.Commands, ";")
			commands = append(commands,
				// C-m sends enter
				pf(`tmux send-keys -t "${lastWindow}" '%s' C-m`, paneCommand),
			)
		}

		// Set layout every time to make sure the panes fit as desired (tmux err: pane too small)
		commands = append(commands,
			pf(`tmux select-layout -t "${lastWindow}" %s`, tx.config.Layout),
		)
	}

	return commands, nil
}

func (tx *TmuxWindow) ShellDeactivateCommands() ([]string, error) {
	return nil, nil
}

func cmdOutput(cmd *exec.Cmd) (string, error) {
	result, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(result)), nil
}

func tmuxCurrentSession() (string, error) {
	return cmdOutput(exec.Command("tmux", "display-message", "-p", "#S"))
}

type NameID struct {
	ID   string
	Name string
}

func tmuxListWindows() (windows []NameID, err error) {
	output, err := cmdOutput(exec.Command("tmux", "list-windows", "-F", "#I|#W"))
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(output, "\n") {
		data := strings.Split(line, "|")
		windows = append(windows, NameID{ID: data[0], Name: data[1]})
	}
	return
}

func tmuxCurrentWindow() (string, error) {
	return cmdOutput(exec.Command("tmux", "display-message", "-p", "#W"))
}

func NewTmuxModule(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	moduleConfig := &TmuxWindowConfig{}
	err := util.UnmarshalAndValidate(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	if moduleConfig.Layout == "" {
		moduleConfig.Layout = "tiled"
	}
	if len(moduleConfig.Panes) == 0 {
		defaultPane := Pane{}
		moduleConfig.Panes = append(moduleConfig.Panes, defaultPane)
	}
	return &TmuxWindow{manager: manager, config: moduleConfig}, nil
}
