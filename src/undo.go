package main

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// UndoState represents a snapshot of the calculator state for undo/redo
type UndoState struct {
	InputValues []string // Store the actual text values
	Results     []string
	Focused     int
	CursorPos   int // Store cursor position of focused input
}

// UndoSystem manages undo/redo functionality
type UndoSystem struct {
	undoStack []UndoState
	redoStack []UndoState
	maxSize   int
}

// NewUndoSystem creates a new undo system with specified max size
func NewUndoSystem() *UndoSystem {
	return &UndoSystem{
		undoStack: make([]UndoState, 0),
		redoStack: make([]UndoState, 0),
		maxSize:   50, // Keep last 50 states
	}
}

// createSnapshot creates a snapshot of the current model state
func (m *Model) createSnapshot() UndoState {
	inputValues := make([]string, len(m.Inputs))
	for i, input := range m.Inputs {
		inputValues[i] = input.Value()
	}
	
	results := make([]string, len(m.Results))
	copy(results, m.Results)
	
	cursorPos := 0
	if m.Focused >= 0 && m.Focused < len(m.Inputs) {
		cursorPos = m.Inputs[m.Focused].Position()
	}
	
	return UndoState{
		InputValues: inputValues,
		Results:     results,
		Focused:     m.Focused,
		CursorPos:   cursorPos,
	}
}

// saveState saves the current state to undo stack and clears redo stack
func (m *Model) saveState() {
	if m.UndoSystem == nil {
		return
	}
	
	snapshot := m.createSnapshot()
	
	// Add to undo stack
	m.UndoSystem.undoStack = append(m.UndoSystem.undoStack, snapshot)
	
	// Limit stack size
	if len(m.UndoSystem.undoStack) > m.UndoSystem.maxSize {
		m.UndoSystem.undoStack = m.UndoSystem.undoStack[1:]
	}
	
	// Clear redo stack when new action is performed
	m.UndoSystem.redoStack = m.UndoSystem.redoStack[:0]
}

// restoreState restores a snapshot to the model
func (m *Model) restoreState(state UndoState) {
	// Recreate inputs with proper configuration
	m.Inputs = make([]textinput.Model, len(state.InputValues))
	for i, value := range state.InputValues {
		ti := textinput.New()
		ti.Width = m.GetTextInputWidth()
		ti.Prompt = ""
		ti.CharLimit = 0
		ti.SetValue(value)
		
		if i == state.Focused {
			ti.Focus()
			// Set cursor position, ensuring it's within bounds
			if state.CursorPos <= len(value) {
				ti.SetCursor(state.CursorPos)
			} else {
				ti.SetCursor(len(value))
			}
		} else {
			ti.Blur()
		}
		
		m.Inputs[i] = ti
	}
	
	// Restore results
	m.Results = make([]string, len(state.Results))
	copy(m.Results, state.Results)
	
	// Restore calculating state (reset to false for all)
	m.Calculating = make([]bool, len(m.Inputs))
	
	// Restore focus
	m.Focused = state.Focused
	if m.Focused >= len(m.Inputs) {
		m.Focused = len(m.Inputs) - 1
	}
	if m.Focused < 0 {
		m.Focused = 0
	}
	
	// Update viewports
	m.updateViewports()
	m.scrollToFocused()
}

// undo reverts to the previous state
func (m *Model) undo() bool {
	if m.UndoSystem == nil || len(m.UndoSystem.undoStack) == 0 {
		return false
	}
	
	// Save current state to redo stack
	currentState := m.createSnapshot()
	m.UndoSystem.redoStack = append(m.UndoSystem.redoStack, currentState)
	
	// Limit redo stack size
	if len(m.UndoSystem.redoStack) > m.UndoSystem.maxSize {
		m.UndoSystem.redoStack = m.UndoSystem.redoStack[1:]
	}
	
	// Pop from undo stack and restore
	lastIndex := len(m.UndoSystem.undoStack) - 1
	state := m.UndoSystem.undoStack[lastIndex]
	m.UndoSystem.undoStack = m.UndoSystem.undoStack[:lastIndex]
	
	m.restoreState(state)
	return true
}

// redo moves forward to a previously undone state
func (m *Model) redo() bool {
	if m.UndoSystem == nil || len(m.UndoSystem.redoStack) == 0 {
		return false
	}
	
	// Save current state to undo stack
	currentState := m.createSnapshot()
	m.UndoSystem.undoStack = append(m.UndoSystem.undoStack, currentState)
	
	// Limit undo stack size
	if len(m.UndoSystem.undoStack) > m.UndoSystem.maxSize {
		m.UndoSystem.undoStack = m.UndoSystem.undoStack[1:]
	}
	
	// Pop from redo stack and restore
	lastIndex := len(m.UndoSystem.redoStack) - 1
	state := m.UndoSystem.redoStack[lastIndex]
	m.UndoSystem.redoStack = m.UndoSystem.redoStack[:lastIndex]
	
	m.restoreState(state)
	return true
}

// canUndo returns true if undo is possible
func (m *Model) canUndo() bool {
	return m.UndoSystem != nil && len(m.UndoSystem.undoStack) > 0
}

// canRedo returns true if redo is possible
func (m *Model) canRedo() bool {
	return m.UndoSystem != nil && len(m.UndoSystem.redoStack) > 0
}