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
			m.updateViewports()
		}
	}

	// Update viewport components normally but prevent feedback during resize
	var inputCmd, resultCmd tea.Cmd
	m.InputViewport, inputCmd = m.InputViewport.Update(msg)
	m.ResultViewport, resultCmd = m.ResultViewport.Update(msg)
	cmds = append(cmds, inputCmd, resultCmd)

	// Only update viewports if not showing completions and not during resize
	if !m.ShowCompletions {
		switch msg.(type) {
		case tea.WindowSizeMsg:
			// Don't update viewports during resize - let bubbletea handle it
		default:
			m.updateViewports()
		}
	}

	return m, tea.Batch(cmds...)
}