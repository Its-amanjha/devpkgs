package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderLeftPanel(width int, boxHeight int) string {
	if m.allMode {
		return m.renderAllLeftPanel(width, boxHeight)
	}
	st := m.states[m.activeTab]
	visibleHeight := boxHeight - 2
	innerWidth := width - 4

	start := 0
	if st.cursor >= visibleHeight {
		start = st.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(st.displayPackages) {
		end = len(st.displayPackages)
	}

	var listItems []string
	for i := start; i < end; i++ {
		pkg := st.displayPackages[i]
		if i == st.cursor {
			style := SelectedItemStyle.Width(innerWidth)
			listItems = append(listItems, style.Render(pkg))
		} else {
			listItems = append(listItems, ItemStyle.Render(pkg))
		}
	}

	boxTitle := fmt.Sprintf("Packages (%d)", len(st.displayPackages))
	return renderPaneBox(width, boxHeight, boxTitle, strings.Join(listItems, "\n"))
}

func (m Model) renderAllLeftPanel(width int, boxHeight int) string {
	visibleHeight := boxHeight - 2

	start := 0
	if m.allCursor >= visibleHeight {
		start = m.allCursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(m.allDisplayPackages) {
		end = len(m.allDisplayPackages)
	}

	typeWidth := 5

	pmBadge := func(name string) lipgloss.Style {
		switch name {
		case "brew":
			return lipgloss.NewStyle().
				Width(typeWidth).
				Background(currentTheme.Primary).
				Foreground(currentTheme.SelectedFg)
		case "npm":
			return lipgloss.NewStyle().
				Width(typeWidth).
				Background(darkenHex(string(currentTheme.Primary), 0.65)).
				Foreground(currentTheme.Text)
		case "pip":
			return lipgloss.NewStyle().
				Width(typeWidth).
				Background(darkenHex(string(currentTheme.Success), 0.65)).
				Foreground(currentTheme.Text)
		default:
			return lipgloss.NewStyle().
				Width(typeWidth).
				Background(currentTheme.Muted).
				Foreground(currentTheme.Text)
		}
	}

	var listItems []string
	for i := start; i < end; i++ {
		pkg := m.allDisplayPackages[i]
		origin := m.allPackageOrigin[pkg]

		originRendered := pmBadge(origin).Render(origin)
		var pkgRendered string
		if i == m.allCursor {
			pkgRendered = SelectedItemStyle.Render(pkg)
		} else {
			pkgRendered = ItemStyle.Render(pkg)
		}

		listItems = append(listItems, originRendered+" "+pkgRendered)
	}

	boxTitle := fmt.Sprintf("PM  ·  All Packages (%d)", len(m.allDisplayPackages))
	return renderPaneBox(width, boxHeight, boxTitle, strings.Join(listItems, "\n"))
}

func (m Model) renderRightPanel(width int, height int) string {
	tabIdx, pkgName, ok := m.selectedPackage()
	if !ok {
		return renderPaneBox(width, height, "Details",
			lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("No packages match your query"))
	}

	st := m.states[tabIdx]
	if st.Brew != nil {
		return m.renderBrewDetail(width, height, pkgName, st)
	}
	if m.tabs[tabIdx].Name() == "npm" {
		return m.renderNpmDetail(width, height, pkgName, st)
	}
	if m.tabs[tabIdx].Name() == "pip" {
		return m.renderPipDetail(width, height, pkgName, st)
	}
	if m.tabs[tabIdx].Name() == "winget" {
		return m.renderWingetDetail(width, height, pkgName, st)
	}
	return renderPaneBox(width, height, "Details",
		lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.DetailText).Render("Details coming soon for this package manager"))
}

func (m Model) renderBrewDetail(width int, height int, pkgName string, st TabState) string {
	if st.Brew == nil {
		return ""
	}

	var contentLines []string
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
	contentLines = append(contentLines, "")

	if st.Brew.APIErr != nil {
		contentLines = append(contentLines, DetailValueStyle.Render("  Formula data unavailable"))
	} else if st.Brew.Info != nil {
		info := st.Brew.Info
		if info.Desc != "" {
			contentLines = append(contentLines, renderSection(width, "Description", info.Desc))
			contentLines = append(contentLines, "")
		}

		type sectionData struct {
			title string
			lines []string
		}
		var sections []sectionData
		var allWidths []int

		var pkgPairs [][2]string
		if ver, ok := st.Brew.InstalledVersions[pkgName]; ok {
			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
		}
		if info.Versions.Stable != "" {
			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Versions.Stable})
		}
		if path, ok := st.Brew.InstallPaths[pkgName]; ok && path != "" {
			pkgPairs = append(pkgPairs, [2]string{"Path", path})
		}
		if len(pkgPairs) > 0 {
			maxLabel := 0
			for _, p := range pkgPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range pkgPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				value := DetailValueStyle.Render(p[1])
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Package", lines})
		}

		var metaPairs [][2]string
		if info.License != "" {
			metaPairs = append(metaPairs, [2]string{"License", info.License})
		}
		if info.Homepage != "" {
			metaPairs = append(metaPairs, [2]string{"Homepage", info.Homepage})
		}
		if st.Brew.Sizes != nil {
			if size, ok := st.Brew.Sizes[pkgName]; ok && size > 0 {
				metaPairs = append(metaPairs, [2]string{"Size", humanSize(size)})
			}
		}
		if len(metaPairs) > 0 {
			maxLabel := 0
			for _, p := range metaPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range metaPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				var value string
				if p[0] == "Homepage" {
					value = LinkStyle.Render(p[1])
				} else {
					value = DetailValueStyle.Render(p[1])
				}
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Metadata", lines})
		}

		if len(info.Dependencies) > 0 {
			line := DetailValueStyle.Render(strings.Join(info.Dependencies, ", "))
			allWidths = append(allWidths, lipgloss.Width(line))
			sections = append(sections, sectionData{"Dependencies", []string{line}})
		}

		sectionWidth := width
		if len(allWidths) > 0 {
			maxW := 0
			for _, w := range allWidths {
				if w > maxW {
					maxW = w
				}
			}
			sectionWidth = min(width, max(maxW+4, 6))
		}

		for _, s := range sections {
			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  No formula data available"))
	}
	contentLines = append(contentLines, "")
	return renderPaneBox(width, height, "Details", strings.Join(contentLines, "\n"))
}

