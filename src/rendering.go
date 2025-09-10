package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)


// styleAnsTokens applies styling to ans tokens in text
func (m Model) styleAnsTokens(text string) string {
	// Style ans1, ans2, etc. with highlight color
	for i := 1; i <= len(m.Results); i++ {
		ansToken := fmt.Sprintf("ans%d", i)
		if strings.Contains(text, ansToken) {
			styledToken := lipgloss.NewStyle().
				Foreground(m.Theme.ansColor).
				Bold(true).
				Render(ansToken)
			text = strings.ReplaceAll(text, ansToken, styledToken)
		}
	}

	// Style standalone 'ans' with highlight color using word boundary
	ansRegex := regexp.MustCompile(`\bans\b`)
	if ansRegex.MatchString(text) {
		styledAns := lipgloss.NewStyle().
			Foreground(m.Theme.ansColor).
			Bold(true).
			Render("ans")
		text = ansRegex.ReplaceAllString(text, styledAns)
	}

	return text
}

// updateViewports updates both input and result viewport content
func (m *Model) updateViewports() {
	m.updateInputViewport()
	m.updateResultViewport()
}

// updateInputViewport updates the input pane content with line number gutter
func (m *Model) updateInputViewport() {
	var inputLines []string
	for i, input := range m.Inputs {
		line := input.Value()
		if line == "" && i == m.Focused {
			line = input.Placeholder
		}

		// Create gutter with line number and separator
		gutter := fmt.Sprintf("%2d│", i+1)
		if i == m.Focused {
			gutter = lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Bold(true).
				Render(gutter)

			// Style ans/res tokens with boxes
			inputView := input.View()
			inputView = m.styleAnsTokens(inputView)
			renderWidth := m.InputViewport.Width - 6
			if renderWidth < 1 {
				renderWidth = 1
			}
			inputView = lipgloss.NewStyle().
				Width(renderWidth).
				Render(inputView)
			combined := lipgloss.JoinHorizontal(lipgloss.Top, gutter, " ", inputView)

			// Add completion popup after focused line if showing completions
			inputLines = append(inputLines, combined)
			if m.ShowCompletions && len(m.Completions) > 0 {
				completionLines := m.renderCompletionPopup()
				inputLines = append(inputLines, completionLines...)
			}
		} else {
			// Replace ans tokens with highlighted actual values on non-focused lines
			displayLine := m.replaceAnsTokensWithValues(line, i)
			// Don't style non-focused gutters - use default colors
			combined := lipgloss.JoinHorizontal(lipgloss.Top, gutter, " ", displayLine)
			inputLines = append(inputLines, combined)
		}
	}
	m.InputViewport.SetContent(strings.Join(inputLines, "\n"))
}

// updateResultViewport updates the results pane content
func (m *Model) updateResultViewport() {
	var resultLines []string
	for i := range m.Inputs {
		result := m.Results[i]

		// Constrain result to fit in result pane width
		if len(result) > m.ResultViewport.Width && m.ResultViewport.Width > 3 {
			result = result[:m.ResultViewport.Width-3] + "..."
		} else if len(result) > m.ResultViewport.Width && m.ResultViewport.Width > 0 {
			result = result[:m.ResultViewport.Width]
		}

		// Ensure positive width for lipgloss
		resultWidth := m.ResultViewport.Width
		if resultWidth <= 0 {
			resultWidth = 20 // Minimum fallback width
		}

		if i == m.Focused {
			result = lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Bold(true).
				Width(resultWidth).
				Render(result)
		} else {
			result = lipgloss.NewStyle().
				Width(resultWidth).
				Render(result)
		}
		resultLines = append(resultLines, result)

		// Add empty lines to match completion popup height
		if i == m.Focused && m.ShowCompletions && len(m.Completions) > 0 {
			popupHeight := len(m.Completions) + 2 // Account for border
			for j := 0; j < popupHeight; j++ {
				resultLines = append(resultLines, "")
			}
		}
	}
	m.ResultViewport.SetContent(strings.Join(resultLines, "\n"))
}

