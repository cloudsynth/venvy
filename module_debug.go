package main

type DebugModule struct{}

func (ps *DebugModule) ShellActivateCommands() ([]string, error) {
	return []string{"set -x"}, nil
}

func (ps *DebugModule) ShellDeactivateCommands() ([]string, error) {
	return []string{"set +x"}, nil
}

func NewDebugModule(manager *ProjectManager, self *Module) (Moduler, error) {
	return &DebugModule{}, nil
}
