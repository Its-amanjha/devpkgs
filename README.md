# devpkgs

A terminal dashboard for packages already installed through Homebrew, npm, pip, and Windows Package Manager (`winget`).

```bash
go run .
```

Use the arrow keys to navigate, `/` to search, `o` for outdated packages, `r` to refresh, `u` to upgrade, `x` to remove, `t` to change theme, and `q` to quit. Upgrade and removal require confirmation. devpkgs reads installed package lists and retrieves public metadata from Homebrew, npm, and PyPI.

Published builds install the `devpkgs` command.

On Windows, run `./install.ps1` from PowerShell after installing Go. The winget tab lists packages recognised by WinGet and supports confirmed upgrade/removal actions.

## Development

```bash
go build ./...
```

The repository carries the upstream MIT licence notice in [LICENSE](LICENSE). New work in this repository is maintained independently.
