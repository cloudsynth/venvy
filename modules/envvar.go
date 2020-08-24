package modules

import (
	"bytes"
	"fmt"
	"github.com/pnegahdar/venvy/manager"
	"github.com/pnegahdar/venvy/util"
	"github.com/subosito/gotenv"
	"io/ioutil"
	"os"
)

type EnvVarConfig struct {
	Vars      map[string]string `json:"vars"`
	Files     []string          `json:"files"`
	UnsetVars []string          `json:"unset_vars"`
}

type EnvvarModule struct {
	config  *EnvVarConfig
	manager *venvy.ProjectManager
}

func (ev *EnvvarModule) ShellActivateCommands() ([]string, error) {
	commands := []string{}

	for _, file := range ev.config.Files {
		fullPath := ev.manager.ResolveRootPath(file)
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("unable to find file %s at %s", file, fullPath)
		}
		pairs := gotenv.Parse(bytes.NewReader(data))
		for key, value := range pairs {
			commands = append(commands, fmt.Sprintf(`export %s="%s"`, key, value))
		}
	}

	for key, value := range ev.config.Vars {
		commands = append(commands, fmt.Sprintf(`export %s="%s"`, key, value))
	}
	for _, varToUnset := range ev.config.UnsetVars {
		commands = append(commands, fmt.Sprintf(`unset %s`, varToUnset))
	}
	return commands, nil
}

func (ev *EnvvarModule) ShellDeactivateCommands() ([]string, error) {
	commands := []string{}
	// Unset vars or return to former (current) value
	for key, _ := range ev.config.Vars {
		currentValue, exists := os.LookupEnv(key)
		if exists {
			// Revert to the old value
			commands = append(commands, fmt.Sprintf(`export %s="%s"`, key, currentValue))
		} else {
			commands = append(commands, fmt.Sprintf("unset %s", key))
		}
	}
	// Put the unset vars back on deactivation
	for _, varToUnset := range ev.config.UnsetVars {
		currentValue, exists := os.LookupEnv(varToUnset)
		if exists {
			commands = append(commands, fmt.Sprintf(`export %s="%s"`, varToUnset, currentValue))
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
	return &EnvvarModule{config: moduleConfig, manager: manager}, nil
}