func (m Model) renderNpmDetail(width int, height int, pkgName string, st TabState) string {
	var contentLines []string
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
	contentLines = append(contentLines, "")

	if !st.NpmDetailsReady {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading registry data..."))
	} else if info, ok := st.NpmDetails[pkgName]; ok {
		if info.Description != "" {
			contentLines = append(contentLines, renderSection(width, "Description", info.Description))
			contentLines = append(contentLines, "")
		}

		type sectionData struct {
			title string
			lines []string
		}
		var sections []sectionData
		var allWidths []int

		var pkgPairs [][2]string
		if ver, ok := st.versions[pkgName]; ok {
			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
		}
		if info.Version != "" {
			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Version})
		}
		if len(pkgPairs) > 0 {
			maxLabel := 0
			for _, p := range pkgPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range pkgPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				value := DetailValueStyle.Render(p[1])
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Package", lines})
		}

		var metaPairs [][2]string
		if info.License != "" {
			metaPairs = append(metaPairs, [2]string{"License", info.License})
		}
		if info.Homepage != "" {
			metaPairs = append(metaPairs, [2]string{"Homepage", info.Homepage})
		}
		if info.Dist != nil && info.Dist.UnpackedSize > 0 {
			metaPairs = append(metaPairs, [2]string{"Size", humanSize(info.Dist.UnpackedSize)})
		}
		if len(metaPairs) > 0 {
			maxLabel := 0
			for _, p := range metaPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range metaPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				var value string
				if p[0] == "Homepage" {
					value = LinkStyle.Render(p[1])
				} else {
					value = DetailValueStyle.Render(p[1])
				}
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Metadata", lines})
		}

		sectionWidth := width
		if len(allWidths) > 0 {
			maxW := 0
			for _, w := range allWidths {
				if w > maxW {
					maxW = w
				}
			}
			sectionWidth = min(width, max(maxW+4, 6))
		}

		for _, s := range sections {
			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
	}
	contentLines = append(contentLines, "")
	return renderPaneBox(width, height, "Details", strings.Join(contentLines, "\n"))
}

