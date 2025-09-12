package main

import (
	"os"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Custom message types
type pasteMsg string
type pasteErrMsg struct{ err error }
type tickMsg time.Time
type processPasteMsg struct{}

// Paste command - reads clipboard content (fallback for manual paste trigger)
func PasteCmd() tea.Cmd {
	return func() tea.Msg {
		str, err := clipboard.ReadAll()
		if err != nil {
			return pasteErrMsg{err}
		}
		return pasteMsg(str)
	}
}

// tick generates periodic tick messages for terminal size checking
func tick() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// processPasteCmd generates paste processing messages
func processPasteCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return processPasteMsg{}
	})
}

// scrollToFocused scrolls viewports to show the focused line
func (m *Model) scrollToFocused() {
	focusedLine := m.Focused

	// Ensure viewport heights are positive to prevent division by zero or negative calculations
	if m.InputViewport.Height <= 0 || m.ResultViewport.Height <= 0 {
		return
	}

	// Calculate safe scroll offset that doesn't exceed content bounds
	if focusedLine >= m.InputViewport.Height {
		newOffset := focusedLine - m.InputViewport.Height + 1
		// Ensure offset doesn't go beyond available content
		maxOffset := len(m.Inputs) - m.InputViewport.Height
		if maxOffset < 0 {
			maxOffset = 0
		}
		if newOffset > maxOffset {
			newOffset = maxOffset
		}
		if newOffset < 0 {
			newOffset = 0
		}

		m.InputViewport.SetYOffset(newOffset)
		m.ResultViewport.SetYOffset(newOffset)
	} else {
		m.InputViewport.SetYOffset(0)
		m.ResultViewport.SetYOffset(0)
	}
}

// handleWindowResize handles terminal window resize events
func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) {
	m.Width = msg.Width
	m.Height = msg.Height
	
	// Ensure minimum viable viewport widths
	inputWidth := int(float64(m.Width)*0.7) - 2
	if inputWidth < 1 {
		inputWidth = 1
	}
	m.InputViewport.Width = inputWidth
	
	resultWidth := int(float64(m.Width)*0.3) - 2
	if resultWidth < 1 {
		resultWidth = 1
	}
	m.ResultViewport.Width = resultWidth
	
	// Ensure minimum viable viewport heights
	viewportHeight := m.Height - 2
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.InputViewport.Height = viewportHeight
	m.ResultViewport.Height = viewportHeight
	
	// Update input widths with safety check
	// Reduce width by 3 chars to start scrolling before hitting the edge
	for i := range m.Inputs {
		inputFieldWidth := m.InputViewport.Width - 6 - 3  // -3 for early scrolling
		if inputFieldWidth < 1 {
			inputFieldWidth = 1
		}
		m.Inputs[i].Width = inputFieldWidth
	}
}

// handleTickMessage handles periodic tick messages for terminal size checking
func (m *Model) handleTickMessage() (tea.Model, tea.Cmd) {
	// Check for terminal size changes
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && (w != m.Width || h != m.Height) {
		// Terminal size changed, generate WindowSizeMsg
		return *m, tea.Batch(tick(), func() tea.Msg {
			return tea.WindowSizeMsg{Width: w, Height: h}
		})
	}
	return *m, tick()
}

// CalculateCmd creates a command to calculate an expression
func CalculateCmd(expr string, results []string, index int) tea.Cmd {
	return func() tea.Msg {
		result := CalculateExpression(expr, results, index)
		return CalculationMsg{Index: index, Result: result}
	}
}

// OpenCompletionsCmd creates a command to open completions
func OpenCompletionsCmd(query string, results []string) tea.Cmd {
	return func() tea.Msg {
		completions := GetCompletions(query, results)
		return OpenCompletionsMsg{Completions: completions, Query: query}
	}
}

// FilterCompletionsCmd creates a command to filter completions
func FilterCompletionsCmd(query string, results []string) tea.Cmd {
	return func() tea.Msg {
		completions := GetCompletions(query, results)
		return FilterCompletionsMsg{Completions: completions, Query: query}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}