package main

import (
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
)

func (m *Model) insertCompletion(completion string) {
	m.saveState()
	newValue, newPos := replaceWordAtCursor(m.Inputs[m.Focused].Value(), m.Inputs[m.Focused].Position(), completion)
	m.Inputs[m.Focused].SetValue(newValue)
	m.Inputs[m.Focused].SetCursor(newPos)
}

func (m *Model) insertSymbol(symbol string) (tea.Model, tea.Cmd) {
	m.saveState()
	newValue, newPos := insertAtRune(m.Inputs[m.Focused].Value(), m.Inputs[m.Focused].Position(), symbol)
	m.Inputs[m.Focused].SetValue(newValue)
	m.Inputs[m.Focused].SetCursor(newPos)
	m.updateViewports()
	return *m, m.startCalculation(m.Focused)
}

func (m *Model) sizeHelpViewport() {
	maxHelpHeight := int(float64(m.Height) * 0.8)
	helpHeight := min(maxHelpHeight, m.Height-6)
	if m.Height <= 10 {
		helpHeight = m.Height - 3
	}
	if helpHeight < 1 {
		helpHeight = 1
	}
	helpWidth := min(80, m.Width-4)
	if helpWidth < 30 {
		helpWidth = 30
	}
	m.HelpViewport.Width = helpWidth
	m.HelpViewport.Height = helpHeight
}

func (m *Model) openHelp() (tea.Model, tea.Cmd) {
	m.ShowHelp = true
	m.sizeHelpViewport()
	m.HelpViewport.SetContent(helpText)
	return *m, textinput.Blink
}

func (m *Model) deleteLine() (tea.Model, tea.Cmd) {
	m.saveState()

	if len(m.Inputs) > 1 {
		m.Inputs = append(m.Inputs[:m.Focused], m.Inputs[m.Focused+1:]...)
		m.Results = append(m.Results[:m.Focused], m.Results[m.Focused+1:]...)
		m.RawResults = append(m.RawResults[:m.Focused], m.RawResults[m.Focused+1:]...)
		m.CalcGens = append(m.CalcGens[:m.Focused], m.CalcGens[m.Focused+1:]...)
		m.ensureLineSlices()
		m.focusLine(m.Focused)
		m.updateViewports()
		return *m, m.startCalculationChain(m.Focused)
	}

	m.Inputs[m.Focused].SetValue("")
	m.Inputs[m.Focused].SetCursor(0)
	m.Results[m.Focused] = ""
	m.RawResults[m.Focused] = ""
	m.CalcGens[m.Focused]++
	m.updateViewports()
	return *m, textinput.Blink
}

func (m *Model) clearAll() (tea.Model, tea.Cmd) {
	m.saveState()
	ti := newLineInput(m.GetTextInputWidth(), defaultPlaceholder)
	ti.Focus()

	m.Inputs = []textinput.Model{ti}
	m.Results = []string{""}
	m.RawResults = []string{""}
	m.CalcGens = []int{0}
	m.Focused = 0
	m.updateViewports()
	m.scrollToFocused()
	return *m, textinput.Blink
}

// newSession stores the current sheet, then asks for the name of the new one.
// Without a session (piped input) there is nothing to name, so just clear.
func (m *Model) newSession() (tea.Model, tea.Cmd) {
	if m.SessionName == "" {
		return m.clearAll()
	}

	m.saveSessionNow()
	m.ShowPicker = true
	m.Picker.Query.SetValue("")
	m.Picker.Selected = 0
	m.Picker.reload()
	return m.openNamePrompt(pickerNew, NewSessionName())
}

func (m *Model) showCompletions() (tea.Model, tea.Cmd) {
	currentWord := currentWordAt(m.Inputs[m.Focused].Value(), m.Inputs[m.Focused].Position())
	return *m, OpenCompletionsCmd(currentWord, m.Results)
}

