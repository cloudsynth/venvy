package modules

import "github.com/pnegahdar/venvy/venvy"

type DebugModule struct{}

func (ps *DebugModule) ShellActivateCommands() ([]string, error) {
	return []string{"set -x"}, nil
}

func (ps *DebugModule) ShellDeactivateCommands() ([]string, error) {
	return []string{"set +x"}, nil
}

func NewDebugModule(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	return &DebugModule{}, nil
}
