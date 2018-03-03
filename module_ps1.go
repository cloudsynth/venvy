package main

import (
	"fmt"
	"github.com/fatih/color"
	"strings"
)

type PS1Config struct {
	Value string
}

type PS1Module struct {
	manager *ProjectManager
	config  *PS1Config
}

func (ps *PS1Module) ShellActivateCommands() ([]string, error) {
	return []string{
		`export OLD_PS1="$PS1"`,
		fmt.Sprintf(`export PS1="%s $PS1"`, ps.config.Value),
	}, nil
}

func (ps *PS1Module) ShellDeactivateCommands() ([]string, error) {
	return []string{
		`export PS1="$OLD_PS1"`,
		"unset OLD_PS1",
	}, nil
}

func NewPS1Module(manager *ProjectManager, self *Module) (Moduler, error) {
	moduleConfig := &PS1Config{}
	err := unmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	if moduleConfig.Value == "" {
		parts := []string{
			color.HiCyanString("["),
			color.HiBlueString(ProjectName),
			color.HiGreenString(":"),
			color.HiMagentaString(manager.activeProject.Name),
			color.HiCyanString("]"),
		}
		moduleConfig.Value = strings.Join(parts, "")
	}
	return &PS1Module{manager: manager, config: moduleConfig}, nil
}
