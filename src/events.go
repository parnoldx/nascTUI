package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbletea"
)

// handlePasteMessage handles clipboard paste content
func (m *Model) handlePasteMessage(content string) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if strings.Contains(content, "\n") {
		// Multi-line content - add to existing inputs
		m.addMultipleInputs(content)
		m.updateViewports()
		m.scrollToFocused()
	} else if content != "" {
		// Single-line content - insert into current input
		currentValue := m.Inputs[m.Focused].Value()
		cursorPos := m.Inputs[m.Focused].Position()
		newValue := currentValue[:cursorPos] + content + currentValue[cursorPos:]
		m.Inputs[m.Focused].SetValue(newValue)
		m.Inputs[m.Focused].SetCursor(cursorPos + len(content))

		// Trigger calculation if non-empty
		if !m.Calculating[m.Focused] && newValue != "" {
			m.Calculating[m.Focused] = true
			cmds = append(cmds, CalculateCmd(newValue, m.Results, m.Focused))
		}
	}
	return *m, tea.Batch(cmds...)
}

// handleCalculationMessage handles calculation completion
func (m *Model) handleCalculationMessage(msg CalculationMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if msg.Index >= 0 && msg.Index < len(m.Results) {
		// Update model state (calculation manager is already updated in AsyncCalculateCmd)
		m.Results[msg.Index] = msg.Result
		m.Calculating[msg.Index] = false
		m.updateViewports()

		// Trigger recalculation of dependent lines
		for i := msg.Index + 1; i < len(m.Inputs); i++ {
			expr := m.Inputs[i].Value()
			if expr != "" && !m.Calculating[i] {
				m.Calculating[i] = true
				cmds = append(cmds, CalculateCmd(expr, m.Results, i))
			}
		}
	}
	return *m, tea.Batch(cmds...)
}

// handleOpenCompletionsMessage handles opening the completions popup
func (m *Model) handleOpenCompletionsMessage(msg OpenCompletionsMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.Completions = msg.Completions
	m.LastCompletionQuery = msg.Query

	if len(m.Completions) == 1 {
		// Auto-insert single completion
		m.insertCompletion(m.Completions[0])
		cmds = m.triggerCalculationIfNeeded()
	} else if len(m.Completions) > 1 {
		m.ShowCompletions = true
		m.SelectedCompletion = 0
		m.updateViewports()
	}

	return *m, tea.Batch(cmds...)
}

// handleFilterCompletionsMessage handles filtering completions
func (m *Model) handleFilterCompletionsMessage(msg FilterCompletionsMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.Completions = msg.Completions
	m.LastCompletionQuery = msg.Query

	if len(m.Completions) == 0 {
		m.ShowCompletions = false
	} else if len(m.Completions) == 1 {
		// Auto-insert single filtered completion
		m.insertCompletion(m.Completions[0])
		m.ShowCompletions = false
		m.LastCompletionQuery = ""
		cmds = m.triggerCalculationIfNeeded()
	} else {
		// Keep selection within bounds
		if m.SelectedCompletion >= len(m.Completions) {
			m.SelectedCompletion = len(m.Completions) - 1
		}
	}

	m.updateViewports()
	return *m, tea.Batch(cmds...)
}

