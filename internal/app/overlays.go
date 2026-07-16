package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderFooter() string {
	total := m.totalPackages()
	countStr := ResultStyle.Render(fmt.Sprintf("%d results across all package managers", total))

	apiErrMsg := ""
	for i := range m.states {
		if m.states[i].Brew != nil && m.states[i].Brew.APIErr != nil {
			apiErrMsg = "  " + ErrorStyle.Render("API unavailable")
			break
		}
	}

	themeName := ""
	if currentTheme != nil {
		themeName = currentTheme.Name
	}
	help := FooterStyle.Render(fmt.Sprintf("[← → tabs] [/ search] [o outdated] [r refresh] [u upgrade] [x remove] [t theme %s] [q quit]", themeName))
	status := ""
	if m.actionStatus != "" {
		status = "  " + FooterStyle.Render(m.actionStatus)
	}
	return countStr + apiErrMsg + status + "  " + help
}

func (m Model) renderActionOverlay() string {
	action := string(m.pendingAction)
	content := fmt.Sprintf("%s %q using %s?\n\nEnter: confirm   y: with logs   Esc/n: cancel", action, m.pendingPackage, m.tabs[m.pendingTab].Name())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, renderPaneBox(60, 6, "Confirm package action", content))
}

func (m Model) renderThemeOverlay() string {
	boxW := min(52, m.width-6)
	innerW := boxW - 4

	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)
	titleText := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary).Render(" Choose Theme ")
	dashLen := max(0, boxW-4-lipgloss.Width(titleText))
	titleLine := border.Render("╭─"+strings.Repeat("─", dashLen/2)) + titleText +
		border.Render(strings.Repeat("─", dashLen-dashLen/2)+"─╮")

	nameColW := 14
	descColW := innerW - nameColW - 3

	var items []string
	for i, t := range themes {
		name := lipgloss.NewStyle().Width(nameColW).Render(t.Name)
		desc := t.Description
		if len(desc) > descColW {
			desc = desc[:descColW]
		}

		var line string
		if i == m.themeCursor {
			arrow := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render("›")
			nameStyled := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Text).Render(name)
			descStyled := lipgloss.NewStyle().Foreground(currentTheme.Primary).Render(desc)
			line = fmt.Sprintf("  %s %s %s", arrow, nameStyled, descStyled)
		} else {
			nameStyled := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Text).Render(name)
			descStyled := lipgloss.NewStyle().Foreground(currentTheme.DimText).Render(desc)
			line = fmt.Sprintf("   %s %s", nameStyled, descStyled)
		}
		padded := lipgloss.NewStyle().Width(innerW).Render(line)
		items = append(items, border.Render("│ ")+padded+border.Render(" │"))
	}
	content := strings.Join(items, "\n")

	bottom := border.Render("╰" + strings.Repeat("─", boxW-2) + "╯")
	footer := lipgloss.NewStyle().
		Foreground(currentTheme.DimText).
		Italic(true).
		Render("  ↑↓ navigate · enter select · esc close")

	overlay := strings.Join([]string{titleLine, content, bottom, footer}, "\n")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) renderLogOverlay() string {
	boxW := min(78, m.width-6)
	if boxW < 24 {
		boxW = 24
	}
	if boxW > m.width {
		boxW = m.width
	}
	
	boxH := min(22, m.height-6)
	if boxH < 8 {
		boxH = 8
	}
	if boxH > m.height {
		boxH = m.height
	}
	
	border := lipgloss.NewStyle().Foreground(currentTheme.Primary)
	titleText := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Primary).Render(" Installation Logs ")
	dashLen := max(0, boxW-4-lipgloss.Width(titleText))
	titleLine := border.Render("╭─"+strings.Repeat("─", dashLen/2)) + titleText +
		border.Render(strings.Repeat("─", dashLen-dashLen/2)+"─╮")
		
	contentH := boxH - 4
	
	var linesToShow []string
	totalLines := len(m.logLines)
	
	startLine := totalLines - contentH
	if m.logScrollActive {
		startLine = totalLines - contentH - m.logScrollOffset
	}
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + contentH
	if endLine > totalLines {
		endLine = totalLines
	}
	
	for i := startLine; i < endLine; i++ {
		linesToShow = append(linesToShow, m.logLines[i])
	}
	
	// Pad to fill box height
	for len(linesToShow) < contentH {
		linesToShow = append(linesToShow, "")
	}
	
	var items []string
	innerW := boxW - 4
	for _, l := range linesToShow {
		padded := lipgloss.NewStyle().Width(innerW).MaxHeight(1).Render(l)
		items = append(items, border.Render("│ ")+padded+border.Render(" │"))
	}
	content := strings.Join(items, "\n")
	
	bottomRepeat := boxW - 2
	if bottomRepeat < 0 {
		bottomRepeat = 0
	}
	bottom := border.Render("╰" + strings.Repeat("─", bottomRepeat) + "╯")
	
	footerText := "  ↑↓ scroll · esc/l close"
	if m.logActive {
		footerText = "  " + m.spinner.View() + " running... "
	}
	footer := lipgloss.NewStyle().Foreground(currentTheme.DimText).Italic(true).Render(footerText)
	
	overlay := strings.Join([]string{titleLine, content, bottom, footer}, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}
