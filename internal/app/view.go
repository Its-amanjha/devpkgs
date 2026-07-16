package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if !m.allLoaded() && !m.searchTabActive {
		return m.renderLoading()
	}

	if !m.allMode {
		st := m.states[m.activeTab]
		if st.err != nil {
			return ErrorStyle.Render(fmt.Sprintf("Error: %v", st.err))
		}
	}

	if m.width < 60 {
		return m.listViewFallback()
	}

	contentWidth := m.width - 6
	leftWidth := int(float64(contentWidth) * 0.35)
	rightWidth := contentWidth - leftWidth

	searchLine := m.renderSearchBar(contentWidth, m.searchActive)
	searchOffset := strings.Count(searchLine, "\n") + 1

	boxHeight := max(0, m.height-12-searchOffset)

	var leftPanel, rightPanel string
	if m.searchTabActive {
		leftPanel = renderPaneBox(leftWidth, boxHeight, "Search Results", "Type a package name and press Enter to search.")
		rightPanel = renderPaneBox(rightWidth, boxHeight, "Details", "")
	} else {
		leftPanel = m.renderLeftPanel(leftWidth, boxHeight)
		rightPanel = m.renderRightPanel(rightWidth, boxHeight)
	}

	leftStyled := lipgloss.NewStyle().Width(leftWidth).Height(boxHeight).Render(leftPanel)
	rightStyled := lipgloss.NewStyle().Width(rightWidth).Height(boxHeight).Render(rightPanel)

	top := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

	var bodyParts []string
	bodyParts = append(bodyParts, m.renderHeader())
	bodyParts = append(bodyParts, "")
	bodyParts = append(bodyParts, m.renderTabBar(contentWidth))
	bodyParts = append(bodyParts, "")
	bodyParts = append(bodyParts, searchLine)
	bodyParts = append(bodyParts, "")
	bodyParts = append(bodyParts, top)
	bodyParts = append(bodyParts, m.renderFooter())

	body := lipgloss.JoinVertical(lipgloss.Left, bodyParts...)

	rendered := docStyle.Render(body)

	if m.logOverlay {
		return m.renderLogOverlay()
	}
	if m.themeOverlay {
		return m.renderThemeOverlay()
	}
	if m.actionOverlay {
		return m.renderActionOverlay()
	}

	return rendered
}

func (m Model) renderHeader() string {
	label := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary).Render("devpkgs — a terminal dashboard for everything you've installed ")

	sparkText := ""
	sparkW := min(20, max(1, (m.width-20)/3))
	if len(m.sparklineHistory) > 1 {
		maxVal := 0.0
		for _, v := range m.sparklineHistory {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}
		norm := make([]float64, len(m.sparklineHistory))
		for i, v := range m.sparklineHistory {
			norm[i] = v / maxVal
		}
		sparkH := 2
		sparkText = RenderBrailleSparkline(norm, sparkW, sparkH)
	}

	if sparkText != "" {
		return label + "\n" + sparkText
	}
	return label
}

func (m Model) renderLoading() string {
	doneStyle := lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)
	fillStyle := lipgloss.NewStyle().Foreground(currentTheme.DetailText)
	emptyStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText)

	spinnerLine := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render(
		m.spinner.View() + " Loading packages...",
	)

	sparkW := min(30, max(5, m.width/3))
	var sparkArea string
	if len(m.sparklineHistory) > 1 {
		maxVal := 0.0
		for _, v := range m.sparklineHistory {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}
		norm := make([]float64, len(m.sparklineHistory))
		for i, v := range m.sparklineHistory {
			norm[i] = v / maxVal
		}
		sparkH := 3
		sparkArea = RenderBrailleSparkline(norm, sparkW, sparkH)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+spinnerLine)
	lines = append(lines, "")

	for i, tab := range m.tabs {
		name := strings.ToUpper(tab.Name())
		label := labelStyle.Render(name)
		st := m.states[i]

		if !st.loading {
			lines = append(lines, "  "+label+"  "+doneStyle.Render("[✓]"))
			continue
		}

		n := int(st.progress * 20)
		if n > 20 {
			n = 20
		}
		bar := "[" + fillStyle.Render(strings.Repeat("█", n)) + emptyStyle.Render(strings.Repeat("░", 20-n)) + "]"
		lines = append(lines, "  "+label+"  "+bar)
	}

	if sparkArea != "" {
		lines = append(lines, "")
		lines = append(lines, sparkArea)
	}

	return LoadingStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderSearchBar(width int, focused bool) string {
	borderColor := currentTheme.Primary
	if !focused {
		borderColor = currentTheme.Muted
	}
	border := lipgloss.NewStyle().Foreground(borderColor)
	violetBold := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary)

	badge := "Search"
	if m.searchTabActive {
		badge = "Search Registry"
	}
	top := border.Render("╭━ ") +
		violetBold.Render(badge) +
		border.Render(" "+strings.Repeat("━", max(0, width-5-lipgloss.Width(badge)))+"╮")

	inner := width - 4

	var inputLine string
	if focused {
		cursor := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render("█")
		if m.searchQuery == "" {
			inputLine = cursor + " " + SearchPlaceholderStyle.Render("Search packages...")
		} else {
			inputLine = DetailValueStyle.Render(m.searchQuery) + cursor
		}
	} else {
		if m.searchQuery == "" {
			inputLine = SearchPlaceholderStyle.Render("Search packages...")
		} else {
			inputLine = DetailValueStyle.Render(m.searchQuery)
		}
	}

	padded := lipgloss.NewStyle().Width(inner).Render(inputLine)
	body := border.Render("│ ") + padded + border.Render(" │")
	bottom := border.Render("╰" + strings.Repeat("─", width-2) + "╯")

	return strings.Join([]string{top, body, bottom}, "\n")
}
