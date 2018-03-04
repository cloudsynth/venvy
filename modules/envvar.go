package modules

import (
	"github.com/pnegahdar/venvy/venvy"
	"github.com/pnegahdar/venvy/util"
	"os"
	"fmt"
)

type EnvVarConfig struct {
	Vars map[string]string
}

type EnvvarModule struct {
	manager *venvy.ProjectManager
	lastDir string
	config  *EnvVarConfig
}

func (ev *EnvvarModule) ShellActivateCommands() ([]string, error) {
	commands := []string{}
	for key, value := range ev.config.Vars {
		commands = append(commands, fmt.Sprintf(`export %s="%s"`, key, value))
	}
	return commands, nil
}

func (ev *EnvvarModule) ShellDeactivateCommands() ([]string, error) {
	commands := []string{}
	for key, _ := range ev.config.Vars {
		currentValue, exists := os.LookupEnv(key)
		if exists {
			// Revert to the old value
			commands = append(commands, fmt.Sprintf(`export %s="%s"`, key, currentValue))
		} else {
			commands = append(commands, fmt.Sprintf("unset %s", key))
		}
	}
	return commands, nil
}

func NewEnvVarModule(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	moduleConfig := &EnvVarConfig{}
	err := util.UnmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	return &EnvvarModule{manager: manager, config: moduleConfig}, nil
}
