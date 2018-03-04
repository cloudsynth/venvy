package modules

import (
	"github.com/pnegahdar/venvy/venvy"
	"github.com/pnegahdar/venvy/util"
)

type ExecConfig struct {
	ActivationCommands   []string `json:"activation_commands"`
	DeactivationCommands []string `json:"deactivation_commands"`
}

type ExecModule struct {
	manager *venvy.ProjectManager
	config  *ExecConfig
}

func (ps *ExecModule) ShellActivateCommands() ([]string, error) {
	return ps.config.ActivationCommands, nil

}

func (ps *ExecModule) ShellDeactivateCommands() ([]string, error) {
	return ps.config.DeactivationCommands, nil
}

func NewExecModule(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	moduleConfig := &ExecConfig{}
	err := util.UnmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	return &ExecModule{manager: manager, config: moduleConfig}, nil
}