// handleMouseMessage handles mouse interactions
func (m *Model) handleMouseMessage(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle mouse scroll in help popup
	if m.ShowHelp {
		switch msg.Type {
		case tea.MouseWheelUp:
			m.HelpViewport.LineUp(3) // Scroll up 3 lines
			return *m, nil
		case tea.MouseWheelDown:
			m.HelpViewport.LineDown(3) // Scroll down 3 lines
			return *m, nil
		}
	}

	if msg.Type == tea.MouseLeft {
		// Check if click is in result pane area
		resultPaneStart := int(float64(m.Width) * 0.7)
		if msg.X >= resultPaneStart && msg.Y >= 1 && msg.Y <= m.Height-2 {
			// Calculate which result line was clicked (accounting for viewport offset)
			clickedLine := msg.Y - 1 + m.ResultViewport.YOffset
			if clickedLine >= 0 && clickedLine < len(m.Results) && m.Results[clickedLine] != "" {
				// Save state before inserting ans reference
				m.saveState()
				
				// Insert ans reference at current cursor position
				ansRef := fmt.Sprintf("ans%d", clickedLine+1)

				currentValue := m.Inputs[m.Focused].Value()
				cursorPos := m.Inputs[m.Focused].Position()
				newValue := currentValue[:cursorPos] + ansRef + currentValue[cursorPos:]
				m.Inputs[m.Focused].SetValue(newValue)
				m.Inputs[m.Focused].SetCursor(cursorPos + len(ansRef))

				// Trigger async recalculation for current and dependent lines
				currentExpr := m.Inputs[m.Focused].Value()
				if !m.Calculating[m.Focused] && currentExpr != "" {
					m.Calculating[m.Focused] = true
					cmds = append(cmds, CalculateCmd(currentExpr, m.Results, m.Focused))
				}
				m.updateViewports()
			}
		} else if msg.X < resultPaneStart && msg.Y >= 1 && msg.Y <= m.Height-2 {
			// Check if click is in input pane area
			clickedLine := msg.Y - 1 + m.InputViewport.YOffset
			if clickedLine >= 0 && clickedLine < len(m.Inputs) {
				// Change focus to clicked line
				m.Inputs[m.Focused].Blur()
				m.Focused = clickedLine
				m.Inputs[m.Focused].Focus()
				
				// Calculate cursor position based on click location
				// The gutter has: line number (2 chars) + "│" (1 char) + " " (1 char) = 4 base chars
				gutterWidth := 4
				inputValue := m.Inputs[m.Focused].Value()
				
				if msg.X >= gutterWidth {
					// Click is in the input area, calculate position
					// Subtract 2 to account for cursor being offset to the right
					clickPos := msg.X - gutterWidth - 2
					
					// Clamp to valid cursor positions (0 to length of input)
					if clickPos >= len(inputValue) {
						// Click beyond input text, place cursor at end
						m.Inputs[m.Focused].SetCursor(len(inputValue))
					} else if clickPos < 0 {
						// Safety check, place cursor at start
						m.Inputs[m.Focused].SetCursor(0)
					} else {
						// Click within input text, place cursor at click position
						m.Inputs[m.Focused].SetCursor(clickPos)
					}
				} else {
					// Click in gutter area, place cursor at end of line
					m.Inputs[m.Focused].SetCursor(len(inputValue))
				}
				
				m.updateViewports()
				m.scrollToFocused()
			}
		}
	}

	return *m, tea.Batch(cmds...)
}

// handleKeyMessage handles keyboard input
func (m *Model) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle completions first
	if m.ShowCompletions {
		return m.handleCompletionKeys(msg)
	}

	// Handle help popup first
	if m.ShowHelp {
		return m.handleHelpKeys(msg)
	}

	// Handle go-to-line dialog
	if m.ShowGoToLine {
		return m.handleGoToLineKeys(msg)
	}

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return *m, tea.Quit

	case tea.KeyCtrlH:
		return m.openHelp()

	case tea.KeyCtrlR:
		return m.insertSymbol("√")

	case tea.KeyCtrlA:
		return m.insertSymbol("ans")

	case tea.KeyCtrlT:
		return m.pasteInputTemplate()

	case tea.KeyCtrlD:
		return m.deleteLine()

	case tea.KeyCtrlN:
		return m.clearAll()
		
	case tea.KeyCtrlL:
		return m.openGoToLine()
		
	case tea.KeyCtrlZ:
		// Undo
		if m.undo() {
			return *m, nil
		}
		return *m, nil
		
	case tea.KeyCtrlY:
		// Redo (Ctrl+Y)
		if m.redo() {
			return *m, nil
		}
		return *m, nil
		
	case tea.KeyCtrlS:
		// Copy result of focused line (Ctrl+S)
		return m.copyFocusedResult()
	}

	// Handle Ctrl+P for π symbol
	if msg.Type == tea.KeyCtrlP && !m.ShowCompletions {
		return m.insertSymbol("π")
	}

	// Handle Ctrl+Space for content assist
	if msg.Type == tea.KeyCtrlAt || msg.String() == "\x00" {
		return m.showContentAssist()
	}

	switch msg.Type {
	case tea.KeyTab:
		return m.showCompletions()

	case tea.KeyBackspace:
		if m.Inputs[m.Focused].Value() == "" && len(m.Inputs) > 1 {
			return m.deleteLine()
		}

	case tea.KeyEnter:
		return m.createNewLine()

	case tea.KeyUp:
		return m.focusPreviousLine()

	case tea.KeyDown:
		return m.focusNextLine()

	case tea.KeyPgUp:
		return m.focusFirstLine()

	case tea.KeyPgDown:
		return m.focusLastLine()
	}

	return *m, tea.Batch(cmds...)
}

