package main

/*
#cgo pkg-config: libqalculate
#cgo CXXFLAGS: -std=c++11
#cgo LDFLAGS: -lstdc++
#include <stdlib.h>
#include <stdbool.h>

char* calculate_expression(const char* expression);
void free_result(char* result);
bool update_exchange_rates_if_needed();
char* get_all_functions();
char* get_all_variables();
*/
import "C"

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unsafe"
)

const (
	ErrorCalculationFailed = "Calculation failed"
	ErrorExpressionInvalid = "Invalid expression"
	MinVariableNameLength  = 3
)

var operators = []string{"+", "-", "*", "/", "=", "(", ")"}

var (
	eRegex       = regexp.MustCompile(`(\d+\.?\d*)E([+-]?\d+)`)
	caretRegex   = regexp.MustCompile(`\^([+-]?\d+)`)
	ansWordRegex = regexp.MustCompile(`\bans\b`)
	ansNumRegex  = regexp.MustCompile(`ans(\d+)`)
)

type CalculationMsg struct {
	Index     int
	Gen       int
	Expr      string
	Result    string
	RawResult string
}

type OpenCompletionsMsg struct {
	Completions []string
	Query       string
}

type FilterCompletionsMsg struct {
	Completions []string
	Query       string
}

var completionsCache struct {
	mu                sync.Mutex
	initialized       bool
	basicFunctions    []string
	advancedFunctions []string
	functionNames     []string
	variableNames     []string
}

func CheckForCalculation(input string) bool {
	if strings.TrimSpace(input) == "" {
		return false
	}
	if strings.Contains(input, "http://") || strings.Contains(input, "https://") {
		return false
	}
	if containsDigit(input) {
		return true
	}
	if input == "tutorial()" {
		return false
	}
	for _, op := range operators {
		if strings.Contains(input, op) {
			return true
		}
	}

	basicFunctions, advancedFunctions := getLibqalculateCompletions()
	for _, fct := range basicFunctions {
		if strings.Contains(input, fct+"(") {
			return true
		}
	}
	for _, fct := range advancedFunctions {
		if strings.Contains(input, fct+"(") {
			return true
		}
	}

	completionsCache.mu.Lock()
	variables := completionsCache.variableNames
	completionsCache.mu.Unlock()
	for _, variable := range variables {
		if len(variable) > MinVariableNameLength && strings.Contains(input, variable) {
			return true
		}
	}

	if strings.Contains(input, "ans") {
		return true
	}
	return false
}

func prepareString(input string) string {
	result := input

	if commentPos := strings.Index(result, "//"); commentPos != -1 {
		result = result[:commentPos]
	}
	// "#" starts a comment (hex literals like 0xFF are unaffected)
	if commentPos := strings.Index(result, "#"); commentPos != -1 {
		result = result[:commentPos]
	}

	result = strings.ReplaceAll(result, "€", "EUR")
	result = strings.ReplaceAll(result, "$", "USD")
	result = strings.ReplaceAll(result, "£", "GBP")
	result = strings.ReplaceAll(result, "¥", "JPY")

	return result
}

