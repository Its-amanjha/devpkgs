# Design Spec: Bulk Actions & Multi-Select

## Overview
Add multi-select capability to the package list, allowing users to tag multiple packages and run sequential bulk upgrade/remove actions.

## Requirements

### Selection UX
- `Space` key toggles selection on the current cursor package.
- Selected packages show `[✓]` prefix, unselected show `[ ]`.
- Selection is scoped to the **current tab only** — switching tabs does not affect other tabs' selections.
- Footer displays count: "3 selected" when packages are tagged.

### Triggering Bulk Actions
- When 1+ packages are selected, pressing `u` (upgrade) or `x` (remove) triggers a bulk confirmation dialog: "Upgrade 3 packages using winget?"
- Confirmation options: `Enter` = silent, `y` = with live logs, `Esc/n` = cancel.
- When **no** packages are selected, `u` / `x` operates on the single cursor item (preserving current behavior).

### Sequential Execution
- Packages are upgraded/removed one at a time in list order.
- Footer progress indicator: "Upgrading 2/5: express..."
- If `y` (with logs) was chosen, the logs overlay streams output for each package, resetting `logLines` between packages.
- On completion of all packages, the affected tab refreshes and all selections are cleared.

### Edge Cases
- Up-to-date packages are automatically skipped during bulk upgrade.
- If all selected packages are up-to-date, show a notice in the footer and skip the action.
- Errors on one package do not stop the queue — the next package proceeds and the error is logged in the footer status.

## Technical Approach
- Add `selected map[string]bool` to each `TabState` in `model.go`.
- Add `bulkQueue []string`, `bulkIndex int`, `bulkAction pm.Action`, `bulkLogs bool` to `Model` for queue tracking.
- Modify list item rendering in `panels.go` to show `[✓]`/`[ ]` prefix.
- Handle `Space` key in `update.go` to toggle selection.
- On bulk confirm, populate the queue and trigger the first action.
- On `ActionMsg` completion during bulk mode, advance `bulkIndex` and trigger the next action (or finish and refresh).
