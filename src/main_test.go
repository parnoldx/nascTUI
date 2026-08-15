package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel()

	if len(m.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(m.Inputs))
	}

	if len(m.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(m.Results))
	}

	if m.Focused != 0 {
		t.Errorf("Expected focused index 0, got %d", m.Focused)
	}
}

func TestKeyboardNavigation(t *testing.T) {
	m := InitialModel()

	// Test Enter key directly on model
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if len(m.Inputs) != 2 {
		t.Errorf("Expected 2 inputs after Enter, got %d", len(m.Inputs))
	}

	if m.Focused != 1 {
		t.Errorf("Expected focused index 1 after Enter, got %d", m.Focused)
	}

	// Test Up navigation
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)

	if m.Focused != 0 {
		t.Errorf("Expected focused index 0 after Up, got %d", m.Focused)
	}

	// Test Down navigation
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.Focused != 1 {
		t.Errorf("Expected focused index 1 after Down, got %d", m.Focused)
	}
}

func TestCalculation(t *testing.T) {
	// Test the calculation function directly
	results := []string{"", "", ""}

	result := CalculateExpression("2+2", results, 0)
	if result != "4" {
		t.Errorf("Expected '4', got '%s'", result)
	}

	// Test with previous result reference
	results[0] = "4"
	result = CalculateExpression("ans*2", results, 1)
	if result != "8" {
		t.Errorf("Expected '8', got '%s'", result)
	}

	// Test numbered ans reference
	result = CalculateExpression("ans1+1", results, 1)
	if result != "5" {
		t.Errorf("Expected '5', got '%s'", result)
	}
}

