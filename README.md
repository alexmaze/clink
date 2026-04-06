# clink Configuration Distribution Tool

`clink` is a configuration distribution tool for Linux and macOS. It uses a single `config.yaml` to describe local symlinks, local copies, and remote SSH uploads, and provides a unified workflow for execution, validation, restoration, and management around that configuration.

This version has been fully refactored with the following product constraints:

- Linux and macOS only
- Unified subcommand-based CLI
- All runtime output and interaction are in English
- Incompatible with legacy backup formats and old usage patterns

## Installation

```sh
go install github.com/alexmaze/clink@latest
```

Or use:

```sh
make build
```

## Command Overview

```sh
clink apply   -c <config.yaml>
clink check   -c <config.yaml>
clink restore -c <config.snapshot.yaml>
clink add     -c <config.yaml> <source>
clink version
```

Common flags:

- `-c, --config`: specify the config file path
- `-r, --rule`: filter rules by name or 1-based index; can be repeated
- `-d, --dry-run`: generate the plan only without executing it
- `-y, --yes`: skip confirmation
- `--non-interactive`: disable interactive input
- `--output text|json`: choose text or JSON output

## Configuration File

Example:

```yaml
mode: symlink

hooks:
  pre: echo "start apply"
  post: echo "finish apply"

ssh_servers:
  prod:
    host: 192.168.1.10
    port: 22
    user: root
    key: ~/.ssh/id_rsa

vars:
  APP_HOME: /opt/myapp

rules:
  - name: shell
    items:
      - src: ./dotfiles/.zshrc
        dest: ~/.zshrc

  - name: app-config
    mode: copy
    items:
      - src: ./app/config.yaml
        dest: ${APP_HOME}/config.yaml

  - name: remote-nginx
    mode: ssh
    ssh: prod
    items:
      - src: ./nginx/nginx.conf
        dest: /etc/nginx/nginx.conf
```

Notes:

- Top-level `mode` is the default distribution mode, and defaults to `symlink`
- `rules[].mode` can override the top-level default
- Local `src` supports relative paths, absolute paths, and `~/`
- Local `dest` supports relative paths, absolute paths, and `~/`
- Under SSH mode, `dest` must be an absolute path on the remote host
- Variable syntax is `${VAR_NAME}`
- If neither `key` nor `password` is provided in `ssh_servers`, the CLI prompts for a password interactively at runtime

## apply

`apply` generates an execution plan from the config, backs up the current target state first, and then performs the distribution.

```sh
clink apply -c ./config.yaml
clink apply -c ./config.yaml -r shell
clink apply -c ./config.yaml -d
clink apply -c ./config.yaml --output json
```

Behavior:

- Local modes back up the original target to `~/.clink/<timestamp>/payload/...`
- Remote mode downloads the original remote file into the same backup directory first
- Each run writes:
  - `config.snapshot.yaml`
  - `manifest.json`
  - `payload/`

## check

`check` validates the current state against the config.

```sh
clink check -c ./config.yaml
clink check -c ./config.yaml -r 1
```

Validation semantics:

- `symlink`: verifies that the target is a symbolic link and points to the expected source path
- `copy`: verifies that the target exists and its content hash matches the source
- `ssh`: verifies that the remote target exists and its file type matches the local source

The command exits with a non-zero code if any item fails.

## restore

`restore` restores from `manifest.json` in a backup directory created by the new version.

```sh
clink restore
clink restore --backup 20260406_120000
clink restore --backup ~/.clink/20260406_120000 -d
```

Behavior:

- In interactive mode, omitting `--backup` opens a backup selection list
- In non-interactive mode, omitting `--backup` selects the latest backup by default
- Local restore uses a temporary path and then atomically replaces the target
- SSH restore reuses connections grouped by server
- Backup directories from older versions are not supported; backups without `manifest.json` cannot be restored

## add

`add` brings existing files under `clink` management.

```sh
clink add -c ./config.yaml ~/.vimrc
clink add -c ./config.yaml --name shell --mode symlink ~/.zshrc
clink add -c ./config.yaml --rule shell --dest ~/.bashrc ./local/.bashrc
```

Behavior:

- External source files are copied to `<config-dir>/.clink/sources/<rule-slug>/`
- The default mode for a newly added rule is `symlink`
- You can only append to local rules, not SSH rules
- `dry-run` shows the plan only and does not modify files or config

## Backup Layout

Example:

```text
~/.clink/
  20260406_120000/
    config.snapshot.yaml
    manifest.json
    payload/
      shell/
        home/alex/.zshrc
      remote-nginx/
        etc/nginx/nginx.conf
```

`manifest.json` explicitly records the following for each backup entry:

- rule name
- mode
- source
- destination
- backup file path
- SSH server
- path kind
- backup content hash

## Development

Common commands:

```sh
make build
make test
make build-all
```

Notes:

- `make build-all` only builds Linux and macOS artifacts
- Windows support is out of scope for the current project
- Runtime output remains in English; keep new code comments and docs aligned with the repository conventions

## Compatibility Notes

This version is not compatible with older versions:

- Old command entrypoints are not supported
- Old backup directory layouts are not supported
- Historical interaction patterns are not supported

When upgrading, generate new backups and use the new command structure.
