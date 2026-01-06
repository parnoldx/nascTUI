package main

/*
#cgo pkg-config: libqalculate
#cgo CXXFLAGS: -std=c++11
#cgo LDFLAGS: -lstdc++
#include <stdlib.h>
#include <stdbool.h>

char* calculate_expression(const char* expression);
void free_result(char* result);
void abort_calculation();
bool update_exchange_rates_if_needed();
int get_function_count();
char* get_function_name(int index);
char* get_function_category(int index);
int get_variable_count();
char* get_variable_name(int index);
char* get_variable_category(int index);
*/
import "C"

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// Constants for configuration values
const (
	CalculationTimeout     = 5 * time.Second  // Timeout for calculations
	MinVariableNameLength  = 3                // Minimum length for variable name matching
	ErrorCalculationFailed = "Calculation failed"
	ErrorExpressionInvalid = "Invalid expression"
	ErrorTimeout          = "Calculation timeout"
)

var operators = []string{"+", "-", "*", "/", "=", "(", ")"}

// Cache for libqalculate completions to avoid expensive C calls on every request
var completionsCache struct {
	initialized       bool
	basicFunctions    []string
	advancedFunctions []string
}

type CalculationMsg struct {
	Index  int
	Result string
}

type OpenCompletionsMsg struct {
	Completions []string
	Query       string
}

type FilterCompletionsMsg struct {
	Completions []string
	Query       string
}

// CalculationManager handles calculation state and cancellation
type CalculationManager struct {
	mu         sync.RWMutex
	running    map[int]context.CancelFunc  // index -> cancel function
	results    []string
	calculating []bool
}

// NewCalculationManager creates a new calculation manager
func NewCalculationManager(size int) *CalculationManager {
	return &CalculationManager{
		running:     make(map[int]context.CancelFunc),
		results:     make([]string, size),
		calculating: make([]bool, size),
	}
}

// Resize adjusts the manager for new input count
func (cm *CalculationManager) Resize(newSize int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Cancel all running calculations beyond new size
	for i := newSize; i < len(cm.results); i++ {
		if cancel, exists := cm.running[i]; exists {
			cancel()
			delete(cm.running, i)
		}
	}
	
	// Resize slices
	if newSize > len(cm.results) {
		// Expand
		for i := len(cm.results); i < newSize; i++ {
			cm.results = append(cm.results, "")
			cm.calculating = append(cm.calculating, false)
		}
	} else if newSize < len(cm.results) {
		// Shrink
		cm.results = cm.results[:newSize]
		cm.calculating = cm.calculating[:newSize]
	}
}

// StartCalculation cancels any existing calculation for the index and starts a new one
func (cm *CalculationManager) StartCalculation(index int, expr string) context.Context {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Cancel existing calculation if any
	if cancel, exists := cm.running[index]; exists {
		cancel()
		delete(cm.running, index)
		// Only abort libqalculate if we're cancelling an existing calculation
		C.abort_calculation()
	}
	
	// Create new context for this calculation
	ctx, cancel := context.WithTimeout(context.Background(), CalculationTimeout)
	cm.running[index] = cancel
	cm.calculating[index] = true
	
	return ctx
}

// CompleteCalculation marks a calculation as complete and stores the result
func (cm *CalculationManager) CompleteCalculation(index int, result string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Remove from running map
	if cancel, exists := cm.running[index]; exists {
		cancel()
		delete(cm.running, index)
	}
	
	cm.results[index] = result
	cm.calculating[index] = false
}

// CancelCalculation cancels a specific calculation
func (cm *CalculationManager) CancelCalculation(index int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cancel, exists := cm.running[index]; exists {
		cancel()
		delete(cm.running, index)
	}
	
	cm.calculating[index] = false
}

// GetState returns the current state (thread-safe)
func (cm *CalculationManager) GetState() ([]string, []bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	results := make([]string, len(cm.results))
	calculating := make([]bool, len(cm.calculating))
	
	copy(results, cm.results)
	copy(calculating, cm.calculating)
	
	return results, calculating
}

// IsCalculating checks if a specific index is calculating
func (cm *CalculationManager) IsCalculating(index int) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if index >= 0 && index < len(cm.calculating) {
		return cm.calculating[index]
	}
	return false
}