func (m *Model) createNewLine() (tea.Model, tea.Cmd) {
	m.saveState()
	newInput := newLineInput(m.GetTextInputWidth(), "")

	insertIndex := m.Focused + 1
	m.Inputs = append(m.Inputs[:insertIndex], append([]textinput.Model{newInput}, m.Inputs[insertIndex:]...)...)
	m.Results = append(m.Results[:insertIndex], append([]string{""}, m.Results[insertIndex:]...)...)
	m.RawResults = append(m.RawResults[:insertIndex], append([]string{""}, m.RawResults[insertIndex:]...)...)
	m.CalcGens = append(m.CalcGens[:insertIndex], append([]int{0}, m.CalcGens[insertIndex:]...)...)

	m.focusLine(insertIndex)
	m.updateViewports()
	m.scrollToFocused()
	return *m, textinput.Blink
}

func (m *Model) focusPreviousLine() (tea.Model, tea.Cmd) {
	if m.Focused > 0 {
		m.focusLine(m.Focused - 1)
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

func (m *Model) focusNextLine() (tea.Model, tea.Cmd) {
	if m.Focused < len(m.Inputs)-1 {
		m.focusLine(m.Focused + 1)
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

func (m *Model) focusFirstLine() (tea.Model, tea.Cmd) {
	if m.Focused != 0 {
		m.focusLine(0)
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

func (m *Model) focusLastLine() (tea.Model, tea.Cmd) {
	lastIndex := len(m.Inputs) - 1
	if m.Focused != lastIndex {
		m.focusLine(lastIndex)
		m.scrollToFocused()
	}
	return *m, textinput.Blink
}

func (m *Model) pasteInputTemplate() (tea.Model, tea.Cmd) {
	cmd := m.addMultipleInputs(inputTemplate)
	m.updateViewports()
	m.scrollToFocused()
	return *m, tea.Batch(cmd, textinput.Blink)
}

func (m *Model) handleBracketedPaste(pastedContent string) (tea.Model, tea.Cmd) {
	if containsNewline(pastedContent) {
		normalized := strings.ReplaceAll(pastedContent, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "\n")
		cmd := m.addMultipleInputs(normalized)
		m.updateViewports()
		m.scrollToFocused()
		return *m, cmd
	}

	m.saveState()
	newValue, newPos := insertAtRune(m.Inputs[m.Focused].Value(), m.Inputs[m.Focused].Position(), pastedContent)
	m.Inputs[m.Focused].SetValue(newValue)
	m.Inputs[m.Focused].SetCursor(newPos)
	cmd := m.startCalculation(m.Focused)
	m.updateViewports()
	return *m, tea.Batch(cmd, textinput.Blink)
}

func (m *Model) openGoToLine() (tea.Model, tea.Cmd) {
	m.ShowGoToLine = true
	m.GoToLineInput.SetValue("")
	m.GoToLineInput.Focus()
	return *m, textinput.Blink
}

func (m *Model) goToLine() (tea.Model, tea.Cmd) {
	lineInput := strings.TrimSpace(m.GoToLineInput.Value())

	m.ShowGoToLine = false
	m.GoToLineInput.Blur()

	if lineInput == "" {
		return *m, textinput.Blink
	}

	lineNumber, err := strconv.Atoi(lineInput)
	if err != nil || lineNumber < 1 {
		return *m, textinput.Blink
	}

	targetIndex := lineNumber - 1
	if targetIndex >= len(m.Inputs) {
		targetIndex = len(m.Inputs) - 1
	}

	m.focusLine(targetIndex)
	m.updateViewports()
	m.scrollToFocused()
	return *m, textinput.Blink
}

func (m *Model) cancelGoToLine() (tea.Model, tea.Cmd) {
	m.ShowGoToLine = false
	m.GoToLineInput.SetValue("")
	m.GoToLineInput.Blur()
	return *m, textinput.Blink
}

func (m *Model) copyFocusedResult() (tea.Model, tea.Cmd) {
	if m.Focused >= 0 && m.Focused < len(m.Results) && m.Results[m.Focused] != "" {
		_ = clipboard.WriteAll(m.Results[m.Focused])
	}
	return *m, nil
}
