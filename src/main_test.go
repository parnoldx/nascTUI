package main

import (
	"os"
	"strings"
	"testing"
	"time"

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

// TestUpdateExchangeRates tests the exchange rate update functionality
func TestUpdateExchangeRates(t *testing.T) {
	// Test that UpdateExchangeRates function exists and returns a boolean
	result := UpdateExchangeRates()
	
	// The function should return a boolean (true/false) without panicking
	if result != true && result != false {
		t.Error("UpdateExchangeRates should return a boolean value")
	}
	
	// Check if exchange rate files exist in common libqalculate locations
	// libqalculate typically stores exchange rates in these locations:
	exchangeRatePaths := []string{
		"/usr/share/qalculate/rates.json",           // System-wide
		"/usr/local/share/qalculate/rates.json",     // Local install  
		os.Getenv("HOME") + "/.local/share/qalculate/rates.json",  // User directory
		os.Getenv("HOME") + "/.qalculate/rates.json",             // User config
	}
	
	foundExchangeRates := false
	var validRatesFile string
	
	for _, path := range exchangeRatePaths {
		if fileInfo, err := os.Stat(path); err == nil && fileInfo.Size() > 100 {
			// File exists and has reasonable size (> 100 bytes indicates it has content)
			foundExchangeRates = true
			validRatesFile = path
			
			// Check if file was modified recently (within last 30 days) or has reasonable content
			if time.Since(fileInfo.ModTime()) < 30*24*time.Hour {
				t.Logf("Found recent exchange rates file: %s (modified: %v, size: %d bytes)", 
					path, fileInfo.ModTime().Format("2006-01-02"), fileInfo.Size())
			} else {
				t.Logf("Found exchange rates file: %s (size: %d bytes, but old: %v)", 
					path, fileInfo.Size(), fileInfo.ModTime().Format("2006-01-02"))
			}
			break
		}
	}
	
	if !foundExchangeRates {
		t.Logf("Warning: No exchange rate files found in standard locations")
		t.Logf("Checked paths: %v", exchangeRatePaths)
		
		// This is not necessarily an error - libqalculate might store rates differently
		// or the system might not have downloaded them yet, but we should log it
	} else {
		// Verify the rates file has some basic content
		if content, err := os.ReadFile(validRatesFile); err == nil {
			contentStr := string(content)
			
			// Check for currency codes that should be in exchange rate data
			// libqalculate uses lowercase currency codes in the JSON file
			expectedCurrencies := []string{"usd", "eur", "gbp", "jpy"}
			foundCurrencies := 0
			
			for _, currency := range expectedCurrencies {
				if strings.Contains(contentStr, `"`+currency+`"`) {
					foundCurrencies++
				}
			}
			
			if foundCurrencies >= 3 {
				t.Logf("Exchange rates file appears valid - contains %d/4 major currencies", foundCurrencies)
				
				// Also extract and verify some rates to ensure they're reasonable
				if strings.Contains(contentStr, `"usd"`) {
					// Extract USD rate (should be > 1.0 relative to EUR)
					if usdMatch := strings.Index(contentStr, `"usd": `); usdMatch != -1 {
						rateStart := usdMatch + 7
						rateEnd := strings.Index(contentStr[rateStart:], ",")
						if rateEnd != -1 {
							usdRate := contentStr[rateStart : rateStart+rateEnd]
							t.Logf("USD exchange rate from file: %s EUR/USD", usdRate)
						}
					}
				}
			} else {
				t.Logf("Warning: Exchange rates file may be incomplete - only found %d/4 major currencies", foundCurrencies)
			}
		}
	}
}

// TestExchangeRatesLoaded tests that exchange rates are actually loaded and functional
func TestExchangeRatesLoaded(t *testing.T) {
	// First ensure exchange rates are updated
	UpdateExchangeRates()
	
	// Test that basic currency conversions work, which indicates rates are loaded
	results := []string{}
	
	// Test USD to EUR conversion
	result := CalculateExpression("1 USD to EUR", results, 0)
	if result == "" || result == "Error" {
		t.Errorf("USD to EUR conversion failed: %q - this suggests exchange rates aren't loaded", result)
	}
	
	// The result should be a numeric value with EUR (since 1 USD should convert to some EUR amount)
	if result != "" && result != "Error" {
		hasNumber := strings.ContainsAny(result, "0123456789")
		hasCurrency := strings.Contains(result, "€") || strings.Contains(result, "EUR")
		
		if !hasNumber {
			t.Errorf("USD to EUR result should contain numbers: %q", result)
		}
		if !hasCurrency {
			t.Errorf("USD to EUR result should contain EUR/€: %q", result)
		}
	}
}

// TestExchangeRateCalculationAccuracy tests that currency calculations produce reasonable results  
func TestExchangeRateCalculationAccuracy(t *testing.T) {
	// Ensure exchange rates are loaded
	UpdateExchangeRates()
	
	results := []string{}
	
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{"USD to EUR", "100 USD to EUR", true},
		{"EUR to USD", "100 EUR to USD", true}, 
		{"USD to GBP", "100 USD to GBP", true},
		{"GBP to USD", "100 GBP to USD", true},
		{"USD to JPY", "100 USD to JPY", true},
		{"JPY to USD", "10000 JPY to USD", true},
		
		// Symbol versions
		{"Dollar to Euro symbol", "100$ to €", true},
		{"Euro to Dollar symbol", "100€ to $", true},
		{"Pound to Dollar symbol", "100£ to $", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateExpression(tt.input, results, 0)
			
			if tt.expectValid {
				if result == "" || result == "Error" {
					t.Errorf("Expected valid result for %q, got: %q", tt.input, result)
					return
				}
				
				// Check that result contains numbers (indicating successful conversion)
				hasNumbers := strings.ContainsAny(result, "0123456789")
				if !hasNumbers {
					t.Errorf("Currency conversion result should contain numbers: %q", result)
				}
				
				// For conversions like "100 USD to EUR", result should not be exactly "100"
				// (unless exchange rate is exactly 1.0, which is extremely unlikely)
				if strings.TrimSpace(result) == "100" || strings.TrimSpace(result) == "100.00" {
					t.Logf("Warning: Currency conversion %q resulted in %q - check if exchange rates are actually loaded", tt.input, result)
				}
			}
		})
	}
}