func CheckForCalculation(input string) bool {
	// Check for null or empty input (after removing spaces)
	if input == "" || strings.ReplaceAll(input, " ", "") == "" {
		return false
	}
	
	// Check for URLs
	if strings.Contains(input, "http://") {
		return false
	}
	
	// Check if contains digits (digit_regex.match equivalent)
	digitRegex := regexp.MustCompile(`\d`)
	if digitRegex.MatchString(input) {
		return true
	}
	
	// Special commands
	if input == "tutorial()" {
		// tutorial() - could implement later
		return false
	}
	
	// Check for operators in enable_calc list (using global operators)
	for _, op := range operators {
		if strings.Contains(input, op) {
			return true
		}
	}
	
	// Check for function usage (function_name + "(")
	basicFunctions, advancedFunctions := getLibqalculateCompletions()
	allFunctions := append(basicFunctions, advancedFunctions...)
	
	for _, fct := range allFunctions {
		if strings.Contains(input, fct+"(") {
			return true
		}
	}
	
	// Check for variable usage (length > MinVariableNameLength)
	_, allVariables := getLibqalculateCompletions()
	for _, variable := range allVariables {
		if len(variable) > MinVariableNameLength && strings.Contains(input, variable) {
			return true
		}
	}
	
	// Check for defined variables (ans references)
	if strings.HasPrefix(input, "ans") {
		return true
	}
	
	// User functions check would go here if we had user-defined functions
	
	return false
}

func prepareString(input string) string {
	result := input

	// Remove comments after "//" or "#"
	if commentPos := strings.Index(result, "//"); commentPos != -1 {
		result = result[:commentPos]
	}
	if commentPos := strings.Index(result, "#"); commentPos != -1 {
		result = result[:commentPos]
	}

	// Replace currency symbols with currency codes
	result = strings.ReplaceAll(result, "€", "EUR")
	result = strings.ReplaceAll(result, "$", "USD")
	result = strings.ReplaceAll(result, "£", "GBP")
	result = strings.ReplaceAll(result, "¥", "JPY")

	return result
}

func prettyPrint(output string) string {
	result := output
	
	// Superscript digit mapping
	superscriptDigits := map[rune]string{
		'0': "⁰", '1': "¹", '2': "²", '3': "³", '4': "⁴", 
		'5': "⁵", '6': "⁶", '7': "⁷", '8': "⁸", '9': "⁹",
	}
	
	// Convert scientific notation like "1.23E-4" to "1.23 × 10⁻⁴"
	eRegex := regexp.MustCompile(`(\d+\.?\d*)E([+-]?\d+)`)
	result = eRegex.ReplaceAllStringFunc(result, func(match string) string {
		parts := eRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		
		base := parts[1]
		exponent := parts[2]
		
		// Convert exponent to superscript
		superscriptExp := ""
		if strings.HasPrefix(exponent, "-") {
			superscriptExp += "⁻"
			exponent = exponent[1:]
		} else if strings.HasPrefix(exponent, "+") {
			exponent = exponent[1:]
		}
		
		for _, digit := range exponent {
			if sup, exists := superscriptDigits[digit]; exists {
				superscriptExp += sup
			}
		}
		
		return base + " × 10" + superscriptExp
	})
	
	// Convert ^ exponent notation to superscript
	caretRegex := regexp.MustCompile(`\^([+-]?\d+)`)
	result = caretRegex.ReplaceAllStringFunc(result, func(match string) string {
		parts := caretRegex.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		
		exponent := parts[1]
		superscriptExp := ""
		
		if strings.HasPrefix(exponent, "-") {
			superscriptExp += "⁻"
			exponent = exponent[1:]
		} else if strings.HasPrefix(exponent, "+") {
			exponent = exponent[1:]
		}
		
		for _, digit := range exponent {
			if sup, exists := superscriptDigits[digit]; exists {
				superscriptExp += sup
			}
		}
		
		return superscriptExp
	})
	
	return result
}

