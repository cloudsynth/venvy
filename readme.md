Venvy 
=====

A fast unixy modular sh hook system bound to subcomamnds. Useful for bootstrapping and managing local development and CI environments.

# Getting Started

### Install

For golang:

```
go install -u github.com/pnegahdar/venvy
```

### Create a config

Create a `venvy.toml` in your project root.

```toml
[[modules]]
name = "py3"
type = "python"
	[modules.config]
	python = "python3.6"
	dependencies = ['requirements.txt']
	auto_install_deps = true

[[modules]]
name = "py2"
type = "python"
	[modules.config]
	python = "python2.7"
	dependencies = ['requirements.txt']
	auto_install_deps = true

[[projects]]
name = "my_project"
modules = ["py3"]

[[projects]]
name = "my_project27" 
modules = ["py2"]
```

### Use your virtual environments

#### Config file discovery

venvy searches through your git files and your current directory for a file named `venvy.toml`. 

venvy preserves a history of files it's seen so they can be activated from anywhere. To registry a new file run `venvy` once in a directory containing it.

#### Activate the environment:

```
venvy my_project
```

#### Execute in environment:

```
venvy my_project -- python -V
```

#### Reset the environment:

```
venvy --reset my_project
```

#### Create a temporary environment:

```
venvy --temp my_project
# OR e.g. test multiple version of python
venvy --temp my_project -- py.test
venvy --temp my_project27 -- py.test
```

#### Debug the environment:

```
venvy --debug my_project
```


#### Deactivate:

**Note**: venvy always deactivates before activating a new venv so you generally wont need to do this.

```
devenv
```

For teams you are encouraged to put developer specific configs in a shared file so teammates can improve their development environments. 

# Modules

### Python

**Type**: python

The python module manages virtualenvs and pip installs for you. 

Full config:

```toml
[[modules]]
name = "py3"
type = "python"
    
    # Optional:
	[modules.config]
	python = "python3.6" # Default: python, the python to use for the virtualenv
	dependencies = ["Cython==0.27.3", "requirements.txt"] # Default: [], Files and named dependencies supported 
	auto_install_deps = false # Default: false, Whether to install changing deps on activation
	pip_install_command = "pip install" # Default: "pip install", the command to use for the install 
	prep_pip_install_command = "pip install" # Default: "pip install", the command to use to upgrade pip/virtualenv
	virtualenv_command = "virtualenv" # Default: "vritualenv", the command to use to build the virtualenv
```


### Exec

**Type**: exec

Executes a series of commands on activation or on deactivation of the environment.

```toml
[[modules]]
name = "greet"
type = "exec"

    # Optional:
    [modules.config]
    activation_commands = ["echo Welcome"]
    deactivation_commands = ["echo Goodbye"]
```


### Jump

**Type**: jump
**Note**: Included in all projects unless `Project.disable_builtin_modules = true`.

`cd`s into the `Project.Root` or the directory of the `venvy.toml` and returns to the directory before on deactivation.

```toml
[[modules]]
name = "jump"
type = "jump"

    # Optional:
    [modules.config]
    to_dir = "/var/lib/postgres" # Default: Project.Root or path of the venvy.toml, the path to cd to on activation
    disable_jump_back = false # Default: false, disable return to Cwd on deactivation
```

### PS1 (prompt)

**Type**: ps1
**Note**: Included in all projects unless `Project.disable_builtin_modules = true`.

Sets a PS1 (prompt) prefix so you know what environment you're working in.

```toml
[[modules]]
name = "ps"
type = "ps1"

    # Optional:
    [modules.config]
    value = "prefix ->" # Default: colorized Project.Name, the prefix value of the PS1
```

### Debug

**Type**: debug

Enables sh debug on the venvy run scripts (i.e `set -x` and `set +x`).

```toml
[[modules]]
name = "debug"
type = "debug"
```


**Note**: Add this first on the Project.modules list to make sure it is enabled first and disabled last.

```toml
[[project]]
name = "broken_activation"
modules = ["debug", "py3", ...]
```

## Adding modules:

A module defines the following interface:

```go
type Moduler interface {
	ShellActivateSh() ([]string, error)
	ShellDeactivateSh() ([]string, error)
}
```

To create a new module look at files in the repo named `module_*.go`. For a very simple one look at `module_debug.go` for more complex example look at `module_python.go`. 


FAQ: 

- Why not sub-shell? I prefer this dev UX. If you prefer subshells call `$SHELL` first.
  

### Planned modules:

- Brew packages
- Apt packages
- Tmux window config
- Nvm/Npm (similar to python)
- Golang/deps (similar to python)

   

