package main

import (
	"context"
	"encoding/json"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

const ProjectName = "venvy"
const Version = "0.0.0"

var ActivateFileEnvVar = fmt.Sprintf("%s_ACTIVATE_FILE", strings.ToUpper(ProjectName))
var DeactivateFileEnvVar = fmt.Sprintf("%s_DEACTIVATE_FILE", strings.ToUpper(ProjectName))
var EvalHeleperCommand = fmt.Sprintf(`eval $(%s shell-init)`, ProjectName)

var rootCmd = &cobra.Command{
	Use:   ProjectName,
	Short: "Context managers for shell.",
}

// Whether the local users shell is setup (i.e was the activator initialized in .bashrc/etc)
func EvalPaths() (activate string, deactivate string, bothSet bool) {
	activate = os.Getenv(ActivateFileEnvVar)
	deactivate = os.Getenv(DeactivateFileEnvVar)
	bothSet = activate != "" && deactivate != ""
	return
}

const evalTmpl = `
current_cmd_type=$(command -V {{ .ProjectName }});
if [ "${current_cmd_type#*function}" = "$current_cmd_type" ]; then
	original_{{.ProjectName}}_cmd=$(command -v {{ .ProjectName }});
	function {{ .ProjectName }}(){
		devenv || true;
		activate_f=$(mktemp);
		deactivate_f=$(mktemp);
		export DEACTIVATE_F=${deactivate_f};
		env {{ .ActivateFileEnvVar }}=${activate_f} {{ .DeactivateFileEnvVar }}=${deactivate_f} ${original_{{.ProjectName}}_cmd} $@ || return $?;
		if [ -s ${activate_f} ]; then
			. ${activate_f};
		fi;
		rm ${activate_f} > /dev/null 2>&1 || true;
		unset activate_f;
		unset deactivate_f;
	};
	function devenv(){
		if [ ! -z "${DEACTIVATE_F}" ] && [ -s ${DEACTIVATE_F} ]; then
			. ${DEACTIVATE_F};
		fi;
		rm ${DEACTIVATE_F} >/dev/null 2>&1 || true;
		unset DEACTIVATE_F;
	};
fi;
`

var defaultModules = []*Module{
	{Name: "jump_builtin", Type: "jump"},
	{Name: "ps1_builtin", Type: "ps1"},
}

func evalScript() (string, error) {
	originalCmd := os.Args[0]
	return stringTemplate("evalTmpl", evalTmpl, struct {
		ProjectName          string
		ActivateFileEnvVar   string
		DeactivateFileEnvVar string
		OriginalCmd          string
	}{
		ProjectName:          ProjectName,
		ActivateFileEnvVar:   ActivateFileEnvVar,
		DeactivateFileEnvVar: DeactivateFileEnvVar,
		OriginalCmd:          originalCmd,
	})
}

func issueExec(manager *ProjectManager, cmd string) {
	// Add exec module
	execConfig, err := json.Marshal(&ExecConfig{ActivationCommands: []string{cmd}})
	errExit(err)
	execModule := &Module{Name: "exec_subcommand", Type: "exec", Config: json.RawMessage(execConfig)}
	manager.AppendModulesOnProject(execModule)

	// Setup activation scripts`
	activationScript, err := manager.ShellActivateSh()
	errExit(err)
	deactivationScript, err := manager.ShellDeactivateSh()
	errExit(err)
	scriptBody := strings.Join([]string{
		"set -e",
		activationScript,
		deactivationScript,
	}, "\n")

	// Write activation script
	f, err := ioutil.TempFile("", "eval")
	errExit(err)
	f.Write([]byte(scriptBody))
	f.Close()
	logger.Debugf("wrote exec file to %s", f.Name())

	// Execute command
	cmdArgs := []string{"sh", f.Name()}
	baseContext, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	execCmd := exec.CommandContext(baseContext, "/usr/bin/env", cmdArgs...)
	execCmd.Stderr = os.Stderr
	execCmd.Stdout = os.Stdout
	execCmd.Stdin = os.Stdin
	go func() {
		signalC := make(chan os.Signal, 1)
		signal.Notify(signalC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case c := <-signalC:
			logger.Warnf("Got signal %s", c.String())
			cancelFunc()
		case <-baseContext.Done():
			return
		}
	}()
	err = execCmd.Run()
	errExit(err)
}

func issueActivate(manager *ProjectManager, activatePath string, deactivatePath string) {
	// prep activation scripts
	activationScript, err := manager.ShellActivateSh()
	errExit(err)
	deactivationScript, err := manager.ShellDeactivateSh()
	errExit(err)

	// write activation scripts
	err = ioutil.WriteFile(activatePath, []byte(activationScript), 0600)
	logger.Debugf("Writing activation file to %s with contents:\n %s", activatePath, activationScript)
	errExit(err)
	err = ioutil.WriteFile(deactivatePath, []byte(deactivationScript), 0600)
	logger.Debugf("Writing deactivation file to %s with contents:\n %s", deactivatePath, deactivationScript)
	errExit(err)
}

func preFuncCmd(manager *ProjectManager) {
	reset, err := rootCmd.PersistentFlags().GetBool("reset")
	errExit(err)
	if reset {
		path := manager.StoragePath()
		logger.Debugf("Deleting data dir %s", path)
		err := os.RemoveAll(path)
		errExit(err)
		logger.Debugf("Recreating data dir %s", path)
		err = os.MkdirAll(path, 0700)
		errExit(err)
	}

	temp, err := rootCmd.PersistentFlags().GetBool("temp")
	errExit(err)
	if temp {
		name, err := ioutil.TempDir("", ProjectName)
		errExit(err)
		logger.Debugf("Using temp dir for venv %s", name)
		manager.storageRoot = name
	}
}

func makeActivationCommand(config *Config, configF *foundConfig, project *Project) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		manager, err := NewProjectManager(config, configF.Path, project.Name, configF.StorageDir)
		preFuncCmd(manager)
		if !project.DisableBuiltinModules {
			manager.AppendModulesOnProject(defaultModules...)
		}
		errExit(err)
		if len(args) == 0 {
			// Activation
			activatePath, deactivatePath, bothSet := EvalPaths()
			if bothSet {
				issueActivate(manager, activatePath, deactivatePath)
			} else {
				err = fmt.Errorf("please add `%s` to your .bashrc/.zshrc to enable shell support", EvalHeleperCommand)
				errExit(err)
			}
		} else {
			issueExec(manager, strings.Join(args, " "))
		}
	}
}

