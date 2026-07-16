---
name: devpkgs
description: A warm, spacious, and quiet terminal workspace for developers.
colors:
  primary: "#C9A8FF"
  primary-muted: "#9880E8"
  neutral-text: "#F1EEF8"
  neutral-dim: "#A09AB8"
  neutral-detail: "#D8D3F0"
  selected-bg: "#C9A8FF"
  selected-fg: "#16141F"
  success: "#A8FFEC"
  error: "#FF80C8"
  bg: "#16141F"
typography:
  display:
    fontFamily: "Monospace"
    fontWeight: "bold"
  body:
    fontFamily: "Monospace"
    fontWeight: "normal"
  label:
    fontFamily: "Monospace"
    fontWeight: "bold"
rounded:
  panel: "rounded-box-drawing"
spacing:
  panel-padding: "1-cell"
  doc-padding: "3-cells"
components:
  tab-active:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.selected-fg}"
    padding: "0 2 cells"
  tab-inactive:
    textColor: "{colors.neutral-dim}"
    padding: "0 1 cells"
  panel-border:
    textColor: "{colors.primary}"
  selected-item:
    backgroundColor: "{colors.selected-bg}"
    textColor: "{colors.selected-fg}"
---

# Design System: devpkgs

## 1. Overview

**Creative North Star: "The Monospace Sanctuary"**

`devpkgs` is designed to be a calm, spacious, and quiet terminal workspace. Instead of mimicking crowded command-line interfaces filled with raw text and jagged lines, this layout embraces whitespace, clean typography hierarchy, and soft visual boundaries. Depth is established through structural layering and key highlights rather than artificial gradients or heavy shadows.

**Key Characteristics:**
- Spacious cell padding (`1-cell` minimum padding inside boxes).
- Gentle rounded terminal container borders (`╭─ ─╮` Unicode box characters).
- Color accents used sparingly (only for state, indicators, and headers).

## 2. Colors

Colors in this system avoid typical saturated neon palettes or default white/black terminal themes. They are selected for optimal contrast and a soft, modern pastel feel.

### Primary
- **Calm Pastel Lavender** (#C9A8FF): The primary color for borders, titles, active badges, and highlights.
- **Muted Lavender** (#9880E8): A darker variation of the primary accent used for inactive borders or subtle highlights.

### Neutral
- **Velvet Obsidian** (#16141F): The primary background color.
- **Soft Milk White** (#F1EEF8): Standard body text color.
- **Warm Slate** (#A09AB8): Subdued text color for secondary labels, inactive tabs, and footer shortcuts.
- **Powder Lavender** (#D8D3F0): Mid-tone text color used for details and descriptions.

### Status
- **Mint Mint** (#A8FFEC): Indicating success or updated states.
- **Soft Rose** (#FF80C8): Indicating errors, deleted states, or critical warnings.

**The Ten Percent Rule.** Accent colors (Lavender, Mint, Rose) must occupy no more than 10% of the screen. Their scarcity ensures they effectively draw the user's attention.

## 3. Typography

**Display Font:** Terminal Monospace (Bold)
**Body Font:** Terminal Monospace (Regular)
**Label/Mono Font:** Terminal Monospace (Bold)

Typography is clean and relies heavily on weight differences and casing rather than mixing font families.

### Hierarchy
- **Display** (Bold, uppercase): Used for dashboard headers and tab labels.
- **Headline** (Bold): Used for panel titles.
- **Body** (Regular): Used for package details and descriptions. Must wrap text cleanly without clipping.
- **Label** (Bold): Used for inline metadata keys (e.g. `Installed:`, `Latest:`).

## 4. Elevation

As a terminal UI, dropshadows do not exist. Visual depth is established entirely through **Tonal Layering** and overlapping components.

### Shadow Vocabulary
- **Flat-by-Default**: All panels are flat at rest. Background depth is achieved by using borders.
- **Modals & Overlays**: Active modals (e.g., confirmations and theme overlays) are drawn centered on the screen, overlaying panels with highlighted borders.

**The Border Priority Rule.** High-priority or focused containers use a bright primary border (`#C9A8FF`), while secondary or inactive containers use a muted border (`#9880E8`).

## 5. Components

### Tabs
- **Shape**: Plain uppercase text block.
- **Active State**: Primary background (`#C9A8FF`) and selected foreground (`#16141F`) with `2-cell` padding.
- **Inactive State**: Soft dim text color (`#A09AB8`) with `1-cell` padding.

### Panel Containers
- **Borders**: Drawn using Unicode rounded characters (`╭`, `╮`, `╰`, `╯`, `─`, `│`).
- **Padding**: 1-cell horizontal padding between content and borders.

### Search Bar
- **Borders**: Highlighted in primary lavender when active/focused; falls back to muted border when inactive.
- **Cursor**: A solid block (`█`) indicates active text entry.

### Modals
- **Position**: Horizontally and vertically centered.
- **Border**: High-contrast primary border to pop out from background panels.

## 6. Do's and Don'ts

### Do:
- **Do** wrap descriptions to respect panel widths.
- **Do** use `Mint Mint` (`#A8FFEC`) to show matching versions and `Soft Rose` (`#FF80C8`) for outdated/warning flags.
- **Do** align columns cleanly using monospace cell tabs.

### Don't:
- **Don't** use sharp box characters (`┌`, `┐`, `└`, `┘`) for borders; always use rounded ones (`╭`, `╮`, `╰`, `╯`).
- **Don't** overflow text boundaries; clamp or wrap long strings.
- **Don't** mix multiple accent colors in a single line. Keep it to one clean color focus.
