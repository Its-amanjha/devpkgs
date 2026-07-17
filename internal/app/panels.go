package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

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

	showCheckboxes := len(st.selected) > 0
	var listItems []string
	for i := start; i < end; i++ {
		pkg := st.displayPackages[i]
		var displayPkg string
		if showCheckboxes {
			prefix := "[ ] "
			if st.selected[pkg] {
				prefix = "[✓] "
			}
			displayPkg = prefix + truncateString(pkg, max(0, innerWidth-4))
		} else {
			displayPkg = truncateString(pkg, innerWidth)
		}

		if i == st.cursor {
			style := SelectedItemStyle.Width(innerWidth)
			listItems = append(listItems, style.Render(displayPkg))
		} else {
			listItems = append(listItems, ItemStyle.Render(displayPkg))
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

	typeWidth := 6
	innerWidth := width - 4

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
		
		maxPkgLen := innerWidth - 7
		displayPkg := truncateString(pkg, maxPkgLen)
		
		var pkgRendered string
		if i == m.allCursor {
			pkgRendered = SelectedItemStyle.Render(displayPkg)
		} else {
			pkgRendered = ItemStyle.Render(displayPkg)
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Package", lines...))
			contentLines = append(contentLines, "")
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Metadata", lines...))
			contentLines = append(contentLines, "")
		}

		if len(info.Dependencies) > 0 {
			line := DetailValueStyle.Render(strings.Join(info.Dependencies, ", "))
			contentLines = append(contentLines, renderSection(width, "Dependencies", line))
			contentLines = append(contentLines, "")
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  No formula data available"))
	}
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Package", lines...))
			contentLines = append(contentLines, "")
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Metadata", lines...))
			contentLines = append(contentLines, "")
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
	}
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Package", lines...))
			contentLines = append(contentLines, "")
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Metadata", lines...))
			contentLines = append(contentLines, "")
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
	}
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Package", lines...))
			contentLines = append(contentLines, "")
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
				lines = append(lines, label+"  "+value)
			}
			contentLines = append(contentLines, renderSection(width, "Metadata", lines...))
			contentLines = append(contentLines, "")
		}
	} else {
		contentLines = append(contentLines, DetailValueStyle.Render("  Loading..."))
	}
	return renderPaneBox(width, height, "Details", strings.Join(contentLines, "\n"))
}

