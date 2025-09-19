package main

import (
	"slices"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

// insertCompletion inserts a completion at the current cursor position
func (m *Model) insertCompletion(completion string) {
	// Save state before inserting completion
	m.saveState()
	
	currentValue := m.Inputs[m.Focused].Value()
	cursorPos := m.Inputs[m.Focused].Position()

	// Find start of current word to replace
	wordStart := cursorPos
	for wordStart > 0 && currentValue[wordStart-1] != ' ' && !slices.Contains(operators, string(currentValue[wordStart-1])) {
		wordStart--
	}

	newValue := currentValue[:wordStart] + completion + currentValue[cursorPos:]
	m.Inputs[m.Focused].SetValue(newValue)
	m.Inputs[m.Focused].SetCursor(wordStart + len(completion))
}

// insertSymbol inserts a symbol at the current cursor position
func (m *Model) insertSymbol(symbol string) (tea.Model, tea.Cmd) {
	// Save state before inserting symbol
	m.saveState()
	
	var cmds []tea.Cmd

	currentValue := m.Inputs[m.Focused].Value()
	cursorPos := m.Inputs[m.Focused].Position()
	newValue := currentValue[:cursorPos] + symbol + currentValue[cursorPos:]
	m.Inputs[m.Focused].SetValue(newValue)
	m.Inputs[m.Focused].SetCursor(cursorPos + len(symbol))

	// Trigger calculation
	if !m.Calculating[m.Focused] && newValue != "" {
		m.Calculating[m.Focused] = true
		cmds = append(cmds, CalculateCmd(newValue, m.Results, m.Focused))
	}

	return *m, tea.Batch(cmds...)
}

// triggerCalculationIfNeeded triggers calculation if input is non-empty
func (m *Model) triggerCalculationIfNeeded() []tea.Cmd {
	var cmds []tea.Cmd

	currentExpr := m.Inputs[m.Focused].Value()
	if !m.Calculating[m.Focused] && currentExpr != "" {
		m.Calculating[m.Focused] = true
		cmds = append(cmds, CalculateCmd(currentExpr, m.Results, m.Focused))
	} else if currentExpr == "" {
		// Clear result when input is empty
		m.Results[m.Focused] = ""
		m.updateViewports()
	}

	return cmds
}

// openHelp opens the help popup
func (m *Model) openHelp() (tea.Model, tea.Cmd) {
	m.ShowHelp = true
	maxHelpHeight := int(float64(m.Height) * 0.8)
	helpHeight := min(maxHelpHeight, m.Height-6)
	if m.Height <= 10 {
		helpHeight = m.Height - 3
	}
	helpWidth := min(80, m.Width-4)
	if helpWidth < 30 {
		helpWidth = 30
	}
	m.HelpViewport.Width = helpWidth
	m.HelpViewport.Height = helpHeight
	m.HelpViewport.SetContent(helpText)
	return *m, textinput.Blink
}

// deleteLine deletes the current line or clears content if it's the only line
func (m *Model) deleteLine() (tea.Model, tea.Cmd) {
	// Save state before making changes
	m.saveState()
	
	if len(m.Inputs) > 1 {
		// Remove current line
		m.Inputs = append(m.Inputs[:m.Focused], m.Inputs[m.Focused+1:]...)
		m.Results = append(m.Results[:m.Focused], m.Results[m.Focused+1:]...)
		m.Calculating = append(m.Calculating[:m.Focused], m.Calculating[m.Focused+1:]...)

		// Adjust focus
		if m.Focused >= len(m.Inputs) {
			m.Focused = len(m.Inputs) - 1
		}
		for i := range m.Inputs {
			if i == m.Focused {
				m.Inputs[i].Focus()
			} else {
				m.Inputs[i].Blur()
			}
		}
		m.updateViewports()
		return *m, textinput.Blink
	} else {
		// Clear the content of the only line
		m.Inputs[m.Focused].SetValue("")
		m.Inputs[m.Focused].SetCursor(0)
		m.Results[m.Focused] = ""
		m.updateViewports()
		return *m, textinput.Blink
	}
}

// clearAll clears all inputs and results
func (m *Model) clearAll() (tea.Model, tea.Cmd) {
	// Save state before making changes
	m.saveState()
	ti := textinput.New()
	ti.Placeholder = defaultPlaceholder
	ti.Focus()
	ti.Width = m.GetTextInputWidth()
	ti.Prompt = ""
	ti.CharLimit = 0

	m.Inputs = []textinput.Model{ti}
	m.Results = []string{""}
	m.Calculating = []bool{false}
	m.Focused = 0
	m.updateViewports()
	m.scrollToFocused()
	return *m, textinput.Blink
}

// showContentAssist shows content assist popup
func (m *Model) showContentAssist() (tea.Model, tea.Cmd) {
	currentValue := m.Inputs[m.Focused].Value()
	cursorPos := m.Inputs[m.Focused].Position()

	// Get current word being typed
	wordStart := cursorPos
	for wordStart > 0 && currentValue[wordStart-1] != ' ' {
		wordStart--
	}
	currentWord := currentValue[wordStart:cursorPos]

	return *m, OpenCompletionsCmd(currentWord, m.Results)
}

// showCompletions shows completions popup
func (m *Model) showCompletions() (tea.Model, tea.Cmd) {
	currentValue := m.Inputs[m.Focused].Value()
	cursorPos := m.Inputs[m.Focused].Position()

	// Get current word being typed
	wordStart := cursorPos
	for wordStart > 0 && currentValue[wordStart-1] != ' ' {
		wordStart--
	}
	currentWord := currentValue[wordStart:cursorPos]

	return *m, OpenCompletionsCmd(currentWord, m.Results)
}

// createNewLine creates a new input line after the current focused line
func (m *Model) createNewLine() (tea.Model, tea.Cmd) {
	// Save state before making changes
	m.saveState()
	newInput := textinput.New()
	newInput.Placeholder = ""
	newInput.Width = m.GetTextInputWidth() // Account for gutter width
	newInput.Prompt = ""
	
	// Insert new line after the current focused line
	insertIndex := m.Focused + 1
	
	// Insert at the specific position
	m.Inputs = append(m.Inputs[:insertIndex], append([]textinput.Model{newInput}, m.Inputs[insertIndex:]...)...)
	m.Results = append(m.Results[:insertIndex], append([]string{""}, m.Results[insertIndex:]...)...)
	m.Calculating = append(m.Calculating[:insertIndex], append([]bool{false}, m.Calculating[insertIndex:]...)...)

	// Move focus to the newly inserted line
	m.Focused = insertIndex
	for i := range m.Inputs {
		if i == m.Focused {
			m.Inputs[i].Focus()
		} else {
			m.Inputs[i].Blur()
		}
	}
	m.updateViewports()
	m.scrollToFocused()
	return *m, textinput.Blink
}

// focusPreviousLine moves focus to the previous line
func (m *Model) focusPreviousLine() (tea.Model, tea.Cmd) {
	if m.Focused > 0 {
		m.Inputs[m.Focused].Blur()
		m.Focused--
		m.Inputs[m.Focused].Focus()
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

// focusNextLine moves focus to the next line
func (m *Model) focusNextLine() (tea.Model, tea.Cmd) {
	if m.Focused < len(m.Inputs)-1 {
		m.Inputs[m.Focused].Blur()
		m.Focused++
		m.Inputs[m.Focused].Focus()
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

// focusFirstLine moves focus to the first line
func (m *Model) focusFirstLine() (tea.Model, tea.Cmd) {
	if m.Focused != 0 {
		m.Inputs[m.Focused].Blur()
		m.Focused = 0
		m.Inputs[m.Focused].Focus()
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

// focusLastLine moves focus to the last line
func (m *Model) focusLastLine() (tea.Model, tea.Cmd) {
	lastIndex := len(m.Inputs) - 1
	if m.Focused != lastIndex {
		m.Inputs[m.Focused].Blur()
		m.Focused = lastIndex
		m.Inputs[m.Focused].Focus()
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

// pasteInputTemplate pastes the input template content
func (m Model) pasteInputTemplate() (tea.Model, tea.Cmd) {
	// Save state before making changes
	m.saveState()
	m.addMultipleInputs(inputTemplate)

	// Update viewports and scroll
	m.updateViewports()
	m.scrollToFocused()

	return m, textinput.Blink
}

// handleBracketedPaste handles bracketed paste content
func (m *Model) handleBracketedPaste(pastedContent string) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Check for various line ending formats: \n, \r\n, or \r
	if strings.Contains(pastedContent, "\n") || strings.Contains(pastedContent, "\r") {
		// Normalize line endings to \n before processing
		normalized := strings.ReplaceAll(pastedContent, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "\n")

		m.addMultipleInputs(normalized)
		m.updateViewports()
		m.scrollToFocused()
		return *m, tea.Batch(cmds...)
	}
	// Single-line paste falls through to normal textinput processing
	return *m, textinput.Blink
}

// openGoToLine opens the go-to-line input dialog
func (m *Model) openGoToLine() (tea.Model, tea.Cmd) {
	m.ShowGoToLine = true
	m.GoToLineInput.SetValue("")
	m.GoToLineInput.Focus()
	return *m, textinput.Blink
}

// goToLine jumps to the specified line number
func (m *Model) goToLine() (tea.Model, tea.Cmd) {
	lineInput := strings.TrimSpace(m.GoToLineInput.Value())
	
	// Close the go-to-line dialog
	m.ShowGoToLine = false
	m.GoToLineInput.Blur()
	
	if lineInput == "" {
		return *m, textinput.Blink
	}
	
	// Parse line number
	lineNumber, err := strconv.Atoi(lineInput)
	if err != nil || lineNumber < 1 {
		// Invalid line number, do nothing
		return *m, textinput.Blink
	}
	
	// Convert to 0-based index
	targetIndex := lineNumber - 1
	
	// Ensure target line exists
	if targetIndex >= len(m.Inputs) {
		// Jump to last line if target is beyond range
		targetIndex = len(m.Inputs) - 1
	}
	
	// Change focus
	m.Inputs[m.Focused].Blur()
	m.Focused = targetIndex
	m.Inputs[m.Focused].Focus()
	
	// Update viewports and scroll to show the target line
	m.updateViewports()
	m.scrollToFocused()
	
	return *m, textinput.Blink
}

// cancelGoToLine cancels the go-to-line dialog
func (m *Model) cancelGoToLine() (tea.Model, tea.Cmd) {
	m.ShowGoToLine = false
	m.GoToLineInput.SetValue("")
	m.GoToLineInput.Blur()
	return *m, textinput.Blink
}

// copyFocusedResult copies the result of the focused line to clipboard
func (m *Model) copyFocusedResult() (tea.Model, tea.Cmd) {
	if m.Focused >= 0 && m.Focused < len(m.Results) && m.Results[m.Focused] != "" {
		err := clipboard.WriteAll(m.Results[m.Focused])
		if err != nil {
			// Silently ignore clipboard errors
			return *m, nil
		}
	}
	return *m, nil
}