func (m Model) renderPipDetail(width int, height int, pkgName string, st TabState) string {
	var contentLines []string
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
	contentLines = append(contentLines, "")

	if !st.PipDetailsReady {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading registry data..."))
	} else if info, ok := st.PipDetails[pkgName]; ok {
		if info.Summary != "" {
			contentLines = append(contentLines, renderSection(width, "Description", info.Summary))
			contentLines = append(contentLines, "")
		}

		type sectionData struct {
			title string
			lines []string
		}
		var sections []sectionData
		var allWidths []int

		var pkgPairs [][2]string
		if ver, ok := st.versions[pkgName]; ok {
			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
		}
		if info.Version != "" {
			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Version})
		}
		if len(pkgPairs) > 0 {
			maxLabel := 0
			for _, p := range pkgPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range pkgPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				value := DetailValueStyle.Render(p[1])
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Package", lines})
		}

		var metaPairs [][2]string
		if info.License != "" {
			metaPairs = append(metaPairs, [2]string{"License", info.License})
		}
		if info.HomePage != "" {
			metaPairs = append(metaPairs, [2]string{"Homepage", info.HomePage})
		}
		if info.Author != "" {
			metaPairs = append(metaPairs, [2]string{"Author", info.Author})
		}
		if len(metaPairs) > 0 {
			maxLabel := 0
			for _, p := range metaPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range metaPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				var value string
				if p[0] == "Homepage" {
					value = LinkStyle.Render(p[1])
				} else {
					value = DetailValueStyle.Render(p[1])
				}
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Metadata", lines})
		}

		sectionWidth := width
		if len(allWidths) > 0 {
			maxW := 0
			for _, w := range allWidths {
				if w > maxW {
					maxW = w
				}
			}
			sectionWidth = min(width, max(maxW+4, 6))
		}

		for _, s := range sections {
			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
	}
	contentLines = append(contentLines, "")
	return renderPaneBox(width, height, "Details", strings.Join(contentLines, "\n"))
}

func (m Model) renderWingetDetail(width int, height int, pkgName string, st TabState) string {
	var contentLines []string
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, DetailTitleStyle.Render("📦 "+pkgName))
	contentLines = append(contentLines, "")

	if st.WingetDetails == nil {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading registry data..."))
	} else if info, ok := st.WingetDetails[pkgName]; ok {
		if info.Description != "" {
			contentLines = append(contentLines, renderSection(width, "Description", info.Description))
			contentLines = append(contentLines, "")
		}

		type sectionData struct {
			title string
			lines []string
		}
		var sections []sectionData
		var allWidths []int

		var pkgPairs [][2]string
		if ver, ok := st.versions[pkgName]; ok {
			pkgPairs = append(pkgPairs, [2]string{"Installed", ver})
		}
		if info.Version != "" {
			pkgPairs = append(pkgPairs, [2]string{"Latest", info.Version})
		}
		if len(pkgPairs) > 0 {
			maxLabel := 0
			for _, p := range pkgPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range pkgPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				value := DetailValueStyle.Render(p[1])
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Package", lines})
		}

		var metaPairs [][2]string
		if info.License != "" {
			metaPairs = append(metaPairs, [2]string{"License", info.License})
		}
		if info.Homepage != "" {
			metaPairs = append(metaPairs, [2]string{"Homepage", info.Homepage})
		}
		if info.Publisher != "" {
			metaPairs = append(metaPairs, [2]string{"Publisher", info.Publisher})
		}
		if len(metaPairs) > 0 {
			maxLabel := 0
			for _, p := range metaPairs {
				w := lipgloss.Width(p[0])
				if w > maxLabel {
					maxLabel = w
				}
			}
			var lines []string
			for _, p := range metaPairs {
				label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
				var value string
				if p[0] == "Homepage" {
					value = LinkStyle.Render(p[1])
				} else {
					value = DetailValueStyle.Render(p[1])
				}
				line := label + "  " + value
				allWidths = append(allWidths, lipgloss.Width(line))
				lines = append(lines, line)
			}
			sections = append(sections, sectionData{"Metadata", lines})
		}

		sectionWidth := width
		if len(allWidths) > 0 {
			maxW := 0
			for _, w := range allWidths {
				if w > maxW {
					maxW = w
				}
			}
			sectionWidth = min(width, max(maxW+4, 6))
		}

		for _, s := range sections {
			contentLines = append(contentLines, renderSection(sectionWidth, s.title, s.lines...))
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
	}
	contentLines = append(contentLines, "")
	return renderPaneBox(width, height, "Details", strings.Join(contentLines, "\n"))
}

