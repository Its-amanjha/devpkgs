# Feature Overview & Design Specs

Detailed specifications for core functionality implemented in `devpkgs`.

---

## 1. Local Package Managers Dashboard

- **Unified Navigation**: Switch between tabs using `←` / `→` or number shortcuts.
- **Instant Search & Filtering**: Use `/` to focus the search bar and filter package lists instantly by name or manager.
- **Outdated Filter**: Toggle `o` to filter and display only packages with pending updates.

## 2. Bulk Execution Engine

- **Multi-Select**: Toggle package selection across lists using `Space`.
- **Sequential Execution**: Bulk upgrades (`u`) and removals (`x`) run sequentially to avoid process locking and standard input conflicts.
- **Error Tracking**: Tracks failed operations and reports overall queue status upon completion.

## 3. Real-Time Streaming Terminal Modal

- **Subprocess Piping**: Captures live output from underlying CLI commands line-by-line.
- **Log Modal Overlay**: Press `l` to toggle the log viewer at any time during or after an execution.

## 4. Theme System

- **Dynamic Theme Selection**: Press `t` to open the interactive theme picker.
- **High Contrast Palettes**: Pre-tested themes designed for terminal readability and accessibility.
