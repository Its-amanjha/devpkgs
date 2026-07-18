# devpkgs

A beautiful, high-performance, and lightweight terminal dashboard to manage all packages installed on your developer machine across **Homebrew, Windows Package Manager (`winget`), NPM, and Pip**.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and styled with [Lipgloss](https://github.com/charmbracelet/lipgloss).

-----

## Features

* **Unified Local Dashboard**: Instantly view installed packages and versions from multiple managers side-by-side.
* **Instant Startup Loading**: Native Go size calculation, optimized `winget list` streaming, and fully offline Homebrew metadata loading.
* **Multi-Select & Bulk Operations**: Highlight multiple packages using `Space` and run upgrades or removals sequentially in a single confirmation.
* **Interactive Live Logs Panel**: Watch package installations/updates stream line-by-line in a scrollable, real-time terminal modal.
* **Online Registry Search & Install**: Search npm, PyPI, WinGet, and Homebrew online registries from a unified Search tab, and download packages instantly.
* **Custom Color Themes**: Switch between premium color schemes (e.g., Catppuccin Mocha, Gruvbox Dark, Tokyo Night) dynamically using a built-in theme picker modal.

-----

## Keybindings & Navigation

| Key | Action | Context |
| :--- | :--- | :--- |
| **`←` / `→`** | Switch package manager tabs / Search tab | Dashboard |
| **`↑` / `↓`** | Navigate package lists / search results | Dashboard |
| **`/`** | Focus search input bar (clears query) | Dashboard |
| **`Esc` / `Enter`** | Unfocus search input bar to navigate results | Search Input |
| **`Space`** | Toggle package selection (for bulk actions) | Local tabs |
| **`i`** | Install highlighted package | Search Results |
| **`u`** | Bulk upgrade selected packages (or current highlighted package) | Local tabs |
| **`x`** | Bulk remove/uninstall selected packages (or current highlighted package) | Local tabs |
| **`o`** | Toggle "Outdated Only" packages filter | Local tabs |
| **`r`** | Refresh installed packages list | Local tabs |
| **`l`** | Toggle live logs panel overlay (view active/past logs) | Dashboard |
| **`t`** | Open dynamic Theme selector modal | Dashboard |
| **`q` / `Ctrl+C`** | Quit application | Any |

---

## Installation & Setup

### Prerequisites
* Go 1.25.5 or later installed on your path.
* Access to package managers (`brew`, `winget`, `npm`, `pip`) on your local machine.

### Installation

#### Windows
Run the PowerShell setup script:
```powershell
./install.ps1
```

#### macOS / Linux
Run the shell setup script:
```bash
./install.sh
```

---

## Development

Build the project locally:
```bash
go build ./...
```

Run tests:
```bash
go test ./...
```

Run dev console:
```bash
go run .
```

---

## License
MIT License. Created by Aman Kumar Jha. Independent maintenance and contributions are welcome.
