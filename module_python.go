package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

const DefaultPython = "python"
const DefaultVirtualenv = "virtualenv"
const DefaultPipInstallCommand = "pip install"

type PyModuleConfig struct {
	Python                string
	AutoInstallDeps       bool `json:"auto_install_deps"`
	Dependencies          []string
	PipInstallCommand     string `json:"pip_install_command"`
	PrepPipInstallCommand string `json:"prep_pip_install_command"`
	VirtualEnvCommand     string `json:"virtualenv_command"`
}

type PythonModule struct {
	manager *ProjectManager
	config  *PyModuleConfig
}

func (pm *PythonModule) venvDir() string {
	return pm.manager.StoragePath("pyvenv")
}

func (pm *PythonModule) activteShPath() string {
	return path.Join(pm.venvDir(), "bin", "activate")
}

func (pm *PythonModule) autoInstallHashPath() string {
	return path.Join(pm.venvDir(), "autoinstall_dep_sha.txt")
}

func (pm *PythonModule) autoInstallLastHash() string {
	data, err := ioutil.ReadFile(pm.autoInstallHashPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (pm *PythonModule) autoInstallCalculateDepHash() (string, error) {
	if len(pm.config.Dependencies) == 0 {
		return "", nil
	}
	hash := md5.New()
	for _, dep := range pm.config.Dependencies {
		// read the deps if its a file, preferring .txt instead of isFile type check for safety sake
		if strings.HasSuffix(dep, ".txt") {
			fullPath := dep
			if !path.IsAbs(dep) {
				fullPath = pm.manager.RootPath(dep)
			}
			data, err := ioutil.ReadFile(fullPath)
			if err != nil {
				return "", err
			}
			hash.Write(data)
		} else {
			hash.Write([]byte(dep))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (pm *PythonModule) autoInstallArgs() string {
	args := []string{}
	for _, dep := range pm.config.Dependencies {
		if strings.HasSuffix(dep, ".txt") {
			fullPath := pm.manager.RootPath(dep)
			args = append(args, fmt.Sprintf("-r %s", fullPath))
		} else {
			args = append(args, dep)
		}
	}
	return strings.Join(args, " ")
}

func (pm *PythonModule) venvExists() bool {
	return pathExists(pm.activteShPath())
}

func (pm *PythonModule) ShellActivateCommands() ([]string, error) {
	currentDepHash, err := pm.autoInstallCalculateDepHash()
	if err != nil {
		return nil, err
	}
	lastDepHash := pm.autoInstallLastHash()
	hashChanged := currentDepHash != lastDepHash
	lines := []string{}
	addArgs := func(args ...string) {
		lines = append(lines, strings.Join(args, " "))
	}
	if !pm.venvExists() {
		// Upgrade pip/venv before creating the venv [pip install --upgrade pip virtualenv]
		addArgs(pm.config.PrepPipInstallCommand, "--upgrade", "pip", "virtualenv")
		// Create the venv [virtualenv -p python /path/to/venv]
		addArgs(pm.config.VirtualEnvCommand, "-p", pm.config.Python, pm.venvDir())
	}

	// Activate the venv '.' = source [. /path/venv/bin/activate]
	addArgs("VIRTUAL_ENV_DISABLE_PROMPT=1", ".", pm.activteShPath())

	if !pm.venvExists() {
		// If the venv is new lets upgrade pip in the venv as well [pip install --upgrade pip]
		addArgs(pm.config.PrepPipInstallCommand, "--upgrade", "pip")
	}

	if hashChanged && pm.config.AutoInstallDeps {
		// run the install [pip install -r requirements.txt deps]
		addArgs(pm.config.PipInstallCommand, pm.autoInstallArgs())
		// write the hash so we don't reinstaall these deps [echo sd2if1jdfs > .venvy/project/pyvenv/auto_install.txt]
		addArgs("echo", currentDepHash, ">", pm.autoInstallHashPath())
	}
	return lines, nil
}

func (pm *PythonModule) ShellDeactivateCommands() ([]string, error) {
	return []string{"deactivate"}, nil
}

func NewPythonModule(manager *ProjectManager, self *Module) (Moduler, error) {
	moduleConfig := &PyModuleConfig{}
	err := unmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	if moduleConfig.Python == "" {
		moduleConfig.Python = DefaultPython
	}
	if moduleConfig.PipInstallCommand == "" {
		moduleConfig.PipInstallCommand = DefaultPipInstallCommand
	}
	if moduleConfig.PrepPipInstallCommand == "" {
		moduleConfig.PrepPipInstallCommand = DefaultPipInstallCommand
	}
	if moduleConfig.VirtualEnvCommand == "" {
		moduleConfig.VirtualEnvCommand = DefaultVirtualenv

	}
	return &PythonModule{manager: manager, config: moduleConfig}, nil
}
