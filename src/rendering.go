package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) styleAnsTokens(text string) string {
	styled := text
	for i := len(m.Results); i >= 1; i-- {
		ansToken := fmt.Sprintf("ans%d", i)
		if strings.Contains(styled, ansToken) {
			styledToken := lipgloss.NewStyle().
				Foreground(m.Theme.ansColor).
				Bold(true).
				Render(ansToken)
			styled = strings.ReplaceAll(styled, ansToken, styledToken)
		}
	}

	if ansWordRegex.MatchString(stripANSIEscapeCodes(styled)) {
		styledAns := lipgloss.NewStyle().
			Foreground(m.Theme.ansColor).
			Bold(true).
			Render("ans")
		styled = ansWordRegex.ReplaceAllString(styled, styledAns)
	}

	return styled
}

func (m *Model) updateViewports() {
	m.updateInputViewport()
	m.updateResultViewport()
}

func (m *Model) updateInputViewport() {
	var inputLines []string
	for i, input := range m.Inputs {
		line := input.Value()
		if line == "" && i == m.Focused {
			line = input.Placeholder
		}

		gutterPlain := fmt.Sprintf("%2d│", i+1)
		gutterStyle := lipgloss.NewStyle().Foreground(m.Theme.gutterColor)
		if i == m.Focused {
			gutterStyle = lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Bold(true)
		}
		gutter := gutterStyle.Render(gutterPlain)

		if i == m.Focused {
			inputView := m.styleAnsTokens(input.View())
			combined := lipgloss.JoinHorizontal(lipgloss.Top, gutter, " ", inputView)
			inputLines = append(inputLines, combined)
			if m.ShowCompletions && len(m.Completions) > 0 {
				inputLines = append(inputLines, m.renderCompletionPopup()...)
			}
			continue
		}

		displayLine := m.replaceAnsTokensWithValues(line, i)
		maxDisplayWidth := m.GetTextInputWidth()
		if lipgloss.Width(displayLine) > maxDisplayWidth {
			displayLine = truncateVisual(displayLine, maxDisplayWidth)
		}
		combined := lipgloss.JoinHorizontal(lipgloss.Top, gutter, " ", displayLine)
		inputLines = append(inputLines, combined)
	}
	m.InputViewport.SetContent(strings.Join(inputLines, "\n"))
}

func (m *Model) updateResultViewport() {
	var resultLines []string
	resultWidth := m.ResultViewport.Width
	if resultWidth <= 0 {
		resultWidth = 20
	}

	for i := range m.Inputs {
		result := m.Results[i]
		if lipgloss.Width(result) > resultWidth {
			result = truncateVisual(result, resultWidth)
		}

		style := lipgloss.NewStyle().Foreground(m.Theme.resultColor)
		if i == m.Focused {
			style = lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Bold(true)
		}
		result = padOrTrimVisual(style.Render(result), resultWidth)
		resultLines = append(resultLines, result)

		if i == m.Focused && m.ShowCompletions && len(m.Completions) > 0 {
			popupHeight := len(m.Completions) + 2
			if popupHeight > 12 {
				popupHeight = 12
			}
			for j := 0; j < popupHeight; j++ {
				resultLines = append(resultLines, "")
			}
		}
	}

	newContent := strings.Join(resultLines, "\n")
	if newContent != m.LastResultContent {
		m.ResultViewport.SetContent(newContent)
		m.LastResultContent = newContent
	}
}

