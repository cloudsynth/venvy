package venvy

import (
	"encoding/json"
)

const ProjectName = "venvy"
const Version = "0.0.0"

type Module struct {
	Name   string          `validate:"cleanName"`
	Type   string          `validate:"required"`
	Config json.RawMessage `validate:"-"`
}

type Moduler interface {
	ShellActivateCommands() ([]string, error)
	ShellDeactivateCommands() ([]string, error)
}

type Project struct {
	Name                  string `validate:"cleanName"`
	Root                  string
	Generation            int `validate:"min=0"`
	Modules               []string
	ScriptSubcommands     []string `json:"script_subcommands"`
	DisableBuiltinModules bool     `json:"disable_builtin_modules"`
}

type Config struct {
	Projects []*Project `validate:"dive"`
	Modules  []*Module  `validate:"dive"`
}

// Modules need to implement the following initialization interface
type ModuleMaker func(configManager *ProjectManager, self *Module) (Moduler, error)

type ModuleMakerTypeMap map[string]ModuleMaker
