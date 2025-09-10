package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"golang.org/x/term"
)

const defaultPlaceholder = "Press Ctrl+H for help"

type Model struct {
	Inputs         []textinput.Model
	Results        []string
	Focused        int
	Width          int
	Height         int
	InputViewport  viewport.Model
	ResultViewport viewport.Model
	Theme          Theme
	Calculating    []bool
	ShowCompletions bool
	Completions     []string
	SelectedCompletion int
	LastCompletionQuery string
	ShowHelp       bool
	HelpViewport   viewport.Model
	UndoSystem     *UndoSystem
	ShowGoToLine   bool
	GoToLineInput  textinput.Model
}

func (m Model) GetTextInputWidth() int {
	return int(float64(m.Width)*0.7) - 6
}

func GetTextInputWidth(width int) int {
	return int(float64(width)*0.7) - 6
}

func InitialModel() Model {
	terminalWidth, terminalHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	
	ti := textinput.New()
	ti.Placeholder = defaultPlaceholder
	ti.Focus()
	ti.Width = GetTextInputWidth(terminalWidth)
	ti.Prompt = ""
	ti.CharLimit = 0
	
	inputVp := viewport.New(int(float64(terminalWidth)*0.7)-2, terminalHeight-2)
	resultVp := viewport.New(int(float64(terminalWidth)*0.3)-2, terminalHeight-2)
	helpVp := viewport.New(0, 0)

	// Initialize go-to-line input
	gotoInput := textinput.New()
	gotoInput.Placeholder = ""
	gotoInput.Width = 20
	gotoInput.CharLimit = 5 // Max 5 digits should be enough
	gotoInput.Validate = func(s string) error {
		// Only allow digits
		for _, r := range s {
			if r < '0' || r > '9' {
				return fmt.Errorf("only numbers allowed")
			}
		}
		return nil
	}

	return Model{
		Inputs:         []textinput.Model{ti},
		Results:        []string{""},
		Calculating:    []bool{false},
		Focused:        0,
		Width:          terminalWidth,
		Height:         terminalHeight,
		InputViewport:  inputVp,
		ResultViewport: resultVp,
		HelpViewport:   helpVp,
		Theme:          newTheme(),
		UndoSystem:     NewUndoSystem(),
		ShowGoToLine:   false,
		GoToLineInput:  gotoInput,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, func() tea.Msg { return tickMsg{} })
}

func readStdin() string {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped
		reader := bufio.NewReader(os.Stdin)
		input, err := io.ReadAll(reader)
		if err == nil {
			return strings.TrimSpace(string(input))
		}
	}
	return ""
}

// Add multiple inputs to existing ones
func (m *Model) addMultipleInputs(content string) {
	if content == "" {
		return
	}
	
	// Save state before making changes (only if we actually have content to add)
	m.saveState()
	
	lines := strings.Split(strings.TrimSpace(content), "\n")
	
	for _, line := range lines {
		// Trim whitespace but keep the line content
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		newInput := textinput.New()
		newInput.Placeholder = ""
		newInput.Width = m.GetTextInputWidth()
		newInput.Prompt = ""
		newInput.SetValue(line)
		newInput.SetCursor(len(line))
		
		m.Inputs = append(m.Inputs, newInput)
		m.Results = append(m.Results, "")
		m.Calculating = append(m.Calculating, false)
		
		index := len(m.Results) - 1
		m.Results[index] = CalculateExpression(line, m.Results, index)
	}
	
	// If no inputs were added and we have no existing inputs, create default
	if len(m.Inputs) == 0 {
		ti := textinput.New()
		ti.Placeholder = defaultPlaceholder
		ti.Focus()
		ti.Width = m.GetTextInputWidth()
		ti.Prompt = ""
		ti.CharLimit = 0
		
		m.Inputs = []textinput.Model{ti}
		m.Results = []string{""}
		m.Calculating = []bool{false}
		m.Focused = 0
	} else {
		// Focus on the last added input
		m.Focused = len(m.Inputs) - 1
		for i := range m.Inputs {
			if i == m.Focused {
				m.Inputs[i].Focus()
				m.Inputs[i].SetCursor(len(m.Inputs[i].Value()))
			} else {
				m.Inputs[i].Blur()
			}
		}
	}
}

func main() {
	go func() {
		if UpdateExchangeRates() {
			log.Println("Exchange rates updated successfully")
		}
	}()
	
	// Check for piped input
	initialInput := readStdin()
	
	model := InitialModel()
	if initialInput != "" {
		model.addMultipleInputs(initialInput)
	}
		
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}