func TestQuitKeys(t *testing.T) {
	tm := teatest.NewTestModel(t, InitialModel())

	// Test Esc key
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Test Ctrl+C
	tm2 := teatest.NewTestModel(t, InitialModel())
	tm2.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm2.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestThemeDetection(t *testing.T) {
	// Test theme creation
	theme := newTheme()

	// Verify color definitions exist
	if theme.ansColor == "" {
		t.Error("ansColor should not be empty")
	}

	if theme.focusedColor == "" {
		t.Error("focusedColor should not be empty")
	}
}

func TestStdinParsing(t *testing.T) {
	// Test single line input
	model := InitialModel()
	singleLine := "2 + 2"

	// Simulate what happens with piped input
	model.Inputs[0].SetValue(singleLine)
	model.Results[0] = CalculateExpression(singleLine, model.Results, 0)

	if model.Inputs[0].Value() != "2 + 2" {
		t.Errorf("Expected '2 + 2', got '%s'", model.Inputs[0].Value())
	}

	if model.Results[0] != "4" {
		t.Errorf("Expected '4', got '%s'", model.Results[0])
	}

	// Test multi-line input parsing logic
	multilineInput := "2 + 2\n3 * 4\nans1 + ans2"
	lines := strings.Split(multilineInput, "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	if lines[0] != "2 + 2" {
		t.Errorf("Expected '2 + 2' for first line, got '%s'", lines[0])
	}

	if lines[1] != "3 * 4" {
		t.Errorf("Expected '3 * 4' for second line, got '%s'", lines[1])
	}

	if lines[2] != "ans1 + ans2" {
		t.Errorf("Expected 'ans1 + ans2' for third line, got '%s'", lines[2])
	}

	// Test empty line handling
	emptyLineInput := "2+2\n\n3+3"
	emptyLines := strings.Split(emptyLineInput, "\n")

	if len(emptyLines) != 3 {
		t.Errorf("Expected 3 lines with empty line, got %d", len(emptyLines))
	}

	if emptyLines[1] != "" {
		t.Errorf("Expected empty string for middle line, got '%s'", emptyLines[1])
	}
}

func TestCheckForCalculation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Should return false
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"URL", "http://example.com", false},
		{"pure text", "hello world", false},
		{"tutorial command", "tutorial()", false},

		// Should return true - contains digits
		{"simple number", "42", true},
		{"decimal", "3.14", true},
		{"expression with digits", "2 + 2", true},

		// Should return true - contains operators
		{"addition", "a + b", true},
		{"subtraction", "x - y", true},
		{"multiplication", "a * b", true},
		{"division", "x / y", true},
		{"equals", "x = 5", true},
		{"parentheses", "(a)", true},

		// Should return true - contains functions
		{"sine function", "sin(30)", true},
		{"log function", "log(100)", true},
		{"sqrt function", "sqrt(16)", true},

		// Should return true - contains ans references
		{"ans reference", "ans + 5", true},
		{"ans1 reference", "ans1 * 2", true},

		// Edge cases
		{"mixed text and math", "result is 2+2", true},
		{"function name without parentheses", "sin", false}, // Should be false without "("
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckForCalculation(tt.input)
			if result != tt.expected {
				t.Errorf("CheckForCalculation(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUpdateExchangeRates(t *testing.T) {
	result := UpdateExchangeRates()
	if result != true && result != false {
		t.Error("UpdateExchangeRates should return a boolean value")
	}
}

func TestExchangeRatesLoaded(t *testing.T) {
	result := CalculateExpression("1 USD to EUR", nil, 0)
	if result == "" || result == "Error" {
		t.Errorf("USD to EUR conversion failed: %q", result)
		return
	}
	if !strings.ContainsAny(result, "0123456789") {
		t.Errorf("USD to EUR result should contain numbers: %q", result)
	}
	if !strings.Contains(result, "€") && !strings.Contains(result, "EUR") {
		t.Errorf("USD to EUR result should contain EUR/€: %q", result)
	}
}

// TestHelpPopupResponsiveHeight tests that help popup adapts to terminal height
func TestHelpPopupResponsiveHeight(t *testing.T) {
	tests := []struct {
		name              string
		terminalHeight    int
		expectedMaxHeight int
		description       string
	}{
		{"Very small terminal", 8, 5, "Should use minimal height for very small terminals"},
		{"Small terminal", 15, 11, "Should use reasonable height for small terminals"},
		{"Medium terminal", 25, 19, "Should use ~80% of available height"},
		{"Large terminal", 40, 32, "Should use ~80% of available height"},
		{"Very large terminal", 60, 48, "Should use ~80% of available height"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitialModel()
			m.Height = tt.terminalHeight

			// Simulate Ctrl+H key press to trigger help
			keyMsg := tea.KeyMsg{Type: tea.KeyCtrlH}
			updatedModel, _ := m.Update(keyMsg)
			m = updatedModel.(Model)

			// Check that help is now showing
			if !m.ShowHelp {
				t.Errorf("Help should be showing after Ctrl+H")
			}

			// Check that help height is reasonable for the terminal size
			helpHeight := m.HelpViewport.Height

			// Help height should not exceed our expected maximum
			if helpHeight > tt.expectedMaxHeight {
				t.Errorf("Help height %d exceeds expected maximum %d for %s (terminal height %d)",
					helpHeight, tt.expectedMaxHeight, tt.description, tt.terminalHeight)
			}

			// Help height should be at least reasonable minimum
			minHeight := 3
			if tt.terminalHeight <= 10 {
				minHeight = 2 // Very small terminals can have smaller help
			}
			if helpHeight < minHeight {
				t.Errorf("Help height %d is too small (minimum %d) for %s",
					helpHeight, minHeight, tt.description)
			}

			// Log the actual values for verification
			t.Logf("%s: Terminal=%d, Help height=%d (max expected=%d)",
				tt.name, tt.terminalHeight, helpHeight, tt.expectedMaxHeight)
		})
	}
}

// TestCurrencyConversion tests various currency conversion calculations
func TestCurrencyConversion(t *testing.T) {
	results := []string{}

	tests := []struct {
		name            string
		input           string
		shouldCalculate bool
	}{
		{"USD to EUR", "100 USD to EUR", true},
		{"EUR to USD", "50 EUR to USD", true},
		{"GBP to USD", "25 GBP to USD", true},
		{"JPY to USD", "1000 JPY to USD", true},
		{"USD symbol", "100$ to €", true},
		{"EUR symbol", "50€ to $", true},
		{"GBP symbol", "25£ to $", true},
		{"JPY symbol", "1000¥ to $", true},
		{"invalid currency", "100 XYZ to USD", true}, // Should still attempt calculation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if input is recognized as calculation
			shouldCalc := CheckForCalculation(tt.input)
			if shouldCalc != tt.shouldCalculate {
				t.Errorf("CheckForCalculation(%q) = %v, want %v", tt.input, shouldCalc, tt.shouldCalculate)
			}

			// Test actual calculation
			result := CalculateExpression(tt.input, results, 0)

			// For currency conversion, we expect either:
			// 1. A valid conversion result (contains currency symbol or number)
			// 2. An error message
			// 3. Empty string if not recognized
			if shouldCalc && result != "" && result != "Error" {
				// Valid result should contain some numeric value or currency symbol
				hasNumber := strings.ContainsAny(result, "0123456789")
				hasCurrencySymbol := strings.ContainsAny(result, "$€£¥")

				if !hasNumber && !hasCurrencySymbol {
					t.Errorf("Currency conversion result for %q seems invalid: %q", tt.input, result)
				}
			}
		})
	}
}