func postString(output string) string {
	result := output
	
	// Replace currency codes back to symbols
	result = strings.ReplaceAll(result, "EUR", "€")
	result = strings.ReplaceAll(result, "USD", "$")
	result = strings.ReplaceAll(result, "GBP", "£")
	result = strings.ReplaceAll(result, "JPY", "¥")
	
	// Remove space before degree symbol
	result = strings.ReplaceAll(result, " °", "°")
	
	// Apply pretty printing
	result = prettyPrint(result)
	
	return result
}

func CalculateExpression(expr string, results []string, currentIndex int) string {
	if expr == "" {
		return ""
	}

	// Easter egg: detect "0/0" or "infinity"
	trimmedExpr := strings.TrimSpace(strings.ToLower(expr))
	if trimmedExpr == "0/0" {
		return "¯\\_(ツ)_/¯"
	}
	if trimmedExpr == "infinity" || trimmedExpr == "inf" {
		return "∞ The void stares back ∞"
	}

	// Check if this input should be calculated
	if !CheckForCalculation(expr) {
		return ""
	}
	
	// Preprocess the input
	processedExpr := prepareString(expr)
	
	// First replace numbered ans (ans1, ans2, etc.) - only from previous lines
	for i := 0; i < currentIndex && i < len(results); i++ {
		ansPattern := fmt.Sprintf("ans%d", i+1)
		if results[i] != "" {
			processedExpr = strings.ReplaceAll(processedExpr, ansPattern, results[i])
		} else {
			processedExpr = strings.ReplaceAll(processedExpr, ansPattern, "0")
		}
	}
	
	// Then replace standalone 'ans' with last non-empty result from previous lines
	ansRegex := regexp.MustCompile(`\bans\b`)
	if ansRegex.MatchString(processedExpr) {
		replaced := false
		for i := currentIndex - 1; i >= 0; i-- {
			if results[i] != "" {
				processedExpr = ansRegex.ReplaceAllString(processedExpr, results[i])
				replaced = true
				break
			}
		}
		// If ans couldn't be replaced (first line or no previous results), replace with 0
		if !replaced {
			processedExpr = ansRegex.ReplaceAllString(processedExpr, "0")
		}
	}
	
	cExpr := C.CString(processedExpr)
	defer C.free(unsafe.Pointer(cExpr))
	
	cResult := C.calculate_expression(cExpr)
	if cResult == nil {
		return ErrorCalculationFailed
	}
	defer C.free_result(cResult)
	
	rawResult := C.GoString(cResult)
	
	// Check for common error patterns in the result
	if rawResult == "" {
		return ErrorExpressionInvalid
	}
	
	trimmedResult := strings.TrimSpace(rawResult)
	
	// Check for libqalculate error indicators
	if strings.Contains(strings.ToLower(trimmedResult), "error") ||
	   strings.Contains(strings.ToLower(trimmedResult), "undefined") ||
	   strings.Contains(strings.ToLower(trimmedResult), "invalid") {
		return trimmedResult // Return the actual error message from libqalculate
	}
	
	// Postprocess the result
	result := postString(trimmedResult)
	return result
}

func CalculateExpressionWithContext(ctx context.Context, expr string, results []string, currentIndex int) string {
	// Check if context was cancelled before starting
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return ErrorTimeout
		}
		return ""
	default:
	}
	
	// For now, just use the regular calculation function
	// The cancellation will be handled at a higher level through the CalculationManager
	return CalculateExpression(expr, results, currentIndex)
}

func UpdateExchangeRates() bool {
	// Update exchange rates if they're older than 7 days
	return bool(C.update_exchange_rates_if_needed())
}

