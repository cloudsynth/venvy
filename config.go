package main

import (
	"fmt"
	"encoding/json"
	"strings"
	"github.com/mitchellh/go-homedir"
	"os"
	"html/template"
	"path"
	"bytes"
)

type Module struct {
	Name   string
	Type   string
	Config json.RawMessage
}

type moduleMaker func(configManager *ConfigManager, self *Module) (Moduler, error)

var ModuleMakers = map[string]moduleMaker{
	"python": NewPythonModule,
}

type Moduler interface {
	ShellActivateSh() (string, error)
	ShellDeactivateSh() (string, error)
}

type Project struct {
	Name              string // az- validator
	Root              string
	Generation        int
	SubprocessModules []string
	ShellModules      []string
	ScriptModules     []string
	ScriptSearchGlobs []string
}

type Config struct {
	Projects []*Project
	Modules  []*Module
}

type ConfigManager struct {
	Config
	configPath    string
	activeProject *Project
	storageRoot   string
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (cm *ConfigManager) Path(elem ...string) string {
	allElems := append([]string{cm.storageRoot, cm.activeProject.Name}, elem...)
	targetDir := path.Join(allElems...)
	expanded, err := homedir.Expand(targetDir)
	if err != nil {
		panic(err)
	}
	return expanded
}

func stringTemplate(tmplName, tmpl string, data interface{}) (string, error) {
	parsedTemplate, err := template.New(tmplName).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = parsedTemplate.Execute(&out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func (cm *ConfigManager) Modulers(moduleNames []string) (map[*Module]Moduler, error) {
	modules := map[*Module]Moduler{}
	for _, moduleName := range moduleNames {
		found := false
		for _, module := range cm.Modules {
			if module.Name == moduleName {
				found = true
				moduleMaker, ok := ModuleMakers[module.Type]
				if !ok {
					return nil, fmt.Errorf("module %s for project %s has unkown type %s", moduleName, cm.activeProject.Name, module.Type)
				}
				preparedModule, err := moduleMaker(cm, module)
				if err != nil {
					return nil, fmt.Errorf("module %s for project %s could not initializaed, had err %s", module.Name, cm.activeProject.Name, err)
				}
				modules[module] = preparedModule
			}
		}
		if !found {
			return nil, fmt.Errorf("module %s not found for project %s", moduleName, cm.activeProject.Name)
		}
	}
	return modules, nil
}

func (cm *ConfigManager) ShellActivateSh() (string, error) {
	modules, err := cm.Modulers(cm.activeProject.ScriptModules)
	if err != nil {
		return "", err
	}
	activationScripts := []string{}
	// Go forwards for activate backwards for deactivate
	for module, moduler := range modules {
		script, err := moduler.ShellActivateSh()
		if err != nil {
			return "", fmt.Errorf("module %s for project %s could not generate activation script, had err %s", module.Name, cm.activeProject.Name, err)
		}
		activationScripts = append(activationScripts, script)
	}
	return strings.Join(activationScripts, "\n"), nil
}

func NewConfigManager(config Config, configPath string, activeProject string, storageRoot string) (*ConfigManager, error) {
	storageRoot, err := homedir.Expand(storageRoot)
	if err != nil {
		return nil, err
	}
	manager := &ConfigManager{Config: config, configPath: configPath, storageRoot: storageRoot}
	for _, proj := range config.Projects {
		if proj.Name == activeProject {
			manager.activeProject = proj
		}
	}
	if manager.activeProject == nil {
		return nil, fmt.Errorf("project %s not found in config", activeProject)
	}
	err = os.MkdirAll(storageRoot, 0600)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

func FindConfigs() ([]*ConfigManager, error) {
	pyModule := &Module{
		Name:   "py2",
		Config: []byte("{}"),
		Type:   "python",
	}

	chiefProject := &Project{
		Name:              "chief",
		Root:              ".",
		SubprocessModules: []string{"py2"},
		ShellModules:      []string{"py2"},
		ScriptModules:     []string{"py2"},
		ScriptSearchGlobs: []string{},
	}

	config := Config{
		Modules:  []*Module{pyModule},
		Projects: []*Project{chiefProject},
	}

	manager, err := NewConfigManager(config, "~/Projects/go/src/github.com/pnegahdr/venv/inenv.toml", "chief", "~/.venvy/")
	if err != nil {
		panic(err)
	}
	return []*ConfigManager{manager}, nil
}