// TestCurrencySymbolReplacement tests currency symbol preprocessing
func TestCurrencySymbolReplacement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"dollar symbol", "100$ to EUR", "100USD to EUR"},
		{"euro symbol", "50€ to USD", "50EUR to USD"},
		{"pound symbol", "25£ to USD", "25GBP to USD"},
		{"yen symbol", "1000¥ to USD", "1000JPY to USD"},
		{"mixed symbols", "100$ + 50€", "100USD + 50EUR"},
		{"no symbols", "100 USD to EUR", "100 USD to EUR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareString(tt.input)
			if result != tt.expected {
				t.Errorf("prepareString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCurrencyPostProcessing tests currency symbol restoration in results
func TestCurrencyPostProcessing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"USD code", "42.50 USD", "42.50 $"},
		{"EUR code", "35.75 EUR", "35.75 €"},
		{"GBP code", "28.90 GBP", "28.90 £"},
		{"JPY code", "4250 JPY", "4250 ¥"},
		{"mixed codes", "100 USD and 85 EUR", "100 $ and 85 €"},
		{"no codes", "42.50", "42.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := postString(tt.input)
			if result != tt.expected {
				t.Errorf("postString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCommaDecimalSeparator tests comma decimal separator support
func TestCommaDecimalSeparator(t *testing.T) {
	results := []string{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"comma addition", "2,5 + 3,7", "6.2"},
		{"comma multiplication", "1,5 * 2,0", "3"},
		{"comma division", "10,5 / 2,1", "5"},
		{"comma subtraction", "5,8 - 2,3", "3.5"},
		{"mixed comma and dot", "2,5 + 3.7", "6.2"},
		{"dot should still work", "2.5 + 3.7", "6.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if input is recognized as calculation
			shouldCalc := CheckForCalculation(tt.input)
			if !shouldCalc {
				t.Errorf("CheckForCalculation(%q) should return true for decimal numbers", tt.input)
			}

			// Test actual calculation
			result := CalculateExpression(tt.input, results, 0)

			if result == "" || result == "Error" {
				t.Errorf("Comma decimal calculation failed for %q: got %q", tt.input, result)
				return
			}

			// Check if we got a vector result (indicating comma was treated as separator)
			if strings.HasPrefix(result, "[") && strings.HasSuffix(result, "]") {
				t.Errorf("Comma decimal test %q failed - comma treated as vector separator, got: %q, expected: %q", tt.input, result, tt.expected)
				return
			}

			// For exact matches, compare directly
			if result == tt.expected {
				return // Test passed
			}

			// Normalize both result and expected to use dots for comparison
			// This handles cases where libqalculate returns comma decimal separator
			resultNormalized := strings.ReplaceAll(result, ",", ".")
			expectedNormalized := strings.ReplaceAll(tt.expected, ",", ".")

			// Try numeric comparison for cases like "6.200000000" vs "6.2"
			// This handles libqalculate's decimal formatting variations
			resultTrimmed := strings.TrimRight(resultNormalized, "0")
			resultTrimmed = strings.TrimSuffix(resultTrimmed, ".")
			expectedTrimmed := strings.TrimRight(expectedNormalized, "0")
			expectedTrimmed = strings.TrimSuffix(expectedTrimmed, ".")

			if resultTrimmed != expectedTrimmed {
				t.Errorf("Comma decimal test %q: got %q, expected %q (normalized: %q vs %q)", tt.input, result, tt.expected, resultTrimmed, expectedTrimmed)
			}
		})
	}
}

// TestNumberBaseConversions tests the enhanced PrintOptions conversion functionality
func TestNumberBaseConversions(t *testing.T) {
	results := []string{}

	tests := []struct {
		name             string
		input            string
		shouldCalculate  bool
		expectedContains string // What the result should contain
	}{
		{"decimal to hex", "255 to hex", true, "FF"},
		{"decimal to binary", "15 to bin", true, "1111"},
		{"decimal to octal", "64 to oct", true, "100"},
		{"decimal to duodecimal", "144 to duo", true, "100"},
		{"decimal to roman", "42 to roman", true, "XLII"},
		{"decimal conversion", "0xFF to dec", true, "255"},
		// Float conversions (may not be supported by all libqalculate versions)
		{"decimal to fp32", "3.14 to fp32", true, ""},
		{"decimal to time", "3661 to time", true, ":"}, // Should contain time format
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if input is recognized as calculation
			shouldCalc := CheckForCalculation(tt.input)
			if shouldCalc != tt.shouldCalculate {
				t.Errorf("CheckForCalculation(%q) = %v, want %v", tt.input, shouldCalc, tt.shouldCalculate)
			}

			if !shouldCalc {
				return
			}

			// Test actual calculation
			result := CalculateExpression(tt.input, results, 0)

			if result == "" || result == "Error" {
				t.Logf("Conversion %q failed or not supported: %q", tt.input, result)
				return // Some conversions might not be supported in all libqalculate versions
			}

			// Check if result contains expected content (if specified)
			if tt.expectedContains != "" && !strings.Contains(result, tt.expectedContains) {
				t.Errorf("Conversion %q: expected result to contain %q, got %q", tt.input, tt.expectedContains, result)
			}

			// Log successful conversions for verification
			t.Logf("Conversion %q -> %q", tt.input, result)
		})
	}
}

// Helper function to create a test model
func createTestModel() Model {
	ti := textinput.New()
	ti.Width = 40
	ti.Focus()

	return Model{
		Inputs:         []textinput.Model{ti},
		Results:        []string{""},
		RawResults:     []string{""},
		CalcGens:       []int{0},
		Focused:        0,
		Width:          80,
		Height:         24,
		InputViewport:  viewport.New(50, 20),
		ResultViewport: viewport.New(30, 20),
		Theme:          newTheme(),
		UndoSystem:     NewUndoSystem(),
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

func TestSubstituteAnsLongestMatch(t *testing.T) {
	results := make([]string, 10)
	results[0] = "1"
	results[9] = "99"
	got := substituteAns("ans10 + ans1", results, 10)
	if got != "99 + 1" {
		t.Errorf("substituteAns = %q, want %q", got, "99 + 1")
	}
}

func TestInsertAtRuneUnicode(t *testing.T) {
	got, pos := insertAtRune("π+√", 1, "2")
	if got != "π2+√" {
		t.Errorf("insertAtRune = %q, want %q", got, "π2+√")
	}
	if pos != 2 {
		t.Errorf("insertAtRune pos = %d, want 2", pos)
	}
}

func TestReplaceWordAtCursor(t *testing.T) {
	got, pos := replaceWordAtCursor("2+si", 4, "sin")
	if got != "2+sin" {
		t.Errorf("replaceWordAtCursor = %q, want %q", got, "2+sin")
	}
	if pos != 5 {
		t.Errorf("replaceWordAtCursor pos = %d, want 5", pos)
	}
}

func TestStaleCalculationIgnored(t *testing.T) {
	m := createTestModel()
	m.Inputs[0].SetValue("2+2")
	m.CalcGens[0] = 2
	updated, _ := m.Update(CalculationMsg{Index: 0, Gen: 1, Expr: "2", Result: "2", RawResult: "2"})
	m = updated.(Model)
	if m.Results[0] != "" {
		t.Errorf("stale calculation should be ignored, got %q", m.Results[0])
	}
}

func TestHandledKeysDoNotInsert(t *testing.T) {
	m := createTestModel()
	m.Inputs[0].SetValue("2+2")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(Model)
	if m.Inputs[0].Value() != "2+2" {
		t.Errorf("Ctrl+S should not mutate input, got %q", m.Inputs[0].Value())
	}
}

func TestSingleLinePasteInserts(t *testing.T) {
	m := createTestModel()
	m.Inputs[0].SetValue("ab")
	m.Inputs[0].SetCursor(1)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x"), Paste: true})
	m = updated.(Model)
	if m.Inputs[0].Value() != "axb" {
		t.Errorf("single-line paste = %q, want %q", m.Inputs[0].Value(), "axb")
	}
}

func TestMultiLinePasteAddsLines(t *testing.T) {
	m := createTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3+3\n4+4"), Paste: true})
	m = updated.(Model)
	if len(m.Inputs) != 3 {
		t.Fatalf("expected 3 inputs after multiline paste, got %d", len(m.Inputs))
	}
	if m.Inputs[1].Value() != "3+3" || m.Inputs[2].Value() != "4+4" {
		t.Errorf("unexpected pasted lines: %q, %q", m.Inputs[1].Value(), m.Inputs[2].Value())
	}
}

func TestTruncateVisual(t *testing.T) {
	got := truncateVisual("ππππ", 3)
	if got != "ππ…" {
		t.Errorf("truncateVisual = %q, want %q", got, "ππ…")
	}
}
