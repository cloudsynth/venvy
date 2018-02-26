Venvy (WIP)
=====

A unixy modular shell hook system bound to subcomamnds. Useful for bootstrapping and managing local development and CI environments.


```toml
[[module]]
name = "env"
type = "env_inject"
	[modules.config.vars]
	TZ = "UTC"

[[module]]
name = "py3"
type = "python"
	[module.config]
	python = "python3.6"
	deps = ['requirements.txt', "ipython"]
	auto_install_deps = true


[[module]]
name = "tmux"
type = "tmux"
	[module.config]
	panes = [
	    "python manage.py runserver",
	    "celery -A app.celery worker"
	]

# Parham's custom tmux setup	
[[module]]
name = "tmux_parham"
	[module.config]
	panes = [
	    "python manage.py runserver",
	    "celery -A app.celery worker",
	    "celery -A app.celery broker",
	]



[[project]]
name = "acme_dev"
autojump = true
subprocess_modules = ["env", "py3"]
activation_modules = ["env","py3", "tmux"]
scripts_modules = ["env", "py3"]
script_globs = ["app/scripts/*"]

[[project]]
name = "acme_ci"
subprocess_modules = ["env", "py3"]

[[project]]
name = "acme_parham"
acitvation_modules = ["py2", "tmux_parham"]

```


To start working locally:

```
venvy acme_parham
```


Note: for teams you are encouraged to put developer specific configs in a shared file so teammates can improve their development environments. 

### How it works:

There are two modes of operations: **subprocess** and **interactive shell**.

A module defines the following interface:

```go
type Moduler interface {
	ShellActivateSh() (string, error)
	ShellDeactivateSh() (string, error)
}
```

Modules perform checks (golang) to generate their sh scripts.

When running a command like (subprocess):

```
venvy acme_ci -- python manage.py migrate
```

venvy:
1) Merges the `subprocess_module` scripts
2) Executes the scripts in order of the config
3) Runs the command you passed
4) Runs the deactivation commands backwards.
 
When using venvy to modify your current shell session (interactive shell):

```
venvy acme_parham
```

venvy:

1) Uses the hijacked `venvy` command from the `venvy shell-init` placed in the rc file
2) Creates a tempfile and passes it to nested `venvy` call
3) `venvy` Merges the `activation_modules` scripts
4) writes the scripts in order of the config to the tempfile
5) source the temp file in the current session
6) Add a `deactivate` command to your session to end it
 

FAQ: 

- Why not sub-shell? I prefer this dev UX. If you prefer subshells call `$SHELL` first.
  

### Planned hooks:

- Tmux
- Nvm/Npm
- Golang/deps

   

