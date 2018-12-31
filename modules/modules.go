package modules

import "github.com/pnegahdar/venvy/manager"

var DefaultModuleMakers = venvy.ModuleMakerTypeMap{
	"python":      NewPythonModule,
	"jump":        NewJumpModule,
	"ps1":         NewPS1Module,
	"debug":       NewDebugModule,
	"exec":        NewExecModule,
	"env":         NewEnvVarModule,
	"tmux-window": NewTmuxModule,
}
