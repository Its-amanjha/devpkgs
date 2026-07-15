# pkgview

A terminal dashboard for packages already installed through Homebrew, npm, and pip.

```bash
go run .
```

Use the arrow keys to navigate, `/` to search, `t` to change theme, and `q` to quit. pkgview reads installed package lists and retrieves public metadata from Homebrew, npm, and PyPI; it does not modify packages.

## Development

```bash
go build ./...
```

The repository carries the upstream MIT licence notice in [LICENSE](LICENSE). New work in this repository is maintained independently.
