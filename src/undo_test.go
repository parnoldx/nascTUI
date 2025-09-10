package main

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

// Helper function to create a test model
func createTestModel() Model {
	ti := textinput.New()
	ti.Width = 40
	ti.Focus()
	
	return Model{
		Inputs:         []textinput.Model{ti},
		Results:        []string{""},
		Calculating:    []bool{false},
		Focused:        0,
		Width:          80,
		Height:         24,
		InputViewport:  viewport.New(50, 20),
		ResultViewport: viewport.New(30, 20),
		Theme:          newTheme(),
		UndoSystem:     NewUndoSystem(),
	}
}

// Test creating snapshots
func TestCreateSnapshot(t *testing.T) {
	model := createTestModel()
	model.Inputs[0].SetValue("test expression")
	model.Inputs[0].SetCursor(5)
	model.Results[0] = "42"
	
	snapshot := model.createSnapshot()
	
	if len(snapshot.InputValues) != 1 {
		t.Errorf("Expected 1 input value, got %d", len(snapshot.InputValues))
	}
	
	if snapshot.InputValues[0] != "test expression" {
		t.Errorf("Expected 'test expression', got '%s'", snapshot.InputValues[0])
	}
	
	if snapshot.Results[0] != "42" {
		t.Errorf("Expected '42', got '%s'", snapshot.Results[0])
	}
	
	if snapshot.Focused != 0 {
		t.Errorf("Expected focused=0, got %d", snapshot.Focused)
	}
	
	if snapshot.CursorPos != 5 {
		t.Errorf("Expected cursor at position 5, got %d", snapshot.CursorPos)
	}
}

// Test basic undo functionality
func TestBasicUndo(t *testing.T) {
	model := createTestModel()
	model.Inputs[0].SetValue("initial")
	
	// Save initial state
	model.saveState()
	
	// Make a change
	model.Inputs[0].SetValue("modified")
	
	// Undo should restore initial state
	success := model.undo()
	if !success {
		t.Error("Undo should have succeeded")
	}
	
	if model.Inputs[0].Value() != "initial" {
		t.Errorf("Expected 'initial' after undo, got '%s'", model.Inputs[0].Value())
	}
}

// Test basic redo functionality
func TestBasicRedo(t *testing.T) {
	model := createTestModel()
	model.Inputs[0].SetValue("initial")
	
	// Save initial state
	model.saveState()
	
	// Make a change
	model.Inputs[0].SetValue("modified")
	
	// Undo
	model.undo()
	
	// Redo should restore modified state
	success := model.redo()
	if !success {
		t.Error("Redo should have succeeded")
	}
	
	if model.Inputs[0].Value() != "modified" {
		t.Errorf("Expected 'modified' after redo, got '%s'", model.Inputs[0].Value())
	}
}

// Test undo/redo with multiple states
func TestMultipleStates(t *testing.T) {
	model := createTestModel()
	
	// Create sequence of states
	states := []string{"state1", "state2", "state3"}
	
	for _, state := range states {
		model.saveState()
		model.Inputs[0].SetValue(state)
	}
	
	// Undo twice
	model.undo() // Should go to state2
	if model.Inputs[0].Value() != "state2" {
		t.Errorf("Expected 'state2' after first undo, got '%s'", model.Inputs[0].Value())
	}
	
	model.undo() // Should go to state1
	if model.Inputs[0].Value() != "state1" {
		t.Errorf("Expected 'state1' after second undo, got '%s'", model.Inputs[0].Value())
	}
	
	// Redo once
	model.redo() // Should go back to state2
	if model.Inputs[0].Value() != "state2" {
		t.Errorf("Expected 'state2' after redo, got '%s'", model.Inputs[0].Value())
	}
}

// Test cursor position preservation
func TestCursorPreservation(t *testing.T) {
	model := createTestModel()
	model.Inputs[0].SetValue("hello world")
	model.Inputs[0].SetCursor(6) // Position after "hello "
	
	// Save state
	model.saveState()
	
	// Modify input and cursor
	model.Inputs[0].SetValue("modified text")
	model.Inputs[0].SetCursor(3)
	
	// Undo should restore both value and cursor
	model.undo()
	
	if model.Inputs[0].Value() != "hello world" {
		t.Errorf("Expected 'hello world' after undo, got '%s'", model.Inputs[0].Value())
	}
	
	if model.Inputs[0].Position() != 6 {
		t.Errorf("Expected cursor at position 6 after undo, got %d", model.Inputs[0].Position())
	}
}

// Test multiple inputs and focus preservation
func TestMultipleInputsAndFocus(t *testing.T) {
	model := createTestModel()
	
	// Add more inputs
	for i := 0; i < 3; i++ {
		ti := textinput.New()
		ti.Width = 40
		model.Inputs = append(model.Inputs, ti)
		model.Results = append(model.Results, "")
		model.Calculating = append(model.Calculating, false)
	}
	
	// Set values and focus
	model.Inputs[0].SetValue("line1")
	model.Inputs[1].SetValue("line2")
	model.Inputs[2].SetValue("line3")
	model.Focused = 1
	model.Inputs[1].Focus()
	model.Inputs[1].SetCursor(3)
	
	// Save state
	model.saveState()
	
	// Modify state
	model.Inputs[1].SetValue("modified")
	model.Focused = 2
	
	// Undo should restore everything
	model.undo()
	
	if model.Inputs[1].Value() != "line2" {
		t.Errorf("Expected 'line2' after undo, got '%s'", model.Inputs[1].Value())
	}
	
	if model.Focused != 1 {
		t.Errorf("Expected focused=1 after undo, got %d", model.Focused)
	}
	
	if model.Inputs[1].Position() != 3 {
		t.Errorf("Expected cursor at position 3 after undo, got %d", model.Inputs[1].Position())
	}
}

