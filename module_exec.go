package main

type ExecConfig struct {
	ActivationCommands   []string `json:"activation_commands"`
	DeactivationCommands []string `json:"deactivation_commands"`
}

type ExecModule struct {
	manager *ProjectManager
	config  *ExecConfig
}

func (ps *ExecModule) ShellActivateCommands() ([]string, error) {
	return ps.config.ActivationCommands, nil

}

func (ps *ExecModule) ShellDeactivateCommands() ([]string, error) {
	return ps.config.DeactivationCommands, nil
}

func NewExecModule(manager *ProjectManager, self *Module) (Moduler, error) {
	moduleConfig := &ExecConfig{}
	err := unmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	return &ExecModule{manager: manager, config: moduleConfig}, nil
}
