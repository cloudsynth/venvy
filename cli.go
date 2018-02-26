package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const ProjectName = "venvy"
const Version = "0.0.0"

var EvalFileEnvVar = fmt.Sprintf("%s_EVAL_FILE", strings.ToUpper(ProjectName))

var rootCmd = &cobra.Command{
	Use:   ProjectName,
	Short: "Context managers for shell.",
}

// Whether the local users shell is setup (i.e was the activator initialized in .bashrc/etc)
func isShellSetup() bool {
	return false
}

const evalTmpl = `
current_cmd_type=$(command -V {{ .ProjectName }})
if [ "${current_cmd_type#*function}" = "$current_cmd_type" ]; then
	original_cmd=$(command -v {{ .ProjectName }})
	function {{ .ProjectName }}(){
		eval_f=$(mktemp)
		env {{ .EvalFileEnvVar }}=${eval_f} ${original_cmd} $@
		if [ -s ${eval_f} ]; then
			. ${evalF}
		fi
	}
fi
`

func evalScript() (string, error) {
	originalCmd := os.Args[0]
	return stringTemplate("evalTmpl", evalTmpl, struct {
		ProjectName    string
		EvalFileEnvVar string
		OriginalCmd    string
	}{
		ProjectName:    ProjectName,
		EvalFileEnvVar: EvalFileEnvVar,
		OriginalCmd:    originalCmd,
	})
}

func LoadConfigCommands() ([]*cobra.Command, error) {
	cmds := []*cobra.Command{}
	managers, err := FindConfigs()
	if err != nil {
		return nil, err
	}
	for _, manager := range managers {
		for _, project := range manager.Projects {
			newCmd := &cobra.Command{
				Use:   project.Name,
				Short: fmt.Sprintf("Activate environment %s", project.Name),
				Run: func(cmd *cobra.Command, args []string) {
					if len(args) == 0 {
						// Activation
						if isShellSetup() {
							// Issue shell activation
							print("SHELL ACTIVATION")
						} else {
							print("Setup activation")
						}
					} else {
						print("SUBPROCESS EXEC")
					}
				},
			}
			cmds = append(cmds, newCmd)
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

func init() {
	configCmds, err := LoadConfigCommands()
	if err != nil {
		panic(err)
	}

	for _, cmd := range configCmds {
		rootCmd.AddCommand(cmd)
	}
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(evalCmd)
}

func errExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	err := rootCmd.Execute()
	errExit(err)
}