// handleCompletionKeys handles keyboard input when completions are showing
func (m *Model) handleCompletionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyEsc:
		m.ShowCompletions = false
		m.updateViewports()
		return *m, nil

	case tea.KeyEnter, tea.KeyTab, tea.KeyCtrlY:
		if len(m.Completions) > 0 && m.SelectedCompletion < len(m.Completions) {
			// Insert selected completion
			m.insertCompletion(m.Completions[m.SelectedCompletion])
			m.ShowCompletions = false
			m.LastCompletionQuery = ""
			m.updateViewports()
			cmds = m.triggerCalculationIfNeeded()
		}
		return *m, tea.Batch(cmds...)

	case tea.KeyUp:
		if m.SelectedCompletion > 0 {
			m.SelectedCompletion--
		}
		m.updateViewports()
		return *m, nil

	case tea.KeyDown:
		if m.SelectedCompletion < len(m.Completions)-1 {
			m.SelectedCompletion++
		}
		m.updateViewports()
		return *m, nil

	default:
		// Filter completions on any other key press while showing completions
		var cmd tea.Cmd
		m.Inputs[m.Focused], cmd = m.Inputs[m.Focused].Update(msg)
		cmds = append(cmds, cmd)

		// Re-filter completions based on new input
		currentValue := m.Inputs[m.Focused].Value()
		cursorPos := m.Inputs[m.Focused].Position()

		// Get current word being typed
		wordStart := cursorPos
		for wordStart > 0 && currentValue[wordStart-1] != ' ' && !slices.Contains(operators, string(currentValue[wordStart-1])) {
			wordStart--
		}
		currentWord := currentValue[wordStart:cursorPos]

		// Only re-filter if query changed
		if currentWord != m.LastCompletionQuery {
			cmds = append(cmds, FilterCompletionsCmd(currentWord, m.Results))
		}

		// Trigger calculation
		currentExpr := m.Inputs[m.Focused].Value()
		if !m.Calculating[m.Focused] && currentExpr != "" {
			m.Calculating[m.Focused] = true
			cmds = append(cmds, CalculateCmd(currentExpr, m.Results, m.Focused))
		} else if currentExpr == "" {
			// Clear result when input is empty
			m.Results[m.Focused] = ""
			m.updateViewports()
		}

		return *m, tea.Batch(cmds...)
	}
}

// handleHelpKeys handles keyboard input when help is showing
func (m *Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.ShowHelp = false
		return *m, nil

	case tea.KeyUp:
		m.HelpViewport.LineUp(1)
		return *m, nil

	case tea.KeyDown:
		m.HelpViewport.LineDown(1)
		return *m, nil

	case tea.KeyPgUp:
		m.HelpViewport.HalfViewUp()
		return *m, nil

	case tea.KeyPgDown:
		m.HelpViewport.HalfViewDown()
		return *m, nil
	}

	// Update the help viewport with the message
	var cmd tea.Cmd
	m.HelpViewport, cmd = m.HelpViewport.Update(msg)
	return *m, cmd
}

// handleGoToLineKeys handles keyboard input when go-to-line dialog is showing
func (m *Model) handleGoToLineKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		return m.cancelGoToLine()
		
	case tea.KeyEnter:
		return m.goToLine()
		
	default:
		// Update the go-to-line input with the key
		var cmd tea.Cmd
		m.GoToLineInput, cmd = m.GoToLineInput.Update(msg)
		return *m, cmd
	}
}