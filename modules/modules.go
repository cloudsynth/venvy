package modules

import "github.com/pnegahdar/venvy/venvy"

var DefaultModuleMakers = venvy.ModuleMakerTypeMap{
	"python": NewPythonModule,
	"jump":   NewJumpModule,
	"ps1":    NewPS1Module,
	"debug":  NewDebugModule,
	"exec":   NewExecModule,
	"env":    NewEnvVarModule,
}
