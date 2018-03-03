package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-homedir"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var defaultFileName = fmt.Sprintf("%s.toml", ProjectName)
var seenConfigsPath = globalPath("seen_configs.json")
var scriptDocstringRe = regexp.MustCompile(`^[\-;#/\s}{]+["']([^"']+)["']$`)

func dotDir(inDir string) string {
	return path.Join(inDir, "."+ProjectName)
}

func globalPath(elem ...string) string {
	homeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return path.Join(append([]string{dotDir(homeDir)}, elem...)...)
}

type foundScript struct {
	LastModified string
	FilePath     string
	SubCommand   string
	Docstring    string
	ExecPrefix   string
}

type foundConfig struct {
	Path           string                    `json:"path"`
	StorageDir     string                    `json:"storage_dir"`
	ProjectScripts map[string][]*foundScript `json:"known_scripts"`
	config         *Config
	loadOnce       sync.Once
	scriptsOnce    sync.Once
}

func (f *foundConfig) loadConfig() {
	data, err := ioutil.ReadFile(f.Path)
	if err != nil {
		logger.Warnf("unable to read config with error %s", err)
		return
	}
	jsonData, err := tomlToJson(data)
	if err != nil {
		logger.Warnf("unable to convert config with error %s", err)
		return
	}
	newConfig := &Config{}
	err = json.Unmarshal(jsonData, newConfig)
	if err != nil {
		logger.Warnf("unable to unmarshal config with error %s", err)
		return
	}
	err = validateStruct(newConfig)
	if err != nil {
		logger.Warnf("unable to validate config with error %s", err)
		return
	}

	f.config = newConfig
	logger.Debugf("Loaded %d modules and %d projects from config %s", len(newConfig.Modules), len(newConfig.Projects), f.Path)
}

func (f *foundConfig) Config() *Config {
	f.loadOnce.Do(f.loadConfig)
	return f.config
}

var extExecPrefix = map[string]string{
	".py":   "/usr/bin/env python",
	".js":   "/usr/bin/env node",
	".rb":   "/usr/bin/env ruby",
	".bash": "/usr/bin/env bash",
	".sh":   "/usr/bin/env sh",
}

