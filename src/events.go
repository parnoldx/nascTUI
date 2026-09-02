package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
)

func (m *Model) handleAutosave() (tea.Model, tea.Cmd) {
	if m.SessionName == "" {
		return *m, autosaveTick()
	}

	lines := m.sessionLines()
	if strings.Join(lines, "\n") == m.LastSavedText {
		return *m, autosaveTick()
	}
	return *m, tea.Batch(SaveSessionCmd(m.SessionName, lines), autosaveTick())
}

func (m *Model) handlePasteMessage(content string) (tea.Model, tea.Cmd) {
	if content == "" {
		return *m, nil
	}
	if containsNewline(content) {
		cmd := m.addMultipleInputs(content)
		m.updateViewports()
		m.scrollToFocused()
		return *m, cmd
	}

	m.saveState()
	newValue, newPos := insertAtRune(m.Inputs[m.Focused].Value(), m.Inputs[m.Focused].Position(), content)
	m.Inputs[m.Focused].SetValue(newValue)
	m.Inputs[m.Focused].SetCursor(newPos)
	cmd := m.startCalculation(m.Focused)
	m.updateViewports()
	return *m, cmd
}

func (m *Model) handleCalculationMessage(msg CalculationMsg) (tea.Model, tea.Cmd) {
	if msg.Index < 0 || msg.Index >= len(m.Inputs) {
		return *m, nil
	}
	m.ensureLineSlices()
	if msg.Gen != m.CalcGens[msg.Index] {
		return *m, nil
	}
	if msg.Expr != m.Inputs[msg.Index].Value() {
		return *m, m.startCalculation(msg.Index)
	}

	m.Results[msg.Index] = msg.Result
	m.RawResults[msg.Index] = msg.RawResult
	m.updateViewports()
	return *m, m.startCalculationChain(msg.Index + 1)
}

func (m *Model) handleOpenCompletionsMessage(msg OpenCompletionsMsg) (tea.Model, tea.Cmd) {
	m.Completions = msg.Completions
	m.LastCompletionQuery = msg.Query

	if len(m.Completions) == 0 {
		m.ShowCompletions = false
		m.updateViewports()
		return *m, nil
	}

	m.ShowCompletions = true
	m.SelectedCompletion = 0
	m.updateViewports()
	return *m, nil
}

func (m *Model) handleFilterCompletionsMessage(msg FilterCompletionsMsg) (tea.Model, tea.Cmd) {
	m.Completions = msg.Completions
	m.LastCompletionQuery = msg.Query

	if len(m.Completions) == 0 {
		m.ShowCompletions = false
	} else if m.SelectedCompletion >= len(m.Completions) {
		m.SelectedCompletion = len(m.Completions) - 1
	}

	m.updateViewports()
	return *m, nil
}

func (m *Model) handleMouseMessage(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.ShowHelp {
		switch msg.Type {
		case tea.MouseWheelUp:
			m.HelpViewport.LineUp(3)
			return *m, nil
		case tea.MouseWheelDown:
			m.HelpViewport.LineDown(3)
			return *m, nil
		}
	}

	if msg.Type != tea.MouseLeft {
		return *m, nil
	}

	resultPaneStart := int(float64(m.Width) * 0.7)
	if msg.X >= resultPaneStart && msg.Y >= 1 && msg.Y <= m.Height-2 {
		clickedLine := msg.Y - 1 + m.ResultViewport.YOffset
		if clickedLine >= 0 && clickedLine < len(m.Results) && m.Results[clickedLine] != "" {
			m.saveState()
			ansRef := fmt.Sprintf("ans%d", clickedLine+1)
			newValue, newPos := insertAtRune(m.Inputs[m.Focused].Value(), m.Inputs[m.Focused].Position(), ansRef)
			m.Inputs[m.Focused].SetValue(newValue)
			m.Inputs[m.Focused].SetCursor(newPos)
			cmd := m.startCalculation(m.Focused)
			m.updateViewports()
			return *m, cmd
		}
		return *m, nil
	}

	if msg.X < resultPaneStart && msg.Y >= 1 && msg.Y <= m.Height-2 {
		clickedLine := msg.Y - 1 + m.InputViewport.YOffset
		if clickedLine >= 0 && clickedLine < len(m.Inputs) {
			m.focusLine(clickedLine)

			const gutterWidth = 4
			inputValue := m.Inputs[m.Focused].Value()
			if msg.X >= gutterWidth {
				clickPos := runeIndexAtVisual(inputValue, msg.X-gutterWidth)
				m.Inputs[m.Focused].SetCursor(clickPos)
			} else {
				m.Inputs[m.Focused].CursorEnd()
			}

			m.updateViewports()
			m.scrollToFocused()
		}
	}

	return *m, nil
}

