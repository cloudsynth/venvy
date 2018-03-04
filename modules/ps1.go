package modules

import (
	"fmt"
	"github.com/fatih/color"
	"strings"
	"github.com/pnegahdar/venvy/venvy"
	"github.com/pnegahdar/venvy/util"
)

type PS1Config struct {
	Value string
}

type PS1Module struct {
	manager *venvy.ProjectManager
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
		`if [ ! -z "$OLD_PS1" ]; then export PS1="$OLD_PS1"; unset OLD_PS1; fi`,
	}, nil
}

func NewPS1Module(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	moduleConfig := &PS1Config{}
	err := util.UnmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	if moduleConfig.Value == "" {
		parts := []string{
			color.HiCyanString("["),
			color.HiBlueString(venvy.ProjectName),
			color.HiGreenString(":"),
			color.HiMagentaString(manager.Project.Name),
			color.HiCyanString("]"),
		}
		moduleConfig.Value = strings.Join(parts, "")
	}
	return &PS1Module{manager: manager, config: moduleConfig}, nil
}
