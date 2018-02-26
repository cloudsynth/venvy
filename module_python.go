package main

import (
	"encoding/json"
	"path"
)

const DefaultPython = "python"

type PyModuleConfig struct {
	Python string
}

type PythonModule struct {
	manager *ConfigManager
	config  *PyModuleConfig
}

func (pm *PythonModule) venvDir() string {
	return pm.manager.Path("pyvenv")
}

func (pm *PythonModule) activteShPath() string {
	return path.Join(pm.venvDir(), "bin", "activate")
}

func (pm *PythonModule) venvExists() bool {
	return pathExists(pm.activteShPath())
}

// Moduler interface methods
func (pm *PythonModule) ShellActivateSh() (string, error) {
	tmplData := struct {
		VenvExists   bool
		VenvDir      string
		Python       string
		ActivatePath string
	}{
		VenvExists:   pm.venvExists(),
		VenvDir:      pm.venvDir(),
		Python:       pm.config.Python,
		ActivatePath: pm.activteShPath(),
	}
	script := `
{{- if not .VenvExists -}}
virtualenv -p {{ .Python }} {{ .VenvDir }}
{{- end }}
. {{ .ActivatePath -}} 
`
	return stringTemplate("pythonShellActivate", script, tmplData)
}

func (pm *PythonModule) ShellDeactivateSh() (string, error) {
	return "deactivate", nil
}

func NewPythonModule(manager *ConfigManager, self *Module) (Moduler, error) {
	moduleConfig := &PyModuleConfig{}
	err := json.Unmarshal(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	if moduleConfig.Python == "" {
		moduleConfig.Python = DefaultPython
	}
	return &PythonModule{manager: manager, config: moduleConfig}, nil
}
