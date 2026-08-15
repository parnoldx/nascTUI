package main

import (
	"github.com/charmbracelet/bubbles/textinput"
)

type UndoState struct {
	InputValues []string
	Results     []string
	RawResults  []string
	Focused     int
	CursorPos   int
}

type UndoSystem struct {
	undoStack []UndoState
	redoStack []UndoState
	maxSize   int
}

func NewUndoSystem() *UndoSystem {
	return &UndoSystem{
		undoStack: make([]UndoState, 0),
		redoStack: make([]UndoState, 0),
		maxSize:   50,
	}
}

func (m *Model) createSnapshot() UndoState {
	inputValues := make([]string, len(m.Inputs))
	for i, input := range m.Inputs {
		inputValues[i] = input.Value()
	}

	results := make([]string, len(m.Results))
	copy(results, m.Results)
	rawResults := make([]string, len(m.RawResults))
	copy(rawResults, m.RawResults)

	cursorPos := 0
	if m.Focused >= 0 && m.Focused < len(m.Inputs) {
		cursorPos = m.Inputs[m.Focused].Position()
	}

	return UndoState{
		InputValues: inputValues,
		Results:     results,
		RawResults:  rawResults,
		Focused:     m.Focused,
		CursorPos:   cursorPos,
	}
}

func (m *Model) saveState() {
	if m.UndoSystem == nil {
		return
	}

	m.UndoSystem.undoStack = append(m.UndoSystem.undoStack, m.createSnapshot())
	if len(m.UndoSystem.undoStack) > m.UndoSystem.maxSize {
		m.UndoSystem.undoStack = m.UndoSystem.undoStack[1:]
	}
	m.UndoSystem.redoStack = m.UndoSystem.redoStack[:0]
}

func (m *Model) restoreState(state UndoState) {
	width := m.GetTextInputWidth()
	m.Inputs = make([]textinput.Model, len(state.InputValues))
	for i, value := range state.InputValues {
		ti := newLineInput(width, "")
		if i == 0 && value == "" && len(state.InputValues) == 1 {
			ti.Placeholder = defaultPlaceholder
		}
		ti.SetValue(value)

		if i == state.Focused {
			ti.Focus()
			if state.CursorPos <= len([]rune(value)) {
				ti.SetCursor(state.CursorPos)
			} else {
				ti.CursorEnd()
			}
		} else {
			ti.Blur()
		}
		m.Inputs[i] = ti
	}

	m.Results = make([]string, len(state.Results))
	copy(m.Results, state.Results)
	m.RawResults = make([]string, len(state.RawResults))
	copy(m.RawResults, state.RawResults)
	if len(m.RawResults) < len(m.Inputs) {
		for len(m.RawResults) < len(m.Inputs) {
			m.RawResults = append(m.RawResults, "")
		}
	}

	m.CalcGens = make([]int, len(m.Inputs))
	m.Focused = state.Focused
	if m.Focused >= len(m.Inputs) {
		m.Focused = len(m.Inputs) - 1
	}
	if m.Focused < 0 {
		m.Focused = 0
	}

	m.updateViewports()
	m.scrollToFocused()
}

func (m *Model) undo() bool {
	if m.UndoSystem == nil || len(m.UndoSystem.undoStack) == 0 {
		return false
	}

	m.UndoSystem.redoStack = append(m.UndoSystem.redoStack, m.createSnapshot())
	if len(m.UndoSystem.redoStack) > m.UndoSystem.maxSize {
		m.UndoSystem.redoStack = m.UndoSystem.redoStack[1:]
	}

	lastIndex := len(m.UndoSystem.undoStack) - 1
	state := m.UndoSystem.undoStack[lastIndex]
	m.UndoSystem.undoStack = m.UndoSystem.undoStack[:lastIndex]
	m.restoreState(state)
	return true
}

func (m *Model) redo() bool {
	if m.UndoSystem == nil || len(m.UndoSystem.redoStack) == 0 {
		return false
	}

	m.UndoSystem.undoStack = append(m.UndoSystem.undoStack, m.createSnapshot())
	if len(m.UndoSystem.undoStack) > m.UndoSystem.maxSize {
		m.UndoSystem.undoStack = m.UndoSystem.undoStack[1:]
	}

	lastIndex := len(m.UndoSystem.redoStack) - 1
	state := m.UndoSystem.redoStack[lastIndex]
	m.UndoSystem.redoStack = m.UndoSystem.redoStack[:lastIndex]
	m.restoreState(state)
	return true
}