func (m Model) listViewFallback() string {
	var title string
	var list string
	visibleHeight := m.height - 8

	if m.searchTabActive {
		title = TitleStyle.Render(fmt.Sprintf("Search Registry (%d)", len(m.searchResults)))
		if len(m.searchResults) == 0 {
			if m.searchLoading {
				list = "  Searching registries...\n"
			} else {
				list = "  Type a package name and press Enter to search.\n"
			}
		} else {
			start := 0
			if m.searchResultCursor >= visibleHeight {
				start = m.searchResultCursor - visibleHeight + 1
			}
			end := start + visibleHeight
			if end > len(m.searchResults) {
				end = len(m.searchResults)
			}
			for i := start; i < end; i++ {
				res := m.searchResults[i]
				pkgStr := fmt.Sprintf("[%s] %s", res.Manager, res.Name)
				if i == m.searchResultCursor {
					list += SelectedItemStyle.Render(pkgStr) + "\n"
				} else {
					list += ItemStyle.Render(pkgStr) + "\n"
				}
			}
		}
	} else if m.allMode {
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
		showCheckboxes := len(st.selected) > 0
		for i := start; i < end; i++ {
			pkg := st.displayPackages[i]
			displayPkg := pkg
			if showCheckboxes {
				prefix := "[ ] "
				if st.selected[pkg] {
					prefix = "[✓] "
				}
				displayPkg = prefix + pkg
			}
			if i == st.cursor {
				list += SelectedItemStyle.Render(displayPkg) + "\n"
			} else {
				list += ItemStyle.Render(displayPkg) + "\n"
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
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
	valueStyle := lipgloss.NewStyle().PaddingLeft(2).Foreground(currentTheme.Text)

	var res []string
	res = append(res, titleStyle.Render(title+":"))
	for _, line := range lines {
		wrapped := lipgloss.NewStyle().Width(maxWidth - 4).Render(line)
		wrappedLines := strings.Split(wrapped, "\n")
		for _, wl := range wrappedLines {
			res = append(res, valueStyle.Render(wl))
		}
	}

	if title == "Description" && len(res) > 4 {
		res = res[:4]
		res[3] = truncateString(res[3], maxWidth-8) + "..."
	}

	return strings.Join(res, "\n")
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

func (m Model) renderSearchLeftPanel(width, height int) string {
	innerWidth := width - 4
	visibleHeight := height - 4

	var lines []string
	if m.searchLoading && len(m.searchResults) == 0 {
		lines = append(lines, "  "+m.spinner.View()+" Searching registries...")
	} else if len(m.searchResults) == 0 {
		lines = append(lines, "  Type a package name and press Enter to search.")
	} else {
		start := 0
		if m.searchResultCursor >= visibleHeight {
			start = m.searchResultCursor - visibleHeight + 1
		}
		end := start + visibleHeight
		if end > len(m.searchResults) {
			end = len(m.searchResults)
		}

		pmBadgeFn := func(name string) lipgloss.Style {
			switch name {
			case "brew":
				return lipgloss.NewStyle().
					Width(6).
					Background(currentTheme.Primary).
					Foreground(currentTheme.SelectedFg)
			case "npm":
				return lipgloss.NewStyle().
					Width(6).
					Background(darkenHex(string(currentTheme.Primary), 0.65)).
					Foreground(currentTheme.Text)
			case "pip":
				return lipgloss.NewStyle().
					Width(6).
					Background(darkenHex(string(currentTheme.Success), 0.65)).
					Foreground(currentTheme.Text)
			case "winget":
				return lipgloss.NewStyle().
					Width(6).
					Background(darkenHex(string(currentTheme.Primary), 0.35)).
					Foreground(currentTheme.Text)
			default:
				return lipgloss.NewStyle().
					Width(6).
					Background(currentTheme.Muted).
					Foreground(currentTheme.Text)
			}
		}

		for i := start; i < end; i++ {
			res := m.searchResults[i]
			badge := pmBadgeFn(res.Manager).Render(strings.ToUpper(res.Manager))
			
			namePart := res.Name
			descPart := ""
			if res.Description != "" {
				descPart = " — " + res.Description
			}
			
			fullText := namePart + descPart
			maxTextLen := max(0, innerWidth-8)
			truncated := truncateString(fullText, maxTextLen)

			var line string
			if i == m.searchResultCursor {
				line = badge + " " + SelectedItemStyle.Render(truncated)
			} else {
				line = badge + " " + ItemStyle.Render(truncated)
			}
			lines = append(lines, line)
		}

		for len(lines) < visibleHeight {
			lines = append(lines, "")
		}
	}

	content := strings.Join(lines, "\n")
	title := "Search Results"
	if len(m.searchResults) > 0 {
		title = fmt.Sprintf("Search Results (%d)", len(m.searchResults))
	}
	return renderPaneBox(width, height, title, content)
}

func (m Model) renderSearchRightPanel(width, height int) string {
	if len(m.searchResults) == 0 {
		return renderPaneBox(width, height, "Details", "")
	}

	res := m.searchResults[m.searchResultCursor]
	var contentLines []string
	contentLines = append(contentLines, "")

	var pairs [][2]string
	pairs = append(pairs, [2]string{"Name", res.Name})
	pairs = append(pairs, [2]string{"Registry", strings.ToUpper(res.Manager)})
	if res.Version != "" {
		pairs = append(pairs, [2]string{"Version", res.Version})
	}
	if res.Description != "" {
		pairs = append(pairs, [2]string{"Summary", res.Description})
	}

	maxLabel := 0
	for _, p := range pairs {
		w := lipgloss.Width(p[0])
		if w > maxLabel {
			maxLabel = w
		}
	}

	var lines []string
	for _, p := range pairs {
		label := lipgloss.NewStyle().Width(maxLabel).Bold(true).Foreground(currentTheme.Primary).Render(p[0])
		var value string
		if p[0] == "Summary" {
			value = DetailValueStyle.Render(p[1])
		} else {
			value = DetailValueStyle.Render(p[1])
		}
		lines = append(lines, label+"  "+value)
	}

	contentLines = append(contentLines, renderSection(width-4, "Registry Package Information", lines...))
	content := strings.Join(contentLines, "\n")
	return renderPaneBox(width, height, "Details", content)
}
