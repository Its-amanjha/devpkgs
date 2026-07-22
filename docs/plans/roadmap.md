# Project Roadmap & Architecture Goals

This document outlines the architectural roadmap and development milestones for `devpkgs`.

---

## Completed Core Features

### 1. Multi-Manager Unified Dashboard
- Support for **Homebrew**, **WinGet**, **NPM**, and **Pip**.
- Fast startup initialization using native Go directory scanning and streaming CLI parsers.
- Unified "All Packages" view with origin badges.

### 2. Multi-Select & Bulk Queue Operations
- Batch package operations via `Space` key selection.
- Sequential execution queue with real-time log tracking.
- Automatic up-to-date checking to prevent redundant upgrades.

### 3. Live Logs Streaming Overlay
- Real-time `stdout`/`stderr` line-by-line output overlay.
- Scrollable modal viewer with viewport clamping and keyboard navigation.

### 4. Online Registry Search & Installation
- Dedicated Search tab querying online package registries (npm, PyPI, WinGet, Homebrew).
- Direct installation flow with live output progress.

### 5. Dynamic Theme Engine
- Built-in theme picker with pre-calibrated color schemes (Catppuccin Mocha, Gruvbox, Nord, Dracula, etc.).

---

## Future Development Milestones

### Phase 1: Expanded Platform Support
- **Debian / Ubuntu (`apt`) integration**: Native parsing for Debian system packages.
- **Arch Linux (`pacman`) integration**: Support for Arch system packages.

### Phase 2: Configuration & Persistence
- User configuration file (`~/.config/devpkgs/config.yaml`) for setting custom default themes and startup options.
- Custom package manager alias mappings.

### Phase 3: Export & Import Profiles
- Export installed package list to a portable manifest.
- One-click environment bootstrap on a new machine.