// Test results preservation
func TestResultsPreservation(t *testing.T) {
	model := createTestModel()
	
	// Add more results
	model.Results = []string{"result1", "result2", "result3"}
	model.Inputs = make([]textinput.Model, 3)
	model.Calculating = make([]bool, 3)
	
	for i := range model.Inputs {
		model.Inputs[i] = textinput.New()
		model.Inputs[i].Width = 40
	}
	
	// Save state
	model.saveState()
	
	// Modify results
	model.Results[1] = "modified_result"
	
	// Undo should restore results
	model.undo()
	
	if model.Results[1] != "result2" {
		t.Errorf("Expected 'result2' after undo, got '%s'", model.Results[1])
	}
}

// Test stack size limit
func TestStackSizeLimit(t *testing.T) {
	model := createTestModel()
	
	// Fill up the stack beyond max size
	maxSize := model.UndoSystem.maxSize
	for i := 0; i <= maxSize+10; i++ {
		model.saveState()
		model.Inputs[0].SetValue(fmt.Sprintf("state%d", i))
	}
	
	// Stack should be limited to maxSize
	if len(model.UndoSystem.undoStack) > maxSize {
		t.Errorf("Undo stack size %d exceeds maximum %d", len(model.UndoSystem.undoStack), maxSize)
	}
	
	// Should still be able to undo up to maxSize times
	undoCount := 0
	for model.canUndo() {
		model.undo()
		undoCount++
	}
	
	if undoCount != maxSize {
		t.Errorf("Expected to undo %d times, got %d", maxSize, undoCount)
	}
}

// Test redo stack clearing on new action
func TestRedoStackClearing(t *testing.T) {
	model := createTestModel()
	
	// Create states
	model.saveState()
	model.Inputs[0].SetValue("state1")
	
	model.saveState()
	model.Inputs[0].SetValue("state2")
	
	// Undo to enable redo
	model.undo()
	
	if !model.canRedo() {
		t.Error("Should be able to redo after undo")
	}
	
	// New action should clear redo stack
	model.saveState()
	model.Inputs[0].SetValue("new_state")
	
	if model.canRedo() {
		t.Error("Redo stack should be cleared after new action")
	}
}

// Test undo when stack is empty
func TestUndoEmptyStack(t *testing.T) {
	model := createTestModel()
	
	success := model.undo()
	if success {
		t.Error("Undo should fail when stack is empty")
	}
	
	if model.canUndo() {
		t.Error("canUndo should return false when stack is empty")
	}
}

// Test redo when stack is empty
func TestRedoEmptyStack(t *testing.T) {
	model := createTestModel()
	
	success := model.redo()
	if success {
		t.Error("Redo should fail when stack is empty")
	}
	
	if model.canRedo() {
		t.Error("canRedo should return false when stack is empty")
	}
}

// Test state restoration with boundary conditions
func TestStateRestorationBoundaries(t *testing.T) {
	model := createTestModel()
	
	// Test with cursor position beyond input length
	model.Inputs[0].SetValue("short")
	model.Inputs[0].SetCursor(10) // Beyond input length
	
	model.saveState()
	
	// Modify and undo
	model.Inputs[0].SetValue("different text")
	model.undo()
	
	// Cursor should be clamped to input length
	expectedPos := len("short")
	if model.Inputs[0].Position() != expectedPos {
		t.Errorf("Expected cursor at position %d, got %d", expectedPos, model.Inputs[0].Position())
	}
}

// Test integration with saveState method
func TestSaveStateIntegration(t *testing.T) {
	model := createTestModel()
	
	initialStackSize := len(model.UndoSystem.undoStack)
	
	// saveState should add to stack
	model.saveState()
	
	if len(model.UndoSystem.undoStack) != initialStackSize+1 {
		t.Errorf("Expected stack size to increase by 1, got %d", len(model.UndoSystem.undoStack))
	}
	
	// Verify we can undo
	if !model.canUndo() {
		t.Error("Should be able to undo after saveState")
	}
}

// Benchmark undo/redo operations
func BenchmarkUndo(b *testing.B) {
	model := createTestModel()
	
	// Setup some states
	for i := 0; i < 10; i++ {
		model.saveState()
		model.Inputs[0].SetValue(fmt.Sprintf("state%d", i))
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		if model.canUndo() {
			model.undo()
		} else {
			// Reset for continued benchmarking
			for j := 0; j < 5; j++ {
				model.saveState()
				model.Inputs[0].SetValue(fmt.Sprintf("bench_state%d", j))
			}
		}
	}
}

func BenchmarkSaveState(b *testing.B) {
	model := createTestModel()
	model.Inputs[0].SetValue("benchmark test input")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		model.saveState()
	}
}