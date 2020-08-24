package venvy

import (
	"fmt"
	"github.com/pnegahdar/venvy/util"
	"path/filepath"
)

type NamedModuler struct {
	Name   string
	Module Moduler
}

type ProjectManager struct {
	*DataManager
	configManager  *ConfigManager
	Project        *Project
	relatedModules map[string]*Module
}

func (pm *ProjectManager) ConfigManager() *ConfigManager {
	return pm.configManager
}

func (pm *ProjectManager) Modulers() ([]*NamedModuler, error) {
	var modules []*NamedModuler
	for _, moduleName := range pm.Project.Modules {
		module, ok := pm.relatedModules[moduleName]
		if !ok {
			return nil, fmt.Errorf("module %s not found for project %s", moduleName, pm.Project.Name)
		}
		moduleMaker, ok := pm.ConfigManager().ModuleMakers[module.Type]
		if !ok {
			return nil, fmt.Errorf("module %s for project %s has unkown type %s", moduleName, pm.Project.Name, module.Type)
		}
		preparedModule, err := moduleMaker(pm, module)
		if err != nil {
			return nil, fmt.Errorf("module %s for project %s could not initializaed, had err %s", module.Name, pm.Project.Name, err)
		}
		modules = append(modules, &NamedModuler{Name: moduleName, Module: preparedModule})
	}
	return modules, nil
}

func (pm *ProjectManager) RootDir() string {
	configDir := filepath.Dir(pm.ConfigManager().configPath)
	projectRoot := pm.Project.Root
	if projectRoot == "" {
		projectRoot = filepath.Clean(configDir)
	} else {
		projectRoot = util.MustExpandPath(projectRoot)
		if !filepath.IsAbs(projectRoot) {
			projectRoot = filepath.Join(configDir, projectRoot)
		}
	}
	return projectRoot
}

func (pm *ProjectManager) RootPath(elem ...string) string {
	if len(elem) == 1 {
		expandedElem := util.MustExpandPath(elem[0])
		if filepath.IsAbs(expandedElem) {
			return expandedElem
		}
	}
	elem = append([]string{pm.RootDir()}, elem...)
	return filepath.Join(elem...)
}

func (pm *ProjectManager) ResolveRootPath(path string) string {
	fullPath := util.MustExpandPath(path)
	if !filepath.IsAbs(path) {
		fullPath = pm.RootPath(path)
	}
	return fullPath
}

func (pm *ProjectManager) AppendModules(modules ...*Module) {
	for _, module := range modules {
		pm.relatedModules[module.Name] = module
		pm.Project.Modules = append(pm.Project.Modules, module.Name)
	}
}

func (pm *ProjectManager) PrependModules(modules ...*Module) {
	// prepend in reverse so left most arg is up front
	for i := len(modules) - 1; i >= 0; i-- {
		pm.relatedModules[modules[i].Name] = modules[i]
		pm.Project.Modules = append([]string{modules[i].Name}, pm.Project.Modules...)

	}
}

func (pm *ProjectManager) ShellActivateCommands() ([]string, error) {
	modules, err := pm.Modulers()
	if err != nil {
		return nil, err
	}
	activationCommands := []string{}
	// Go forwards for activate backwards for deactivate
	for _, namedModular := range modules {
		lines, err := namedModular.Module.ShellActivateCommands()
		if err != nil {
			return nil, fmt.Errorf("module %s for project %s could not generate activation lines, had err %s", namedModular.Name, pm.Project.Name, err)
		}
		activationCommands = append(activationCommands, lines...)
	}
	return activationCommands, nil
}

func (pm *ProjectManager) ShellDeactivateCommands() ([]string, error) {
	modules, err := pm.Modulers()
	if err != nil {
		return nil, err
	}
	activationCommands := []string{}
	// Go forwards for activate backwards for deactivate
	for i := len(modules) - 1; i >= 0; i-- {
		namedModuler := modules[i]
		lines, err := namedModuler.Module.ShellDeactivateCommands()
		if err != nil {
			return nil, fmt.Errorf("module %s for project %s could not generate activation lines, had err %s", namedModuler.Name, pm.Project.Name, err)
		}
		activationCommands = append(activationCommands, lines...)
	}
	return activationCommands, nil
}
