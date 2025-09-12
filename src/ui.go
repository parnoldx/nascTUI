package main

import (
	"github.com/charmbracelet/bubbletea"
	_ "embed"
)

//go:embed help.txt
var helpText string

//go:embed input.txt
var inputTemplate string

// Update handles all UI state updates and message routing
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case pasteMsg:
		// Handle clipboard paste content (fallback - bracketed paste is preferred)
		return m.handlePasteMessage(string(msg))

	case pasteErrMsg:
		// Handle paste error silently
		return m, nil

	case tickMsg:
		// Check for terminal size changes
		return m.handleTickMessage()

	case CalculationMsg:
		return m.handleCalculationMessage(msg)

	case OpenCompletionsMsg:
		return m.handleOpenCompletionsMessage(msg)

	case FilterCompletionsMsg:
		return m.handleFilterCompletionsMessage(msg)

	case tea.MouseMsg:
		return m.handleMouseMessage(msg)

	case tea.KeyMsg:
		// Check for bracketed paste before textinput processes it
		if msg.Paste {
			pastedContent := string(msg.Runes)
			if result, cmd := m.handleBracketedPaste(pastedContent); cmd != nil {
				return result, cmd
			}
			// Single-line paste falls through to normal textinput processing
		}

		// Handle keyboard input
		if result, cmd := m.handleKeyMessage(msg); cmd != nil {
			return result, cmd
		}

	case tea.WindowSizeMsg:
		m.handleWindowResize(msg)
	}

	// Only update textinput if we're not showing completions (to avoid double updates)
	if !m.ShowCompletions {
		var cmd tea.Cmd
		m.Inputs[m.Focused], cmd = m.Inputs[m.Focused].Update(msg)
		cmds = append(cmds, cmd)

		// Only trigger calculation if not already calculating and input is non-empty
		currentExpr := m.Inputs[m.Focused].Value()
		if !m.Calculating[m.Focused] && currentExpr != "" {
			m.Calculating[m.Focused] = true
			cmds = append(cmds, CalculateCmd(currentExpr, m.Results, m.Focused))
		} else if currentExpr == "" {
			// Clear result when input is empty
			m.Results[m.Focused] = ""
			// Only update input viewport to avoid result pane flickering
			m.updateInputViewport()
		}
	}

	// Minimal viewport updates to prevent flickering
	var inputCmd, resultCmd tea.Cmd
	switch msg.(type) {
	case tea.WindowSizeMsg:
		// Allow viewport updates for resize events
		m.InputViewport, inputCmd = m.InputViewport.Update(msg)
		m.ResultViewport, resultCmd = m.ResultViewport.Update(msg)
		cmds = append(cmds, inputCmd, resultCmd)
	case tea.MouseMsg:
		// Allow input viewport updates for mouse events
		m.InputViewport, inputCmd = m.InputViewport.Update(msg)
		cmds = append(cmds, inputCmd)
	case tickMsg:
		// Completely ignore tick messages for viewport updates
	default:
		// For all other messages, suppress viewport component updates to prevent flickering
		// The viewports will be updated through our manual updateViewports() calls
		
		// Also filter out textinput blink commands that cause flickering
		switch msgTyped := msg.(type) {
		case tea.KeyMsg:
			if msgTyped.Type == tea.KeySpace {
				// Allow space key updates
			}
		}
	}

	// Only update viewports for specific message types to prevent flickering
	if !m.ShowCompletions {
		switch msg.(type) {
		case tea.WindowSizeMsg, tickMsg:
			// Don't update viewports during resize or tick - prevents flickering
		case CalculationMsg:
			// Update viewports when calculation results change
			m.updateViewports()
		case tea.KeyMsg:
			// Only update input viewport during typing, not result viewport
			keyMsg := msg.(tea.KeyMsg)
			switch keyMsg.Type {
			case tea.KeyUp, tea.KeyDown, tea.KeyCtrlK, tea.KeyCtrlJ:
				// Update viewports for navigation commands
				m.updateViewports()
			default:
				// For regular typing, only update input viewport
				m.updateInputViewport()
			}
		default:
			// For other messages (mouse, paste, etc.), update both viewports
			m.updateViewports()
		}
	}

	return m, tea.Batch(cmds...)
}