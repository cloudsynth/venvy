package modules

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/pnegahdar/venvy/manager"
	"github.com/pnegahdar/venvy/util"
	"strconv"
	"strings"
)

type PS1Config struct {
	Value string
	ZshValue string
	BashValue string
}

type PS1Module struct {
	manager *venvy.ProjectManager
	config  *PS1Config
}

const escape = "\x1b"
const noPrintStart = "__noprintstart__"
const noPrintEnd = "__noprintend__"

func replaceNoPrint(data, startSeq, endSeq string) string{
	replaceStart := strings.ReplaceAll(data, noPrintStart, startSeq)
	return strings.ReplaceAll(replaceStart, noPrintEnd, endSeq)

}

func colorString(data string, attrs ...color.Attribute) string {
	colorSeq := func(attrs ...color.Attribute) string {
		format := make([]string, len(attrs))
		for i, v := range attrs {
			format[i] = strconv.Itoa(int(v))
		}
		return strings.Join(format, ";")
	}

	// These hide non printable strings from the prompt length so wrapping doesn't break

	noLengthEscape := func(d string) string {
		return noPrintStart + d + noPrintEnd
	}

	colorPrefix := fmt.Sprintf("%s[%sm", escape, colorSeq(attrs...))
	colorSuffix := fmt.Sprintf("%s[%sm", escape, colorSeq(color.Reset))
	return noLengthEscape(colorPrefix) + data + noLengthEscape(colorSuffix)
}

func (ps *PS1Module) ShellActivateCommands() ([]string, error) {
	return []string{
		`export OLD_PS1="$PS1"`,
		fmt.Sprintf(`GENERIC_PS1="%s"`, ps.config.Value),
		fmt.Sprintf(`BASH_PS1="%s"`, ps.config.BashValue),
		fmt.Sprintf(`ZSH_PS1="%s"`, ps.config.ZshValue),
		fmt.Sprintf(`if [ -n "$ZSH_VERSION" ]; then export PS1="$ZSH_PS1 $PS1"; elif [ -n "$BASH_VERSION" ]; then export PS1="$BASH_PS1 $PS1"; else export PS1="$GENERIC_PS1 $PS1";  fi`),
	}, nil
}

func (ps *PS1Module) ShellDeactivateCommands() ([]string, error) {
	return []string{
		`if [ ! -z "$OLD_PS1" ]; then export PS1="$OLD_PS1"; unset OLD_PS1; fi`,
	}, nil
}

func NewPS1Module(manager *venvy.ProjectManager, self *venvy.Module) (venvy.Moduler, error) {
	moduleConfig := &PS1Config{}
	err := util.UnmarshalEmpty(self.Config, moduleConfig)
	if err != nil {
		return nil, err
	}
	// Set default behavior
	if moduleConfig.Value == "" {
		parts := []string{
			colorString("[", color.FgHiCyan),
			colorString(venvy.ProjectName, color.FgHiBlue),
			colorString(":", color.FgHiGreen),
			colorString(manager.Project.Name, color.FgHiMagenta),
			colorString("]", color.FgHiCyan),
		}
		colored := strings.Join(parts, "")
		moduleConfig.Value = replaceNoPrint(colored, "", "")
		moduleConfig.BashValue = replaceNoPrint(colored, "\\[", "\\]")
		moduleConfig.ZshValue = replaceNoPrint(colored, "%{", "%}")
	}
	if moduleConfig.BashValue == ""{
		moduleConfig.BashValue = moduleConfig.Value
	}
	if moduleConfig.ZshValue == ""{
		moduleConfig.ZshValue = moduleConfig.Value
	}
	return &PS1Module{manager: manager, config: moduleConfig}, nil
}