func extractScript(path string, f os.FileInfo) (*foundScript, error) {
	nameExt := f.Name()
	extension := filepath.Ext(nameExt)
	name := nameExt[0 : len(nameExt)-len(extension)]
	isClean := cleanNameRe.MatchString(name)
	if !isClean {
		return nil, fmt.Errorf("script %s name does not match regex [a-z_-]+ less the extension", name)
	}
	script := &foundScript{
		FilePath:     path,
		SubCommand:   name,
		LastModified: f.ModTime().Format(time.RFC3339),
	}
	if f.Mode()&0111 == 0 {
		// Not execable need exec prefix
		var ok bool
		script.ExecPrefix, ok = extExecPrefix[extension]
		if !ok {
			return nil, fmt.Errorf("script %s not executable and does not have known extension. chmod +x the file an add a shebang.", name)
		}
	}
	fOpen, err := os.Open(path)
	defer fOpen.Close()
	if err != nil {

	}
	scanner := bufio.NewScanner(fOpen)
	linesToScan := 5
	for scanner.Scan() {
		line := scanner.Text()
		match := scriptDocstringRe.FindStringSubmatch(line)
		if len(match) > 1 {
			script.Docstring = match[1]
			break
		}
		linesToScan--
		if linesToScan <= 0 {
			script.Docstring = "No docstring"
			break
		}

	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	return script, nil
}

func (f *foundConfig) loadScripts() {
	f.scriptsOnce.Do(func() {
		allScripts := map[string][]*foundScript{}
		config := f.Config()
		if config == nil {
			return
		}
		for _, project := range config.Projects {
			scriptCacheF := path.Join(f.StorageDir, project.Name, fmt.Sprintf("script_cache_%s.json", project.Name))
			if len(project.ScriptSubcommands) == 0 {
				continue
			}
			os.MkdirAll(path.Dir(scriptCacheF), 0700)
			data, _ := ioutil.ReadFile(scriptCacheF)
			cacheFnameScripts := map[string]*foundScript{}
			err := unmarshalEmpty(data, &cacheFnameScripts)
			if err != nil {
				logger.Debugf("unable to load cache scripts for project %s with err %s", project.Name, err)
			}
			for _, scSource := range project.ScriptSubcommands {
				if !path.IsAbs(scSource) {
					scSource = path.Join(path.Dir(f.Path), scSource)
				}
				fInfo, err := os.Stat(scSource)
				if err != nil {
					logger.Warnf("unable to load scripts from %s for project %s", scSource, project.Name)
					continue
				}
				var files []os.FileInfo
				if fInfo.IsDir() {
					files, err = ioutil.ReadDir(scSource)
				} else {
					files = []os.FileInfo{fInfo}
					scSource = path.Dir(scSource)
				}

				for _, file := range files {
					fname := path.Join(scSource, file.Name())
					cachedScript, ok := cacheFnameScripts[fname]
					if ok && file.ModTime().Format(time.RFC3339) == cachedScript.LastModified {
						allScripts[project.Name] = append(allScripts[project.Name], cachedScript)
					} else {
						parsedScript, err := extractScript(fname, file)
						if err != nil {
							logger.Warnf("unable to load scripts from %s for project %s", fname, project.Name)
							continue
						}
						allScripts[project.Name] = append(allScripts[project.Name], parsedScript)
						cacheFnameScripts[fname] = parsedScript
					}
				}
			}
			if len(cacheFnameScripts) > 0 {
				cacheJson, err := json.Marshal(cacheFnameScripts)
				if err != nil {
					logger.Warnf("unable to marshal cache scripts for project %s with err %s", project.Name, err)
				}
				err = ioutil.WriteFile(scriptCacheF, cacheJson, 0600)
				if err != nil {
					logger.Warnf("unable to save cache scripts for project %s with err %s", project.Name, err)
				}
			}
		}
		f.ProjectScripts = allScripts
	})

}

func (f *foundConfig) Scripts() map[string][]*foundScript {
	f.loadScripts()
	return f.ProjectScripts
}

func configPathsFromGit() []*foundConfig {
	paths := []*foundConfig{}
	gitRoot, err := findPathInAncestors("", ".git")
	storageDir := dotDir(gitRoot)
	if err != nil {
		return paths
	}
	dotGitDir := path.Join(gitRoot, ".git")
	baseArgs := []string{"--git-dir", dotGitDir, "--work-tree", gitRoot}

	addedFilesArgs := []string{"ls-files", "--others", "--exclude-standard", "--full-name"}
	lsSubmodulesArgs := []string{"ls-files", "--recurse-submodules", "--full-name"}

	for _, args := range [][]string{addedFilesArgs, lsSubmodulesArgs} {
		cmd := exec.Command("git", append(baseArgs, args...)...)
		data, err := cmd.Output()
		if err != nil {
			logger.Debugf("ran into err %s doing git ls-files", err)
		} else {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for _, line := range lines {
				if strings.HasSuffix(line, defaultFileName) {
					paths = append(paths, &foundConfig{Path: path.Join(gitRoot, line), StorageDir: storageDir})
				}
			}
		}
	}
	return paths
}

func configPathsFromHistory() []*foundConfig {
	foundConfigs := []*foundConfig{}
	data, err := ioutil.ReadFile(seenConfigsPath)
	if err != nil {
		logger.Debugf("ran into err reading seen configs %s", err)
		return nil
	}
	err = json.Unmarshal(data, &foundConfigs)
	if err != nil {
		logger.Debugf("ran into err unmarshaling seen configs %s", err)
		return nil
	}
	return foundConfigs
}

func configsPathsFromPwd() []*foundConfig {
	workDir, err := os.Getwd()
	if err != nil {
		logger.Debugf("ran into err getting cwd %s", err)
		return nil
	}
	inDirConfig := path.Join(workDir, defaultFileName)
	_, err = os.Stat(inDirConfig)
	if err != nil {
		return nil
	}
	return []*foundConfig{{Path: inDirConfig, StorageDir: dotDir(workDir)}}
}

func LoadConfigs(prefetch bool, saveInHistory bool) []*foundConfig {
	allDiscovered := [][]*foundConfig{
		configPathsFromGit(),
		configsPathsFromPwd(),
		configPathsFromHistory(),
	}
	uniqueConfigs := []*foundConfig{}
	pathsSeen := map[string]bool{}
	for _, discoveredConfigs := range allDiscovered {
		for _, config := range discoveredConfigs {
			_, ok := pathsSeen[config.Path]
			if !ok {
				pathsSeen[config.Path] = true
				if prefetch {
					// go config.Config()
					// Scripts calls .Config()
					go config.Scripts()
				}
				uniqueConfigs = append(uniqueConfigs, config)
			}
		}
	}
	if saveInHistory {
		data, err := json.Marshal(uniqueConfigs)
		if err != nil {
			logger.Debugf("unable to marshall history file with err %s", err)
		}
		err = ioutil.WriteFile(seenConfigsPath, data, 0600)
		if err != nil {
			logger.Debugf("Unable to save to history file at %s with err %s", seenConfigsPath, err)
		}
	}
	return uniqueConfigs
}

func init() {
	os.MkdirAll(globalPath(), 0700)
}