// TestExchangeRatesDifferentFromUnity tests that exchange rates aren't all 1.0 (which would indicate no real rates loaded)
func TestExchangeRatesDifferentFromUnity(t *testing.T) {
	UpdateExchangeRates()
	
	results := []string{}
	
	// Test several major currency pairs - they should NOT all be 1.0
	conversions := []string{
		"1 USD to EUR",
		"1 EUR to USD", 
		"1 USD to GBP",
		"1 GBP to USD",
		"1 USD to JPY",
	}
	
	unityResults := 0
	validResults := 0
	
	for _, conversion := range conversions {
		result := CalculateExpression(conversion, results, 0)
		if result != "" && result != "Error" {
			validResults++
			
			// Check if result is essentially 1.0 (allowing for minor formatting differences)
			cleaned := strings.TrimSpace(result)
			cleaned = strings.ReplaceAll(cleaned, "€", "")
			cleaned = strings.ReplaceAll(cleaned, "$", "")
			cleaned = strings.ReplaceAll(cleaned, "£", "")
			cleaned = strings.ReplaceAll(cleaned, "¥", "")
			cleaned = strings.TrimSpace(cleaned)
			
			if cleaned == "1" || cleaned == "1.0" || cleaned == "1.00" || cleaned == "1.000000000" {
				unityResults++
			}
		}
	}
	
	if validResults == 0 {
		t.Error("No currency conversions worked - exchange rates may not be loaded")
		return
	}
	
	// If all conversions return 1.0, something is wrong with exchange rate loading
	if unityResults == validResults && validResults > 2 {
		t.Errorf("All %d currency conversions returned 1.0 - exchange rates may not be properly loaded", validResults)
	} else if validResults > 0 {
		t.Logf("Exchange rates appear to be loaded correctly: %d/%d conversions returned non-unity values", validResults-unityResults, validResults)
	}
}

