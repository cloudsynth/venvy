package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pnegahdar/venvy/modules"
	"github.com/pnegahdar/venvy/util"
	"github.com/pnegahdar/venvy/manager"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"github.com/fatih/color"
)

var activateFileEnvVar = fmt.Sprintf("%s_ACTIVATE_FILE", strings.ToUpper(venvy.ProjectName))
var deactivateFileEnvVar = fmt.Sprintf("%s_DEACTIVATE_FILE", strings.ToUpper(venvy.ProjectName))
var disableHistoryEnvVar = fmt.Sprintf("%s_DISABLE_CONFIG_HISTORY", strings.ToUpper(venvy.ProjectName))
var evalHeleperCommand = fmt.Sprintf(`eval $(%s shell-init)`, venvy.ProjectName)

var rootCmd = &cobra.Command{
	Use:   venvy.ProjectName,
	Short: "Context managers for shell.",
}

// Whether the local users shell is setup (i.e was the activator initialized in .bashrc/etc)
func EvalPaths() (activate string, deactivate string, bothSet bool) {
	activate = os.Getenv(activateFileEnvVar)
	deactivate = os.Getenv(deactivateFileEnvVar)
	bothSet = activate != "" && deactivate != ""
	return
}

const evalTmpl = `
current_cmd_type=$(command -V {{ .ProjectName }});
if [ "${current_cmd_type#*function}" = "$current_cmd_type" ]; then
	original_{{.ProjectName}}_cmd=$(command -v {{ .ProjectName }});
	function {{ .ProjectName }}(){
		activate_f=$(mktemp);
		deactivate_f=$(mktemp);
		env {{ .ActivateFileEnvVar }}=${activate_f} {{ .DeactivateFileEnvVar }}=${deactivate_f} ${original_{{.ProjectName}}_cmd} $@ || return $?;
		if [ -s ${activate_f} ]; then
			devenv || true;
			env {{ .ActivateFileEnvVar }}=${activate_f} {{ .DeactivateFileEnvVar }}=${deactivate_f} ${original_{{.ProjectName}}_cmd} $@ || return $?;
			export DEACTIVATE_F=${deactivate_f};
			. ${activate_f} || return $?;
		fi;
		rm ${activate_f} > /dev/null 2>&1 || true;
		unset activate_f;
		unset deactivate_f;
	};
	function devenv(){
		if [ ! -z "${DEACTIVATE_F}" ] && [ -s "${DEACTIVATE_F}" ]; then
			. ${DEACTIVATE_F} || return $?;
		fi;
		rm ${DEACTIVATE_F} >/dev/null 2>&1 || true;
		unset DEACTIVATE_F;
	};
fi;
`

var defaultJumpModule = &venvy.Module{Name: "jump_builtin", Type: "jump"}
var defaultPS1Module = &venvy.Module{Name: "ps1_builtin", Type: "ps1"}

func evalScript() (string, error) {
	originalCmd := os.Args[0]
	return util.StringTemplate("evalTmpl", evalTmpl, struct {
		ProjectName          string
		ActivateFileEnvVar   string
		DeactivateFileEnvVar string
		OriginalCmd          string
	}{
		ProjectName:          venvy.ProjectName,
		ActivateFileEnvVar:   activateFileEnvVar,
		DeactivateFileEnvVar: deactivateFileEnvVar,
		OriginalCmd:          originalCmd,
	})
}

