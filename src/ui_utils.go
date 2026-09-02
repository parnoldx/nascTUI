package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
)

const autosaveInterval = 30 * time.Second

type pasteMsg string
type pasteErrMsg struct{ err error }
type kickoffMsg struct{}
type autosaveMsg struct{}
type sessionSavedMsg struct {
	text string
	err  error
}

func autosaveTick() tea.Cmd {
	return tea.Tick(autosaveInterval, func(time.Time) tea.Msg {
		return autosaveMsg{}
	})
}

func SaveSessionCmd(name string, lines []string) tea.Cmd {
	return func() tea.Msg {
		err := SaveSession(name, lines)
		return sessionSavedMsg{text: strings.Join(lines, "\n"), err: err}
	}
}

func CalculateCmd(expr string, rawResults []string, index, gen int) tea.Cmd {
	return func() tea.Msg {
		display, raw := evaluateExpression(expr, rawResults, index)
		return CalculationMsg{
			Index:     index,
			Gen:       gen,
			Expr:      expr,
			Result:    display,
			RawResult: raw,
		}
	}
}

func OpenCompletionsCmd(query string, results []string) tea.Cmd {
	return func() tea.Msg {
		completions := GetCompletions(query, results)
		return OpenCompletionsMsg{Completions: completions, Query: query}
	}
}

func FilterCompletionsCmd(query string, results []string) tea.Cmd {
	return func() tea.Msg {
		completions := GetCompletions(query, results)
		return FilterCompletionsMsg{Completions: completions, Query: query}
	}
}

func (m *Model) scrollToFocused() {
	focusedLine := m.Focused

	if m.InputViewport.Height <= 0 || m.ResultViewport.Height <= 0 {
		return
	}

	if focusedLine >= m.InputViewport.Height {
		newOffset := focusedLine - m.InputViewport.Height + 1
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

func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) {
	m.Width = msg.Width
	m.Height = msg.Height

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

	viewportHeight := m.Height - 2
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.InputViewport.Height = viewportHeight
	m.ResultViewport.Height = viewportHeight

	fieldWidth := m.GetTextInputWidth()
	for i := range m.Inputs {
		m.Inputs[i].Width = fieldWidth
	}

	if m.ShowHelp {
		m.sizeHelpViewport()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
