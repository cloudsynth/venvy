package modules

import (
	"fmt"
	"github.com/pnegahdar/venvy/util"
	"github.com/pnegahdar/venvy/venvy"
	"os"
)

type JumpConfig struct {
	ToDir           string
	DisableJumpBack bool
}

type JumpModule struct {
	manager *venvy.ProjectManager
	lastDir string
	config  *JumpConfig
}

func (jm *JumpModule) ShellActivateCommands() ([]string, error) {
	return []string{fmt.Sprintf("cd %s", jm.config.ToDir)}, nil
}

func (jm *JumpModule) ShellDeactivateCommands() ([]string, error) {
	if jm.lastDir != "" && !jm.config.DisableJumpBack {
		return []string{fmt.Sprintf("cd %s", jm.lastDir)}, nil
	}
	return nil, nil
}

func NewJumpModule(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	moduleConfig := &JumpConfig{}
	lastDir, _ := os.Getwd()
	err := util.UnmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	if moduleConfig.ToDir == "" {
		moduleConfig.ToDir = manager.RootDir()
	}
	return &JumpModule{manager: manager, config: moduleConfig, lastDir: lastDir}, nil
}