func (m *Model) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.ShowCompletions {
		model, cmd := m.handleCompletionKeys(msg)
		return model, cmd, true
	}
	if m.ShowHelp {
		model, cmd := m.handleHelpKeys(msg)
		return model, cmd, true
	}
	if m.ShowGoToLine {
		model, cmd := m.handleGoToLineKeys(msg)
		return model, cmd, true
	}

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return *m, m.quit(), true
	case tea.KeyCtrlO:
		model, cmd := m.openPicker()
		return model, cmd, true
	case tea.KeyCtrlH:
		model, cmd := m.openHelp()
		return model, cmd, true
	case tea.KeyCtrlR:
		model, cmd := m.insertSymbol("√")
		return model, cmd, true
	case tea.KeyCtrlA:
		model, cmd := m.insertSymbol("ans")
		return model, cmd, true
	case tea.KeyCtrlT:
		model, cmd := m.pasteInputTemplate()
		return model, cmd, true
	case tea.KeyCtrlD:
		model, cmd := m.deleteLine()
		return model, cmd, true
	case tea.KeyCtrlN:
		model, cmd := m.newSession()
		return model, cmd, true
	case tea.KeyCtrlL:
		model, cmd := m.openGoToLine()
		return model, cmd, true
	case tea.KeyCtrlZ:
		m.undo()
		return *m, nil, true
	case tea.KeyCtrlY:
		m.redo()
		return *m, nil, true
	case tea.KeyCtrlS:
		model, cmd := m.copyFocusedResult()
		return model, cmd, true
	case tea.KeyCtrlP:
		model, cmd := m.insertSymbol("π")
		return model, cmd, true
	case tea.KeyCtrlAt:
		model, cmd := m.showCompletions()
		return model, cmd, true
	case tea.KeyTab:
		model, cmd := m.showCompletions()
		return model, cmd, true
	case tea.KeyBackspace:
		if m.Inputs[m.Focused].Value() == "" && len(m.Inputs) > 1 {
			model, cmd := m.deleteLine()
			return model, cmd, true
		}
		return *m, nil, false
	case tea.KeyEnter:
		model, cmd := m.createNewLine()
		return model, cmd, true
	case tea.KeyUp:
		model, cmd := m.focusPreviousLine()
		return model, cmd, true
	case tea.KeyDown:
		model, cmd := m.focusNextLine()
		return model, cmd, true
	case tea.KeyPgUp:
		model, cmd := m.focusFirstLine()
		return model, cmd, true
	case tea.KeyPgDown:
		model, cmd := m.focusLastLine()
		return model, cmd, true
	}

	if msg.String() == "\x00" {
		model, cmd := m.showCompletions()
		return model, cmd, true
	}

	return *m, nil, false
}

func (m *Model) handleCompletionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.ShowCompletions = false
		m.updateViewports()
		return *m, nil

	case tea.KeyEnter, tea.KeyTab, tea.KeyCtrlY:
		if len(m.Completions) > 0 && m.SelectedCompletion < len(m.Completions) {
			m.insertCompletion(m.Completions[m.SelectedCompletion])
			m.ShowCompletions = false
			m.LastCompletionQuery = ""
			m.updateViewports()
			return *m, m.startCalculation(m.Focused)
		}
		return *m, nil

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
	}

	var cmd tea.Cmd
	m.Inputs[m.Focused], cmd = m.Inputs[m.Focused].Update(msg)

	currentValue := m.Inputs[m.Focused].Value()
	currentWord := currentWordAt(currentValue, m.Inputs[m.Focused].Position())

	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if currentWord != m.LastCompletionQuery {
		cmds = append(cmds, FilterCompletionsCmd(currentWord, m.Results))
	}
	cmds = append(cmds, m.startCalculation(m.Focused))
	return *m, tea.Batch(cmds...)
}

func (m *Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return *m, m.quit()
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

	switch msg.String() {
	case "j":
		m.HelpViewport.LineDown(1)
	case "k":
		m.HelpViewport.LineUp(1)
	case "q":
		m.ShowHelp = false
	}
	return *m, nil
}

func (m *Model) handleGoToLineKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		return m.cancelGoToLine()
	case tea.KeyEnter:
		return m.goToLine()
	default:
		var cmd tea.Cmd
		m.GoToLineInput, cmd = m.GoToLineInput.Update(msg)
		return *m, cmd
	}
}

func containsNewline(s string) bool {
	for _, r := range s {
		if r == '\n' || r == '\r' {
			return true
		}
	}
	return false
}