func prettyPrint(output string) string {
	result := output

	superscriptDigits := map[rune]string{
		'0': "⁰", '1': "¹", '2': "²", '3': "³", '4': "⁴",
		'5': "⁵", '6': "⁶", '7': "⁷", '8': "⁸", '9': "⁹",
	}

	result = eRegex.ReplaceAllStringFunc(result, func(match string) string {
		parts := eRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		base := parts[1]
		exponent := parts[2]

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

	result = strings.ReplaceAll(result, "EUR", "€")
	result = strings.ReplaceAll(result, "USD", "$")
	result = strings.ReplaceAll(result, "GBP", "£")
	result = strings.ReplaceAll(result, "JPY", "¥")
	result = strings.ReplaceAll(result, " °", "°")

	return prettyPrint(result)
}

// substituteAns replaces ans / ansN with previous raw results.
// Numbered references use the full digit run so ans10 is not treated as ans1+"0".
func substituteAns(expr string, results []string, currentIndex int) string {
	matches := ansNumRegex.FindAllStringSubmatchIndex(expr, -1)
	type repl struct {
		start, end int
		value      string
	}
	repls := make([]repl, 0, len(matches))
	for _, loc := range matches {
		n, err := strconv.Atoi(expr[loc[2]:loc[3]])
		if err != nil {
			continue
		}
		idx := n - 1
		if idx < 0 || idx >= currentIndex || idx >= len(results) {
			continue
		}
		val := results[idx]
		if val == "" {
			val = "0"
		}
		repls = append(repls, repl{loc[0], loc[1], val})
	}
	sort.Slice(repls, func(i, j int) bool { return repls[i].start > repls[j].start })
	for _, r := range repls {
		expr = expr[:r.start] + r.value + expr[r.end:]
	}

	if ansWordRegex.MatchString(expr) {
		val := "0"
		for i := currentIndex - 1; i >= 0; i-- {
			if results[i] != "" {
				val = results[i]
				break
			}
		}
		expr = ansWordRegex.ReplaceAllString(expr, val)
	}
	return expr
}

func evaluateExpression(expr string, rawResults []string, currentIndex int) (display string, raw string) {
	if expr == "" {
		return "", ""
	}

	trimmedExpr := strings.TrimSpace(strings.ToLower(expr))
	if trimmedExpr == "0/0" {
		return "¯\\_(ツ)_/¯", ""
	}
	if trimmedExpr == "infinity" || trimmedExpr == "inf" {
		return "∞ The void stares back ∞", ""
	}

	if !CheckForCalculation(expr) {
		return "", ""
	}

	processedExpr := substituteAns(prepareString(expr), rawResults, currentIndex)

	cExpr := C.CString(processedExpr)
	defer C.free(unsafe.Pointer(cExpr))

	cResult := C.calculate_expression(cExpr)
	if cResult == nil {
		return ErrorCalculationFailed, ""
	}
	defer C.free_result(cResult)

	rawResult := C.GoString(cResult)
	if rawResult == "" {
		return ErrorExpressionInvalid, ""
	}

	trimmedResult := strings.TrimSpace(rawResult)
	lowered := strings.ToLower(trimmedResult)
	if strings.Contains(lowered, "error") ||
		strings.Contains(lowered, "undefined") ||
		strings.Contains(lowered, "invalid") {
		return trimmedResult, ""
	}

	return postString(trimmedResult), trimmedResult
}

// CalculateExpression evaluates expr using previous raw results. The returned
// string is the display form (pretty-printed, currency symbols restored).
func CalculateExpression(expr string, results []string, currentIndex int) string {
	display, _ := evaluateExpression(expr, results, currentIndex)
	return display
}

func UpdateExchangeRates() bool {
	return bool(C.update_exchange_rates_if_needed())
}

func parseNameCategoryList(raw string, handle func(name, category string)) {
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		name, category, ok := strings.Cut(line, "\t")
		if !ok || name == "" || category == "" {
			continue
		}
		handle(name, category)
	}
}

func isAdvancedFunction(name, category string) bool {
	if category == "Utilities" || category == "Step Functions" || strings.Contains(category, "Utilities/") ||
		strings.Contains(category, "Statistics/") || strings.Contains(category, "Economics/") || strings.Contains(category, "Geometry/") ||
		strings.Contains(category, "Special Functions/") || category == "Combinatorics" || category == "Logical" || category == "Date & Time" ||
		category == "Miscellaneous" || category == "Number Theory/Arithmetics" || category == "Number Theory/Integers" ||
		category == "Number Theory/Number Bases" || category == "Number Theory/Polynomials" || category == "Number Theory/Prime Numbers" ||
		category == "Calculus/Named Integrals" || category == "Economics" || category == "Special Functions" ||
		category == "Complex Numbers" {
		return true
	}
	if category == "Exponents & Logarithms" {
		switch name {
		case "lambertw", "cis", "sqrtpi", "pow", "exp10", "exp2":
			return true
		}
	}
	if category == "Matrices & Vectors" {
		switch name {
		case "export", "genvector", "load", "permanent", "area", "matrix2vector":
			return true
		}
	}
	return false
}

func getLibqalculateCompletions() ([]string, []string) {
	completionsCache.mu.Lock()
	defer completionsCache.mu.Unlock()

	if completionsCache.initialized {
		return completionsCache.basicFunctions, completionsCache.advancedFunctions
	}

	var basicFunctions []string
	var advancedFunctions []string
	var functionNames []string
	var variableNames []string

	if cFuncs := C.get_all_functions(); cFuncs != nil {
		raw := C.GoString(cFuncs)
		C.free_result(cFuncs)
		parseNameCategoryList(raw, func(name, category string) {
			functionNames = append(functionNames, name)
			if isAdvancedFunction(name, category) {
				advancedFunctions = append(advancedFunctions, name)
			} else {
				basicFunctions = append(basicFunctions, name)
			}
		})
	}

	if cVars := C.get_all_variables(); cVars != nil {
		raw := C.GoString(cVars)
		C.free_result(cVars)
		parseNameCategoryList(raw, func(name, category string) {
			if category == "Temporary" || category == "Unknowns" || category == "Large Numbers" ||
				category == "Small Numbers" {
				return
			}
			variableNames = append(variableNames, name)
			advancedFunctions = append(advancedFunctions, name)
		})
	}

	sort.Slice(basicFunctions, func(i, j int) bool {
		return strings.ToLower(basicFunctions[i]) < strings.ToLower(basicFunctions[j])
	})
	sort.Slice(advancedFunctions, func(i, j int) bool {
		return strings.ToLower(advancedFunctions[i]) < strings.ToLower(advancedFunctions[j])
	})

	completionsCache.basicFunctions = basicFunctions
	completionsCache.advancedFunctions = advancedFunctions
	completionsCache.functionNames = functionNames
	completionsCache.variableNames = variableNames
	completionsCache.initialized = true

	return basicFunctions, advancedFunctions
}

func GetCompletions(currentInput string, results []string) []string {
	basicFunctions, advancedFunctions := getLibqalculateCompletions()

	ansRefs := []string{"ans"}
	if len(results) == 1 {
		ansRefs = []string{}
	}
	for i, result := range results {
		if result != "" && i != (len(results)-1) {
			ansRefs = append(ansRefs, fmt.Sprintf("ans%d", i+1))
		}
	}

	completions := make([]string, 0, len(ansRefs)+len(basicFunctions)+len(advancedFunctions))
	completions = append(completions, ansRefs...)
	completions = append(completions, basicFunctions...)
	completions = append(completions, advancedFunctions...)

	r, ok := lastRune(currentInput)
	if currentInput == "" || !ok || !unicode.IsLetter(r) {
		return completions
	}

	lastWordStartIndex := strings.LastIndexFunc(currentInput, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r))
	}) + 1
	prefix := currentInput[lastWordStartIndex:]

	filtered := make([]string, 0, 16)
	lowerPrefix := strings.ToLower(prefix)
	for _, comp := range completions {
		if strings.HasPrefix(strings.ToLower(comp), lowerPrefix) {
			filtered = append(filtered, comp)
		}
	}
	return filtered
}
