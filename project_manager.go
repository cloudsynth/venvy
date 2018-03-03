package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
)

type ProjectManager struct {
	*Config
	configPath    string
	activeProject *Project
	loadedModules []*NamedModuler
	lmError       error
	storageRoot   string
	sync.Once
}

type NamedModuler struct {
	Name   string
	Module Moduler
}

func (cm *ProjectManager) StoragePath(elem ...string) string {
	allElems := append([]string{cm.storageRoot, cm.activeProject.Name}, elem...)
	targetDir := path.Join(allElems...)
	return path.Clean(mustExpandPath(targetDir))
}

func (cm *ProjectManager) RootDir() string {
	configDir := path.Dir(mustExpandPath(cm.configPath))
	projectRoot := cm.activeProject.Root
	if projectRoot == "" {
		projectRoot = configDir
	} else {
		projectRoot = mustExpandPath(projectRoot)
		if !path.IsAbs(projectRoot) {
			projectRoot = path.Join(configDir, projectRoot)
		}
	}
	return path.Clean(projectRoot)
}

func (cm *ProjectManager) RootPath(elem ...string) string {
	elem = append([]string{cm.RootDir()}, elem...)
	return path.Join(elem...)
}

func (cm *ProjectManager) Modulers() ([]*NamedModuler, error) {
	cm.Do(func() {
		var modules []*NamedModuler
		for _, moduleName := range cm.activeProject.Modules {
			found := false
			for _, module := range cm.Modules {
				if module.Name == moduleName {
					found = true
					moduleMaker, ok := ModuleMakers[module.Type]
					if !ok {
						cm.lmError = fmt.Errorf("module %s for project %s has unkown type %s", moduleName, cm.activeProject.Name, module.Type)
						return
					}
					preparedModule, err := moduleMaker(cm, module)
					if err != nil {
						cm.lmError = fmt.Errorf("module %s for project %s could not initializaed, had err %s", module.Name, cm.activeProject.Name, err)
						return
					}
					modules = append(modules, &NamedModuler{Name: moduleName, Module: preparedModule})
				}
			}
			if !found {
				cm.lmError = fmt.Errorf("module %s not found for project %s", moduleName, cm.activeProject.Name)
			}
		}
		cm.loadedModules = modules
	})

	return cm.loadedModules, cm.lmError
}

func (cm *ProjectManager) ShellActivateSh() (string, error) {
	modules, err := cm.Modulers()
	if err != nil {
		return "", err
	}
	activationCommands := []string{}
	// Go forwards for activate backwards for deactivate
	for _, namedModular := range modules {
		lines, err := namedModular.Module.ShellActivateCommands()
		if err != nil {
			return "", fmt.Errorf("module %s for project %s could not generate activation lines, had err %s", namedModular.Name, cm.activeProject.Name, err)
		}
		activationCommands = append(activationCommands, lines...)
	}
	return strings.Join(activationCommands, " || return $?\n"), nil
}

func (cm *ProjectManager) ShellDeactivateSh() (string, error) {
	modules, err := cm.Modulers()
	if err != nil {
		return "", err
	}
	activationCommands := []string{}
	// Go forwards for activate backwards for deactivate
	for i := len(modules) - 1; i >= 0; i-- {
		namedModuler := modules[i]
		lines, err := namedModuler.Module.ShellDeactivateCommands()
		if err != nil {
			return "", fmt.Errorf("module %s for project %s could not generate activation lines, had err %s", namedModuler.Name, cm.activeProject.Name, err)
		}
		activationCommands = append(activationCommands, lines...)
	}
	return strings.Join(activationCommands, " || return 1\n"), nil
}

func (cm *ProjectManager) AppendModulesOnProject(modules ...*Module) {
	// Todo: handle conflicts
	cm.Modules = append(cm.Modules, modules...)
	for _, module := range modules {
		cm.activeProject.Modules = append(cm.activeProject.Modules, module.Name)
	}
}

func NewProjectManager(config *Config, configPath string, activeProject string, storageRoot string) (*ProjectManager, error) {
	storageRoot = mustExpandPath(storageRoot)
	manager := &ProjectManager{Config: config, configPath: configPath, storageRoot: storageRoot}
	for _, proj := range config.Projects {
		if proj.Name == activeProject {
			manager.activeProject = proj
		}
	}
	if manager.activeProject == nil {
		return nil, fmt.Errorf("project %s not found in config", activeProject)
	}
	err := os.MkdirAll(storageRoot, 0700)
	if err != nil {
		return nil, err
	}
	return manager, nil
}