// renderCompletionPopup creates the completion popup lines
func (m *Model) renderCompletionPopup() []string {
	var completionItems []string
	maxWidth := 0

	// Implement scrolling window for completions
	maxItems := 10
	startIdx := 0
	endIdx := len(m.Completions)

	// Calculate scrolling window if there are more than maxItems
	if len(m.Completions) > maxItems {
		// Center the selected item in the visible window
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

		// Adjust index for scrolled window
		globalIdx := startIdx + j
		if globalIdx == m.SelectedCompletion {
			item := lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Background(lipgloss.Color("8")).
				Bold(true).
				Render("▶ " + completion)
			completionItems = append(completionItems, item)
		} else {
			item := lipgloss.NewStyle().
				Foreground(lipgloss.Color("7")).
				Render("  " + completion)
			completionItems = append(completionItems, item)
		}
	}

	completionContent := strings.Join(completionItems, "\n")
	popupWidth := maxWidth + 4 // Add padding
	if popupWidth < 20 {
		popupWidth = 20
	} else if popupWidth > 40 {
		popupWidth = 40
	}

	completionStyle := lipgloss.NewStyle().
		Width(popupWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Background(lipgloss.Color("0")).
		Padding(0, 1).
		MarginLeft(6) // Indent to align with input content

	popup := completionStyle.Render(completionContent)
	return strings.Split(popup, "\n")
}

// replaceAnsTokensWithValues replaces ans tokens with actual values for display
func (m *Model) replaceAnsTokensWithValues(line string, currentIndex int) string {
	displayLine := line
	var commentPart string

	// Split at comment boundary
	if commentPos := strings.Index(displayLine, "//"); commentPos != -1 {
		commentPart = displayLine[commentPos:]
		displayLine = displayLine[:commentPos]
	}

	for j := 0; j < currentIndex && j < len(m.Results); j++ {
		if m.Results[j] != "" {
			ansPattern := fmt.Sprintf("ans%d", j+1)
			if strings.Contains(displayLine, ansPattern) {
				styledValue := lipgloss.NewStyle().
					Foreground(m.Theme.ansColor).
					Bold(true).
					Render(m.Results[j])
				displayLine = strings.ReplaceAll(displayLine, ansPattern, styledValue)
			}
		}
	}

	// Replace standalone 'ans' with highlighted last result
	ansRegex := regexp.MustCompile(`\bans\b`)
	if ansRegex.MatchString(displayLine) {
		for j := currentIndex - 1; j >= 0; j-- {
			if m.Results[j] != "" {
				styledValue := lipgloss.NewStyle().
					Foreground(m.Theme.ansColor).
					Bold(true).
					Render(m.Results[j])
				displayLine = ansRegex.ReplaceAllString(displayLine, styledValue)
				break
			}
		}
	}

	// Rejoin with comment part
	return displayLine + commentPart
}

// View renders the main UI view
func (m Model) View() string {
	baseStyle := lipgloss.NewStyle().
		Height(m.Height - 2).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	inputStyle := baseStyle.Copy().
		Width(int(float64(m.Width)*0.7) - 2)

	resultStyle := baseStyle.Copy().
		Width(int(float64(m.Width)*0.3) - 2)

	inputPane := inputStyle.Render(m.InputViewport.View())
	resultPane := resultStyle.Render(m.ResultViewport.View())

	baseView := lipgloss.JoinHorizontal(lipgloss.Top, inputPane, resultPane)

	if m.ShowHelp {
		return m.renderHelpPopup()
	}

	if m.ShowGoToLine {
		return m.renderGoToLineDialog(baseView)
	}

	return baseView
}

// renderHelpPopup renders the help popup overlay
func (m Model) renderHelpPopup() string {
	// Use the scrollable viewport for help content
	helpContent := m.HelpViewport.View()

	scrollInfo := " (↑↓ to scroll, Esc to close)"

	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Padding(1, 2).
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("7")).
		Width(m.HelpViewport.Width + 4).  // Account for padding
		Height(m.HelpViewport.Height + 4) // Account for padding

	// Add title with scroll info
	title := "NaSC Help" + scrollInfo
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.Theme.focusedColor).
		Width(m.HelpViewport.Width)

	helpWithTitle := titleStyle.Render(title) + "\n\n" + helpContent
	helpBox := helpStyle.Render(helpWithTitle)

	// Center the help popup
	overlayStyle := lipgloss.NewStyle().
		Width(m.Width).
		Height(m.Height).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(helpBox)
}

// renderGoToLineDialog renders the go-to-line dialog overlay
func (m Model) renderGoToLineDialog(baseView string) string {
	// Create the go-to-line input dialog
	dialogContent := "Go to line: " + m.GoToLineInput.View()
	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Padding(0, 1).
		Background(lipgloss.Color("0")).
		Width(30).
		Render(dialogContent)

	// Split the base view into lines
	baseLines := strings.Split(baseView, "\n")
	
	// Ensure we have enough lines for the dialog height
	for len(baseLines) < m.Height {
		baseLines = append(baseLines, "")
	}
	
	// Calculate position for dialog (bottom center of input pane)
	inputPaneWidth := int(float64(m.Width) * 0.7)
	dialogY := m.Height - 6 // Position near bottom
	dialogX := inputPaneWidth/2 - 15 + 2 // Center in input pane
	
	// Create the dialog lines
	dialogLines := strings.Split(dialogBox, "\n")
	
	// Insert dialog into the base view at the calculated position
	for i, dialogLine := range dialogLines {
		lineIndex := dialogY + i
		if lineIndex >= 0 && lineIndex < len(baseLines) {
			existingLine := baseLines[lineIndex]
			
			// Get the visual width of the dialog line (without ANSI codes)
			dialogVisualWidth := lipgloss.Width(dialogLine)
			
			// Preserve existing content before and after the dialog
			prefix := ""
			suffix := ""
			
			// Extract prefix (content before dialog position)
			if dialogX > 0 && len(existingLine) > dialogX {
				// Get visual characters up to dialog position, preserving ANSI codes
				prefix = existingLine[:min(len(existingLine), dialogX)]
			} else if dialogX > 0 {
				// Pad if line is shorter than dialog position
				prefix = existingLine + strings.Repeat(" ", dialogX-lipgloss.Width(existingLine))
			}
			
			// Extract suffix (content after dialog)
			suffixStart := dialogX + dialogVisualWidth
			if suffixStart < lipgloss.Width(existingLine) {
				// Get remaining visual characters after dialog, preserving ANSI codes
				remaining := existingLine[min(len(existingLine), suffixStart):]
				suffix = remaining
			}
			
			// Reconstruct line: prefix + dialog + suffix
			baseLines[lineIndex] = prefix + dialogLine + suffix
		}
	}
	
	return strings.Join(baseLines, "\n")
}