func issueExec(manager *venvy.ProjectManager, cmd string) {
	// Add exec module
	execConfig, err := json.Marshal(&modules.ExecConfig{ActivationCommands: []string{cmd}})
	errExit(err)
	execModule := &venvy.Module{Name: "exec_subcommand", Type: "exec", Config: json.RawMessage(execConfig)}
	manager.AppendModules(execModule)

	// Setup activation scripts`
	activationScript, err := manager.ShellActivateCommands()
	errExit(err)
	deactivationScript, err := manager.ShellDeactivateCommands()
	errExit(err)
	scriptBody := strings.Join([]string{
		"set -e",
		strings.Join(activationScript, "\n"),
		strings.Join(deactivationScript, "\n"),
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

func issueActivate(manager *venvy.ProjectManager, activatePath string, deactivatePath string) {
	// prep activation scripts
	activationLines, err := manager.ShellActivateCommands()
	errExit(err)
	activationScript := []byte(strings.Join(activationLines, " && \\\n"))
	deactivationLines, err := manager.ShellDeactivateCommands()
	errExit(err)
	deactivationScript := []byte(strings.Join(deactivationLines, " && \\\n"))

	// write activation scripts
	err = ioutil.WriteFile(activatePath, activationScript, 0600)
	logger.Debugf("Writing %s file to %s with contents:\n\n%s\n", color.GreenString("activation"), activatePath, activationScript)
	errExit(err)
	err = ioutil.WriteFile(deactivatePath, deactivationScript, 0600)
	logger.Debugf("Writing deactivation file to %s with contents:\n\n%s\n", color.BlueString("deactivation"), deactivatePath, deactivationScript)
	errExit(err)
}

func preSubCommand(cmd *cobra.Command, manager *venvy.ProjectManager) {
	reset, err := cmd.Flags().GetBool("reset")
	errExit(err)
	if reset {
		err := manager.Reset()
		errExit(err)
	}

	temp, err := cmd.Flags().GetBool("temp")
	errExit(err)
	if temp {
		name, err := ioutil.TempDir("", venvy.ProjectName)
		errExit(err)
		err = manager.ChDir(name)
		errExit(err)
		logger.Debugf("Using temp dir for venv %s", name)
	}
}

func makeActivationCommand(manager *venvy.ProjectManager) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		showRoot, err := cmd.Flags().GetBool("print-root")
		errExit(err)
		if showRoot {
			fmt.Println(manager.RootDir())
			os.Exit(0)
		}
		preSubCommand(cmd, manager)
		if !manager.Project.DisableBuiltinModules {
			manager.PrependModules(defaultPS1Module, defaultJumpModule)
		}
		if len(args) == 0 {
			// Activation
			activatePath, deactivatePath, bothSet := EvalPaths()
			if bothSet {
				issueActivate(manager, activatePath, deactivatePath)
			} else {
				err := fmt.Errorf("please add `%s` to your .bashrc/.zshrc to enable shell support", evalHeleperCommand)
				errExit(err)
			}
		} else {
			issueExec(manager, strings.Join(args, " "))
		}
	}
}

func makeScriptCommand(manager *venvy.ProjectManager, script *foundScript) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		showPath, err := cmd.Flags().GetBool("print-path")
		errExit(err)
		if showPath {
			fmt.Println(script.FilePath)
			os.Exit(0)
		}
		preSubCommand(cmd, manager)
		toExec := script.FilePath
		if script.ExecPrefix != " " {
			toExec = script.ExecPrefix + " " + toExec
		}
		if len(args) > 0 {
			toExec += " " + strings.Join(args, " ")
		}
		issueExec(manager, toExec)
	}
}

func isCIEnv() bool {
	_, isCI := os.LookupEnv("CI")
	_, isJenkins := os.LookupEnv("JENKINS_URL")
	return isCI || isJenkins
}

func LoadConfigCommands() ([]*cobra.Command, error) {
	cmds := []*cobra.Command{}
	useHistory := true
	if isCIEnv(){
		logger.Debug("Not using config history, CI environment detected.")
	}
	if os.Getenv(disableHistoryEnvVar) != "" {
		logger.Debugf("Not using config history because envar %s is set", disableHistoryEnvVar)
		useHistory = false
	}
	foundConfigs := LoadConfigs(true, useHistory)
	seenProjects := map[string]string{}
	for _, configF := range foundConfigs {
		config := configF.Config()
		if config == nil {
			continue
		}
		configManager, err := venvy.NewConfigManager(config, configF.Path, configF.StorageDir, modules.DefaultModuleMakers)
		errExit(err)
		for _, project := range config.Projects {
			existingPath, ok := seenProjects[project.Name]
			if ok {
				logger.Warnf("Project %s already exists in file %s, skipping the one from file %s. Please resolve conflict.", project.Name, existingPath, configF.Path)
				continue
			}
			seenProjects[project.Name] = configF.Path
			projectManager, err := configManager.ProjectManager(project.Name)
			errExit(err)
			activateCommand := &cobra.Command{
				Use:   project.Name,
				Short: fmt.Sprintf("Activate environment %s", project.Name),
				Run:   makeActivationCommand(projectManager),
			}
			activateCommand.Flags().Bool("reset", false, fmt.Sprintf("reset the environment data before initalizing"))
			activateCommand.Flags().Bool("temp", false, fmt.Sprintf("create a temp data dir for the session"))
			activateCommand.Flags().Bool("print-root", false, fmt.Sprintf("print the root dir of the project"))

			cmds = append(cmds, activateCommand)
			for _, script := range configF.Scripts()[project.Name] {
				subCommand := &cobra.Command{
					Use:   fmt.Sprintf("%s.%s", project.Name, script.SubCommand),
					Short: script.Docstring,
					Run:   makeScriptCommand(projectManager, script),
				}
				subCommand.Flags().Bool("reset", false, fmt.Sprintf("reset the environment data before initalizing"))
				subCommand.Flags().Bool("temp", false, fmt.Sprintf("create a temp data dir for the session"))
				subCommand.Flags().Bool("print-path", false, fmt.Sprintf("print the path of the script"))
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
	debug, err := rootCmd.PersistentFlags().GetBool("verbose")
	errExit(err)
	if debug {
		logger.SetLevel(logger.DebugLevel)
	}
}

func main() {
	var err error
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose logging")

	logger.SetOutput(os.Stderr) // default but explicit
	logger.SetLevel(logger.ErrorLevel)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(venvy.Version)
		},
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(evalCmd)
	cobra.OnInitialize(handleCliInit)

	// Set debug early on so
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose"{
			logger.SetLevel(logger.DebugLevel)
		}
		if arg == "--"{
			break
		}
	}
	configCmds, err := LoadConfigCommands()
	if err != nil {
		errExit(err)
	}

	for _, cmd := range configCmds {
		rootCmd.AddCommand(cmd)
	}
	err = rootCmd.Execute()
	errExit(err)
}
