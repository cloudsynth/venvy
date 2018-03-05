Venvy 
=====

A fast modular (see `modules/`) sh hook system bound to subcomamnds. Useful for bootstrapping and managing local development and CI environments.

# Getting Started

### Install

For golang:

```
go get -u github.com/pnegahdar/venvy
```

For OSX:

```
curl -SL https://github.com/pnegahdar/venvy/releases/download/0.0.0/darwin_amd64 > /usr/local/bin/venvy && \
    chmod +x /usr/local/bin/venvy
```

For Linux:

```
curl -SL https://github.com/pnegahdar/venvy/releases/download/0.0.0/linux_amd64 > /usr/local/bin/venvy && \
    chmod +x /usr/local/bin/venvy
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


[[modules]]
name = "local-env"
type = "env"

    [modules.config.vars]
    SERVER_SITE="http://127.0.0.1:8000"

[[modules]]
name = "dev-mux"
type = "tmux-window"

	[modules.config]
	name = "server"

	    [[modules.config.panes]]
	    commands = ["venvy acme -- python manage.py runserver"]

	    [[modules.config.panes]]
	    commands = ["venvy acme -- python manage.py shell_plus"]

	    # Scratch local dev
	    [[modules.config.panes]]
	    commands = ["venvy acme"]


[[projects]]
name = "acme"
modules = ["py3", "local-env"]
script_subcommands = ["scirpts/deploy.sh"] # Dirs with scripts or files

[[projects]]
name = "acme-py27"
modules = ["py2", "local-env"]
script_subcommands = ["scirpts/deploy.sh"] # Dirs with scripts or files

[[projects]]
name = "acme-dev"
modules = ["dev-mux"]
```

### Use your virtual environments

#### Config file discovery

venvy searches through your git files and your current directory for a file named `venvy.toml`. 

venvy preserves a history of files it's seen so they can be activated from anywhere. To registry a new file run `venvy` once in a directory containing it.

#### Activate the environment:

```
venvy acme
```

#### Start the tmux environment:

```
venvy acme-dev
```

#### Execute in environment:

```
venvy acme -- python -V
```

#### Execute the deploy.sh script in the environment: 

```
vevny acme.deploy
```

#### Reset the environment:

```
venvy acme --reset
```

#### Create or execute in a temporary environment:

```
venvy acme --temp
# OR e.g. test multiple version of python
venvy acme --temp -- py.test
venvy acme-py27 --temp -- py.test
```

#### Debug the environment:

```
venvy acme --verbose
```


#### Deactivate:

**Note**: venvy always deactivates before activating a new venv so you generally wont need to do this.

```
devenv
```

#### Show paths:

```
venvy acme --print--root
venvy acme.deploy --print-path
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
### EnvVars

**Type**: env

Sets and unsets enviornment variables for the environment. 

```toml
[[modules]]
name = "pypath"
type = "env"

    # Optional:
    [modules.config.vars] # Vars to set
    PYTHONPATH="$(venvy my_project --print-root)"
    TZ="UTC"
    
    [modules.config]
    unset_vars = ["IS_TESTING"] # Vars to unset
```

### Tmux Window

**Type**: tmux-window

Launches a managed tmux window with the panes and commands specified. 

```toml
[[modules]]
name = "tmux"
type = "tmux-windoow"

    [modules.config]
    name = "server" # required, name suffix of the tmux window 
    # Optional
    disable_destroy_existing = false # default false, if set to true venvy wont destory the already existing tmux window
    layout = "tile" # default: tile, see 'man tmux', and grep 'The following layouts are supported' for more info 
    
        # Optional array of panes
        [[modules.config.panes]]
        root = "data" # deault: Project root, can be a rel path to the project root or abspth
        commands = ["venvy acme -- python manage.py runserver"] # A lit of commands to run in a window, exit codes are disregarded, all are run.

        [[modules.config.panes]]
        commands = ["venvy acme -- python manage.py shell_plus"]
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
modules = ["debug", "py3", "..."]
```

## Adding modules:

A module defines the following interface:

```go
type Moduler interface {
	ShellActivateSh() ([]string, error)
	ShellDeactivateSh() ([]string, error)
}
```

To create a new module look at files in the repo named `modules/*.go`. For a very simple one look at `modules/debug.go` for more complex example look at `modules/python.go`. 


FAQ: 

- Why not sub-shell? I prefer this dev UX. If you prefer subshells call `$SHELL` first.
  

### Planned modules:

- Brew packages
- Apt packages
- Tmux window config
- Nvm/Npm (similar to python)
- Golang/deps (similar to python)

   

