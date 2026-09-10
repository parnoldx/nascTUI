package main

import (
	_ "embed"
	"github.com/charmbracelet/bubbletea"
)

//go:embed help.txt
var helpText string

//go:embed input.txt
var inputTemplate string

// Update handles all UI state updates and message routing.
// Handled keys/mouse/calc messages return immediately so they are not
// also fed into the focused textinput.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pasteMsg:
		return m.handlePasteMessage(string(msg))

	case pasteErrMsg:
		return m, nil

	case autosaveMsg:
		return m.handleAutosave()

	case sessionSavedMsg:
		if msg.err == nil {
			m.LastSavedText = msg.text
		}
		return m, nil

	case kickoffMsg:
		cmd := m.startCalculationChain(0)
		m.updateViewports()
		return m, cmd

	case CalculationMsg:
		return m.handleCalculationMessage(msg)

	case OpenCompletionsMsg:
		return m.handleOpenCompletionsMessage(msg)

	case FilterCompletionsMsg:
		return m.handleFilterCompletionsMessage(msg)

	case tea.MouseMsg:
		if m.ShowPicker {
			return m, nil
		}
		return m.handleMouseMessage(msg)

	case tea.WindowSizeMsg:
		m.handleWindowResize(msg)
		m.updateViewports()
		return m, nil

	case tea.KeyMsg:
		if m.ShowPicker {
			return m.handlePickerKeys(msg)
		}

		if msg.Paste {
			return m.handleBracketedPaste(string(msg.Runes))
		}

		model, cmd, handled := m.handleKeyMessage(msg)
		if handled {
			return model, cmd
		}
		m = model.(Model)

		oldValue := m.Inputs[m.Focused].Value()
		var inputCmd tea.Cmd
		m.Inputs[m.Focused], inputCmd = m.Inputs[m.Focused].Update(msg)
		var calcCmd tea.Cmd
		if m.Inputs[m.Focused].Value() != oldValue {
			calcCmd = m.startCalculation(m.Focused)
		}
		m.updateViewports()
		return m, tea.Batch(cmd, inputCmd, calcCmd)

	default:
		// Blink/cursor messages must reach the textinputs or the cursor
		// never blinks. Blurred inputs ignore them immediately.
		var cmds []tea.Cmd
		for i := range m.Inputs {
			var cmd tea.Cmd
			m.Inputs[i], cmd = m.Inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		var cmd tea.Cmd
		m.GoToLineInput, cmd = m.GoToLineInput.Update(msg)
		cmds = append(cmds, cmd)
		m.Picker.NameInput, cmd = m.Picker.NameInput.Update(msg)
		cmds = append(cmds, cmd)
		m.updateViewports()
		return m, tea.Batch(cmds...)
	}
}