func makeScriptCommand(config *Config, configF *foundConfig, project *Project, script *foundScript) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		manager, err := NewProjectManager(config, configF.Path, project.Name, configF.StorageDir)
		preFuncCmd(manager)
		errExit(err)
		issueExec(manager, script.ExecPrefix+" "+script.FilePath)
	}
}

func LoadConfigCommands() ([]*cobra.Command, error) {
	cmds := []*cobra.Command{}
	configs := LoadConfigs(true, true)
	seenProjects := map[string]string{}
	for _, configF := range configs {
		config := configF.Config()
		if config == nil {
			continue
		}
		for _, project := range config.Projects {
			existingPath, ok := seenProjects[project.Name]
			if ok {
				logger.Warnf("Project %s already exists in file %s, skipping the one from file %s. Please resolve conflict.", project.Name, existingPath, configF.Path)
				continue
			}
			seenProjects[project.Name] = configF.Path
			activateCommand := &cobra.Command{
				Use:   project.Name,
				Short: fmt.Sprintf("Activate environment %s", project.Name),
				Run:   makeActivationCommand(config, configF, project),
			}
			cmds = append(cmds, activateCommand)
			for _, script := range configF.Scripts()[project.Name] {
				subCommand := &cobra.Command{
					Use:   fmt.Sprintf("%s.%s", project.Name, script.SubCommand),
					Short: script.Docstring,
					Run:   makeScriptCommand(config, configF, project, script),
				}
				cmds = append(cmds, subCommand)

			}
		}
	}
	return cmds, nil

}

var evalCmd = &cobra.Command{
	Use:   "shell-init",
	Short: "Shell helper",
	Run: func(cmd *cobra.Command, args []string) {
		evalScript, err := evalScript()
		errExit(err)
		fmt.Println(evalScript)
	},
}

func errExit(err error) {
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func handleCliInit() {
	// Initialize the logger
	debug, err := rootCmd.PersistentFlags().GetBool("debug")
	errExit(err)
	if debug {
		logger.SetLevel(logger.DebugLevel)
	}
}

func main() {
	var err error
	rootCmd.PersistentFlags().Bool("debug", false, fmt.Sprintf("debug %s", ProjectName))
	rootCmd.PersistentFlags().Bool("reset", false, fmt.Sprintf("reset the environment data before initalizing"))
	rootCmd.PersistentFlags().Bool("temp", false, fmt.Sprintf("create a temp data dir for the session"))

	logger.SetOutput(os.Stderr) // default but explicit

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(evalCmd)
	cobra.OnInitialize(handleCliInit)

	subCommand := ""
	if len(os.Args) > 1 {
		subCommand = os.Args[1]
	}

	if subCommand != "version" && subCommand != "shell-init" {
		configCmds, err := LoadConfigCommands()
		if err != nil {
			errExit(err)
		}

		for _, cmd := range configCmds {
			rootCmd.AddCommand(cmd)
		}
	}
	err = rootCmd.Execute()
	errExit(err)
}