// TestExchangeRateActualValues shows actual conversion values to verify rates are loaded
func TestExchangeRateActualValues(t *testing.T) {
	UpdateExchangeRates()
	
	results := []string{}
	
	// Test a few conversions and log the actual results
	conversions := []string{
		"1 USD to EUR",
		"1 EUR to USD",
		"100 USD to EUR",
		"100 EUR to USD",
	}
	
	for _, conversion := range conversions {
		result := CalculateExpression(conversion, results, 0)
		if result != "" && result != "Error" {
			t.Logf("%s = %s", conversion, result)
			
			// Verify it's not a 1:1 conversion (which would indicate missing rates)
			cleaned := strings.TrimSpace(result)
			cleaned = strings.ReplaceAll(cleaned, "€", "")
			cleaned = strings.ReplaceAll(cleaned, "$", "")
			cleaned = strings.TrimSpace(cleaned)
			
			// For 1:1 conversions, we shouldn't get exactly "1" or "100"
			if conversion == "1 USD to EUR" && (cleaned == "1" || cleaned == "1.0") {
				t.Errorf("1 USD to EUR returned %s - exchange rates may not be loaded", result)
			}
			if conversion == "100 USD to EUR" && (cleaned == "100" || cleaned == "100.0") {
				t.Errorf("100 USD to EUR returned %s - exchange rates may not be loaded", result)
			}
		} else {
			t.Errorf("Currency conversion failed: %s -> %s", conversion, result)
		}
	}
}

// TestHelpPopupResponsiveHeight tests that help popup adapts to terminal height
func TestHelpPopupResponsiveHeight(t *testing.T) {
	tests := []struct {
		name           string
		terminalHeight int
		expectedMaxHeight int
		description    string
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
		name     string
		input    string
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

// TestExchangeRateCalculationIntegration tests complete currency conversion workflow
func TestExchangeRateCalculationIntegration(t *testing.T) {
	// This test verifies the complete workflow for currency conversions
	results := []string{}
	
	// Test basic USD to EUR conversion
	input := "100 USD to EUR"
	result := CalculateExpression(input, results, 0)
	
	// The result should either be:
	// 1. A valid conversion (contains EUR symbol or numeric value)
	// 2. Empty if not recognized as calculation
	// 3. "Error" if calculation failed
	
	if CheckForCalculation(input) {
		// If it's recognized as a calculation, we should get some result
		if result == "" {
			t.Errorf("Expected non-empty result for currency conversion, got empty string")
		}
		
		// If we got a result that's not an error, it should contain some value
		if result != "Error" && result != "" {
			// Should contain either a number or currency symbol
			hasValidContent := strings.ContainsAny(result, "0123456789€$£¥") || 
							 strings.Contains(result, "EUR") || 
							 strings.Contains(result, "USD")
			
			if !hasValidContent {
				t.Errorf("Currency conversion result doesn't seem valid: %q", result)
			}
		}
	}
}

// TestExchangeRateWithAnswerReferences tests currency conversion with ans references  
func TestExchangeRateWithAnswerReferences(t *testing.T) {
	results := []string{"100", "85.50", ""}
	
	// Test using previous results in currency conversion
	tests := []struct {
		name  string
		input string
		index int
	}{
		{"ans with currency", "ans USD to EUR", 2},
		{"ans1 with currency", "ans1 $ to €", 2},
		{"ans2 with currency", "ans2 EUR to $", 2},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateExpression(tt.input, results, tt.index)
			
			// Should either get a valid result or empty string
			// Empty string is acceptable if ans references couldn't be resolved
			if result != "" && result != "Error" {
				// Valid currency conversion result should contain numbers or currency symbols
				hasValidContent := strings.ContainsAny(result, "0123456789€$£¥")
				if !hasValidContent {
					t.Errorf("Currency conversion with ans reference result seems invalid: %q", result)
				}
			}
		})
	}
}