func getLibqalculateCompletions() ([]string, []string) {
	// Return cached results if already initialized
	if completionsCache.initialized {
		return completionsCache.basicFunctions, completionsCache.advancedFunctions
	}
	
	var basicFunctions []string
	var advancedFunctions []string
	
	// Get functions from libqalculate with categories
	functionCount := int(C.get_function_count())
	for i := 0; i < functionCount; i++ {
		cName := C.get_function_name(C.int(i))
		cCategory := C.get_function_category(C.int(i))
		if cName != nil {
			defer C.free_result(cName)
			func_name := C.GoString(cName)
			category := ""
			if cCategory != nil {
				defer C.free_result(cCategory)
				category = C.GoString(cCategory)
			}
			if  func_name == "" || category == "" {
                continue;
    		}

            if category == "Utilities" || category == "Step Functions" || strings.Contains(category, "Utilities/") ||
                strings.Contains(category, "Statistics/") || strings.Contains(category, "Economics/") || strings.Contains(category, "Geometry/") ||
                strings.Contains(category, "Special Functions/") || category == "Combinatorics" || category == "Logical" || category == "Date & Time" ||
                category == "Miscellaneous" || category == "Number Theory/Arithmetics" || category == "Number Theory/Integers" ||
                category == "Number Theory/Number Bases" || category == "Number Theory/Polynomials" || category == "Number Theory/Prime Numbers" ||
                category == "Calculus/Named Integrals" || category == "Economics" || category == "Special Functions"|| 
				category == "Complex Numbers"{
                advancedFunctions = append(advancedFunctions, func_name)
                continue
            } else if category == "Exponents & Logarithms" {
                if func_name == "lambertw" || func_name == "cis" || func_name == "sqrtpi" || func_name == "pow" ||
                    func_name == "exp10" || func_name == "exp2" {
                    advancedFunctions = append(advancedFunctions, func_name)
                    continue
                }
            } else if category == "Matrices & Vectors" {
                if func_name == "export" || func_name == "genvector" || func_name == "load" || func_name == "permanent" ||
                    func_name == "area" || func_name == "matrix2vector" {
                    advancedFunctions = append(advancedFunctions, func_name)
                    continue
                }
			}
			basicFunctions = append(basicFunctions, func_name)
		}
	}
	
	// Get variables from libqalculate with categories
	variableCount := int(C.get_variable_count())
	for i := 0; i < variableCount; i++ {
		cName := C.get_variable_name(C.int(i))
		cCategory := C.get_variable_category(C.int(i))
		if cName != nil {
			defer C.free_result(cName)
			name := C.GoString(cName)
			category := ""
			if cCategory != nil {
				defer C.free_result(cCategory)
				category = C.GoString(cCategory)
			}
			
			if name == "" || category == "" || category == "Temporary" || category == "Unknowns" || category == "Large Numbers" ||
                category == "Small Numbers" {
                continue
            }
			advancedFunctions = append(advancedFunctions, name)
		}
	}
	
	// Cache the results before returning
	completionsCache.basicFunctions = basicFunctions
	completionsCache.advancedFunctions = advancedFunctions
	completionsCache.initialized = true
	
	return basicFunctions, advancedFunctions
}

func GetCompletions(currentInput string, results []string) []string {
	// Get completions from libqalculate with proper categorization
	basicFunctions, advancedFunctions := getLibqalculateCompletions()
	
	// Sort each group alphabetically
	sort.Slice(basicFunctions, func(i, j int) bool {
		return strings.ToLower(basicFunctions[i]) < strings.ToLower(basicFunctions[j])
	})
	sort.Slice(advancedFunctions, func(i, j int) bool {
		return strings.ToLower(advancedFunctions[i]) < strings.ToLower(advancedFunctions[j])
	})
	
	// Add answer references at the beginning (they're most commonly used)
	ansRefs := []string{"ans"}
	if len(results) == 1 {
		ansRefs = []string{}
	}
	for i, result := range results {
		if result != "" && i != (len(results)-1) {
			ansRefs = append(ansRefs, fmt.Sprintf("ans%d", i+1))
		}
	}
	
	// Combine: ans refs, then basic, then advanced
	completions := make([]string, 0, len(ansRefs)+len(basicFunctions)+len(advancedFunctions))
	completions = append(completions, ansRefs...)
	completions = append(completions, basicFunctions...)
	completions = append(completions, advancedFunctions...)
	
	// Filter completions based on current input
	var filtered []string
	r, _ := utf8.DecodeLastRuneInString(currentInput)
	if currentInput == "" || (!unicode.IsLetter(r)) {
		filtered = completions
	} else {
		lastWordStartIndex := strings.LastIndexFunc(currentInput, func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsNumber(r))
		}) + 1
		prefix := currentInput[lastWordStartIndex:]
		for _, comp := range completions {
			if strings.HasPrefix(strings.ToLower(comp), strings.ToLower(prefix)) {
				filtered = append(filtered, comp)
			}
		}
	}
	
	return filtered
}