func (m *Model) renderCompletionPopup() []string {
	var completionItems []string
	maxWidth := 0

	maxItems := 10
	startIdx := 0
	endIdx := len(m.Completions)

	if len(m.Completions) > maxItems {
		startIdx = m.SelectedCompletion - maxItems/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + maxItems
		if endIdx > len(m.Completions) {
			endIdx = len(m.Completions)
			startIdx = endIdx - maxItems
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	displayCompletions := m.Completions[startIdx:endIdx]

	for j, completion := range displayCompletions {
		if len(completion) > maxWidth {
			maxWidth = len(completion)
		}

		globalIdx := startIdx + j
		if globalIdx == m.SelectedCompletion {
			item := lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Background(m.Theme.completionSelBg).
				Bold(true).
				Render("▶ " + completion)
			completionItems = append(completionItems, item)
		} else {
			item := lipgloss.NewStyle().
				Foreground(m.Theme.completionFg).
				Render("  " + completion)
			completionItems = append(completionItems, item)
		}
	}

	completionContent := strings.Join(completionItems, "\n")
	popupWidth := maxWidth + 4
	if popupWidth < 20 {
		popupWidth = 20
	} else if popupWidth > 40 {
		popupWidth = 40
	}

	popup := lipgloss.NewStyle().
		Width(popupWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Background(m.Theme.popupBg).
		Padding(0, 1).
		MarginLeft(6).
		Render(completionContent)
	return strings.Split(popup, "\n")
}

func (m *Model) replaceAnsTokensWithValues(line string, currentIndex int) string {
	displayLine := line
	var commentPart string

	if commentPos := strings.Index(displayLine, "//"); commentPos != -1 {
		commentPart = displayLine[commentPos:]
		displayLine = displayLine[:commentPos]
	}

	matches := ansNumRegex.FindAllStringSubmatchIndex(displayLine, -1)
	type repl struct {
		start, end int
		value      string
	}
	var repls []repl
	for _, loc := range matches {
		n := 0
		fmt.Sscanf(displayLine[loc[2]:loc[3]], "%d", &n)
		idx := n - 1
		if idx < 0 || idx >= currentIndex || idx >= len(m.Results) || m.Results[idx] == "" {
			continue
		}
		repls = append(repls, repl{loc[0], loc[1], m.Results[idx]})
	}
	for i := len(repls) - 1; i >= 0; i-- {
		r := repls[i]
		styledValue := lipgloss.NewStyle().
			Foreground(m.Theme.ansColor).
			Bold(true).
			Render(r.value)
		displayLine = displayLine[:r.start] + styledValue + displayLine[r.end:]
	}

	if ansWordRegex.MatchString(displayLine) {
		for j := currentIndex - 1; j >= 0; j-- {
			if m.Results[j] != "" {
				styledValue := lipgloss.NewStyle().
					Foreground(m.Theme.ansColor).
					Bold(true).
					Render(m.Results[j])
				displayLine = ansWordRegex.ReplaceAllString(displayLine, styledValue)
				break
			}
		}
	}

	return displayLine + commentPart
}

func (m Model) View() string {
	baseStyle := lipgloss.NewStyle().
		Height(m.Height-2).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	inputStyle := baseStyle.
		Width(int(float64(m.Width)*0.7) - 2).
		Background(m.Theme.inputBg)

	resultStyle := baseStyle.
		Width(int(float64(m.Width)*0.3) - 2).
		Background(m.Theme.resultBg)

	inputPane := inputStyle.Render(m.InputViewport.View())
	resultPane := resultStyle.Render(m.ResultViewport.View())
	baseView := lipgloss.JoinHorizontal(lipgloss.Top, inputPane, resultPane)

	if m.ShowHelp {
		return overlayCentered(baseView, m.renderHelpBox(), m.Width, m.Height)
	}
	if m.ShowGoToLine {
		return m.renderGoToLineDialog(baseView)
	}
	return baseView
}

func (m Model) renderHelpBox() string {
	helpContent := m.HelpViewport.View()

	title := "NaSC (↑↓ to scroll, Esc to close)"
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.Theme.focusedColor).
		Width(m.HelpViewport.Width)

	helpWithTitle := titleStyle.Render(title) + "\n\n" + helpContent
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Padding(1, 2).
		Background(m.Theme.popupBg).
		Foreground(m.Theme.unfocusedColor).
		Width(m.HelpViewport.Width + 4).
		Height(m.HelpViewport.Height + 4).
		Render(helpWithTitle)
}

func overlayCentered(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}

	overlayLines := strings.Split(overlay, "\n")
	ow := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > ow {
			ow = w
		}
	}
	oh := len(overlayLines)
	x := (width - ow) / 2
	y := (height - oh) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	for i, line := range overlayLines {
		row := y + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		existing := baseLines[row]
		prefix := padOrTrimVisual(existing, x)
		suffixStart := x + lipgloss.Width(line)
		suffix := ""
		if suffixStart < lipgloss.Width(existing) {
			suffix = visualSlice(existing, suffixStart, lipgloss.Width(existing))
		}
		baseLines[row] = prefix + line + suffix
	}
	return strings.Join(baseLines, "\n")
}

func (m Model) renderGoToLineDialog(baseView string) string {
	dialogContent := "Go to line: " + m.GoToLineInput.View()
	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Padding(0, 1).
		Background(m.Theme.popupBg).
		Width(30).
		Render(dialogContent)

	inputPaneWidth := int(float64(m.Width) * 0.7)
	x := inputPaneWidth/2 - 15 + 2
	y := m.Height - 6
	return overlayAt(baseView, dialogBox, x, y, m.Height)
}

func overlayAt(base, overlay string, x, y, height int) string {
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}
	if x < 0 {
		x = 0
	}

	for i, line := range strings.Split(overlay, "\n") {
		row := y + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		existing := baseLines[row]
		prefix := padOrTrimVisual(existing, x)
		suffixStart := x + lipgloss.Width(line)
		suffix := ""
		if suffixStart < lipgloss.Width(existing) {
			suffix = visualSlice(existing, suffixStart, lipgloss.Width(existing))
		}
		baseLines[row] = prefix + line + suffix
	}
	return strings.Join(baseLines, "\n")
}
