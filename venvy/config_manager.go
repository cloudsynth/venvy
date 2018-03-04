package venvy

import (
	"fmt"
)

type ConfigManager struct {
	*DataManager
	config       *Config
	configPath   string
	ModuleMakers ModuleMakerTypeMap
	pmCache      map[string]*ProjectManager
}

func (cm *ConfigManager) ProjectManager(projectName string) (*ProjectManager, error) {
	cachedV, ok := cm.pmCache[projectName]
	if ok {
		return cachedV, nil
	}
	var project *Project
	for _, proj := range cm.config.Projects {
		if proj.Name == projectName {
			project = proj
		}
	}
	if project == nil {
		return nil, fmt.Errorf("project %s not found in config", projectName)
	}
	dataManager, err := NewDataManager(cm.StoragePath(projectName))
	if err != nil {
		return nil, err
	}
	relatedModules := map[string]*Module{}
	for _, moduleName := range project.Modules {
		for _, configModule := range cm.config.Modules {
			if configModule.Name == moduleName {
				relatedModules[moduleName] = configModule
				break
			}
		}
		if _, ok := relatedModules[moduleName]; !ok {
			return nil, fmt.Errorf("module %s not found which is needed for project %s", moduleName, projectName)
		}
	}
	return &ProjectManager{
		DataManager:    dataManager,
		configManager:  cm,
		Project:        project,
		relatedModules: relatedModules,
	}, nil
}

func NewConfigManager(config *Config, configPath string, storageDir string, makerMap ModuleMakerTypeMap) (*ConfigManager, error) {
	dataManager, err := NewDataManager(storageDir)
	if err != nil {
		return nil, err
	}
	configM := &ConfigManager{
		DataManager:  dataManager,
		ModuleMakers: makerMap,
		config:       config,
		configPath:   configPath,
	}
	return configM, nil
}