func (m Model) listViewFallback() string {
	var title string
	var list string
	visibleHeight := m.height - 8

	if m.allMode {
		title = TitleStyle.Render(fmt.Sprintf("devpkgs  (%d)", len(m.allPackages)))
		start := 0
		if m.allCursor >= visibleHeight {
			start = m.allCursor - visibleHeight + 1
		}
		end := start + visibleHeight
		if end > len(m.allDisplayPackages) {
			end = len(m.allDisplayPackages)
		}
		pmBadgeFn := func(name string) lipgloss.Style {
			switch name {
			case "brew":
				return lipgloss.NewStyle().
					Width(5).
					Background(currentTheme.Primary).
					Foreground(currentTheme.SelectedFg)
			case "npm":
				return lipgloss.NewStyle().
					Width(5).
					Background(darkenHex(string(currentTheme.Primary), 0.65)).
					Foreground(currentTheme.Text)
			case "pip":
				return lipgloss.NewStyle().
					Width(5).
					Background(darkenHex(string(currentTheme.Success), 0.65)).
					Foreground(currentTheme.Text)
			default:
				return lipgloss.NewStyle().
					Width(5).
					Background(currentTheme.Muted).
					Foreground(currentTheme.Text)
			}
		}
		for i := start; i < end; i++ {
			pkg := m.allDisplayPackages[i]
			origin := m.allPackageOrigin[pkg]
			originRendered := pmBadgeFn(origin).Render(origin)
			var pkgRendered string
			if i == m.allCursor {
				pkgRendered = SelectedItemStyle.Render(pkg)
			} else {
				pkgRendered = ItemStyle.Render(pkg)
			}
			list += originRendered + " " + pkgRendered + "\n"
		}
	} else {
		st := m.states[m.activeTab]
		title = TitleStyle.Render(fmt.Sprintf("devpkgs  (%d)", len(st.packages)))
		start := 0
		if st.cursor >= visibleHeight {
			start = st.cursor - visibleHeight + 1
		}
		end := start + visibleHeight
		if end > len(st.displayPackages) {
			end = len(st.displayPackages)
		}
		for i := start; i < end; i++ {
			pkg := st.displayPackages[i]
			if i == st.cursor {
				list += SelectedItemStyle.Render(pkg) + "\n"
			} else {
				list += ItemStyle.Render(pkg) + "\n"
			}
		}
	}

	sep := lipgloss.NewStyle().
		Foreground(currentTheme.Primary).
		Padding(0, 1).
		Render(strings.Repeat("━", m.width-8))

	body := lipgloss.JoinVertical(lipgloss.Left, title, sep, list)
	return docStyle.Render(body)
}

func renderSection(maxWidth int, title string, lines ...string) string {
	violetStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)

	maxContent := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > maxContent {
			maxContent = w
		}
	}
	boxWidth := max(maxContent+4, lipgloss.Width(title)+6)
	boxWidth = min(boxWidth, maxWidth)

	inner := boxWidth - 4
	top := border.Render("╭━ ") +
		violetStyle.Render(title) +
		border.Render(" "+strings.Repeat("━", max(0, boxWidth-5-lipgloss.Width(title)))+"╮")

	var body []string
	for _, line := range lines {
		padded := lipgloss.NewStyle().Width(inner).Render(line)
		body = append(body, border.Render("│ ")+padded+border.Render(" │"))
	}
	bottom := border.Render("╰" + strings.Repeat("─", boxWidth-2) + "╯")

	return strings.Join(append([]string{top}, append(body, bottom)...), "\n")
}

func renderPaneBox(width int, height int, title string, content string) string {
	violetStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)

	top := border.Render("╭━ ") +
		violetStyle.Render(title) +
		border.Render(" "+strings.Repeat("━", max(0, width-5-lipgloss.Width(title)))+"╮")

	inner := width - 4
	lines := strings.Split(content, "\n")

	targetContentHeight := max(0, height-2)
	var body []string
	for i := 0; i < targetContentHeight; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		padded := lipgloss.NewStyle().Width(inner).MaxHeight(1).Render(line)
		body = append(body, border.Render("│ ")+padded+border.Render(" │"))
	}
	bottom := border.Render("╰" + strings.Repeat("─", width-2) + "╯")

	return strings.Join(append([]string{top}, append(body, bottom)...), "\n")
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
