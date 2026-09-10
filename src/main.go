package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

const defaultPlaceholder = "Press Ctrl+H for help"

type Model struct {
	Inputs              []textinput.Model
	Results             []string
	RawResults          []string
	Focused             int
	Width               int
	Height              int
	InputViewport       viewport.Model
	ResultViewport      viewport.Model
	Theme               Theme
	CalcGens            []int
	ShowCompletions     bool
	Completions         []string
	SelectedCompletion  int
	LastCompletionQuery string
	ShowHelp            bool
	HelpViewport        viewport.Model
	UndoSystem          *UndoSystem
	ShowGoToLine        bool
	GoToLineInput       textinput.Model
	LastResultContent   string
	SessionName         string
	LastSavedText       string
	ShowPicker          bool
	Picker              PickerModel
}

func textInputWidth(termWidth int) int {
	width := int(float64(termWidth)*0.7) - 6 - 3 // -3 for early scrolling
	if width < 1 {
		return 1
	}
	return width
}

func (m Model) GetTextInputWidth() int {
	return textInputWidth(m.Width)
}

func newLineInput(width int, placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = width
	ti.Prompt = ""
	ti.CharLimit = 0
	return ti
}

func InitialModel() Model {
	terminalWidth, terminalHeight, _ := term.GetSize(int(os.Stdout.Fd()))

	ti := newLineInput(textInputWidth(terminalWidth), defaultPlaceholder)
	ti.Focus()

	inputVp := viewport.New(int(float64(terminalWidth)*0.7)-2, terminalHeight-2)
	resultVp := viewport.New(int(float64(terminalWidth)*0.3)-2, terminalHeight-2)
	helpVp := viewport.New(0, 0)

	gotoInput := textinput.New()
	gotoInput.Placeholder = ""
	gotoInput.Width = 20
	gotoInput.CharLimit = 5
	gotoInput.Validate = func(s string) error {
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
		RawResults:     []string{""},
		CalcGens:       []int{0},
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
		Picker:         newPicker(),
	}
}

func (m Model) Init() tea.Cmd {
	// Kick off calculations from Update so generation counters land on the live model.
	return tea.Batch(textinput.Blink, func() tea.Msg { return kickoffMsg{} }, autosaveTick())
}

func readStdin() string {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		input, err := io.ReadAll(reader)
		if err == nil {
			return strings.TrimSpace(string(input))
		}
	}
	return ""
}

func (m *Model) ensureLineSlices() {
	n := len(m.Inputs)
	for len(m.Results) < n {
		m.Results = append(m.Results, "")
	}
	for len(m.RawResults) < n {
		m.RawResults = append(m.RawResults, "")
	}
	for len(m.CalcGens) < n {
		m.CalcGens = append(m.CalcGens, 0)
	}
	if len(m.Results) > n {
		m.Results = m.Results[:n]
	}
	if len(m.RawResults) > n {
		m.RawResults = m.RawResults[:n]
	}
	if len(m.CalcGens) > n {
		m.CalcGens = m.CalcGens[:n]
	}
}

func (m *Model) appendLine(value string) int {
	ti := newLineInput(m.GetTextInputWidth(), "")
	ti.SetValue(value)
	ti.CursorEnd()
	m.Inputs = append(m.Inputs, ti)
	m.Results = append(m.Results, "")
	m.RawResults = append(m.RawResults, "")
	m.CalcGens = append(m.CalcGens, 0)
	return len(m.Inputs) - 1
}

func (m *Model) focusLine(index int) {
	if len(m.Inputs) == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.Inputs) {
		index = len(m.Inputs) - 1
	}
	for i := range m.Inputs {
		if i == index {
			m.Inputs[i].Focus()
		} else {
			m.Inputs[i].Blur()
		}
	}
	m.Focused = index
	m.updateViewports()
}

// addMultipleInputs appends non-empty lines. Calculations are started by the
// returned command so Update() never blocks on libqalculate.
func (m *Model) addMultipleInputs(content string) tea.Cmd {
	if content == "" {
		return nil
	}

	m.saveState()

	firstNew := -1
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := m.appendLine(line)
		if firstNew == -1 {
			firstNew = idx
		}
	}

	if len(m.Inputs) == 0 {
		ti := newLineInput(m.GetTextInputWidth(), defaultPlaceholder)
		ti.Focus()
		m.Inputs = []textinput.Model{ti}
		m.Results = []string{""}
		m.RawResults = []string{""}
		m.CalcGens = []int{0}
		m.Focused = 0
		return nil
	}

	m.focusLine(len(m.Inputs) - 1)
	if firstNew >= 0 {
		return m.startCalculationChain(firstNew)
	}
	return nil
}

func (m *Model) startCalculation(index int) tea.Cmd {
	if index < 0 || index >= len(m.Inputs) {
		return nil
	}
	m.ensureLineSlices()

	expr := m.Inputs[index].Value()
	if strings.TrimSpace(expr) == "" {
		m.Results[index] = ""
		m.RawResults[index] = ""
		m.CalcGens[index]++
		return m.startCalculationChain(index + 1)
	}

	m.CalcGens[index]++
	raw := append([]string(nil), m.RawResults...)
	return CalculateCmd(expr, raw, index, m.CalcGens[index])
}

func (m *Model) startCalculationChain(from int) tea.Cmd {
	for i := from; i < len(m.Inputs); i++ {
		if strings.TrimSpace(m.Inputs[i].Value()) != "" {
			return m.startCalculation(i)
		}
		if i < len(m.Results) {
			m.Results[i] = ""
			m.RawResults[i] = ""
		}
	}
	return nil
}

var version = "dev" // set at build time via -ldflags

// loadSessionContent replaces the whole sheet with the stored lines. Blank lines
// are kept so ans1/ans2 references keep pointing at the same rows.
func (m *Model) loadSessionContent(name string, lines []string) tea.Cmd {
	if len(lines) == 0 {
		lines = []string{""}
	}

	width := m.GetTextInputWidth()
	m.Inputs = make([]textinput.Model, len(lines))
	for i, line := range lines {
		placeholder := ""
		if len(lines) == 1 && line == "" {
			placeholder = defaultPlaceholder
		}
		ti := newLineInput(width, placeholder)
		ti.SetValue(line)
		m.Inputs[i] = ti
	}

	m.Results = make([]string, len(lines))
	m.RawResults = make([]string, len(lines))
	m.CalcGens = make([]int, len(lines))
	m.SessionName = name
	m.LastSavedText = strings.Join(lines, "\n")
	m.UndoSystem = NewUndoSystem()

	m.focusLine(len(m.Inputs) - 1)
	m.Inputs[m.Focused].CursorEnd()
	m.updateViewports()
	m.scrollToFocused()
	return m.startCalculationChain(0)
}

func (m Model) sessionLines() []string {
	lines := make([]string, len(m.Inputs))
	for i, input := range m.Inputs {
		lines[i] = input.Value()
	}
	return lines
}

// saveSessionNow writes synchronously: on quit a tea.Cmd would race with tea.Quit.
func (m *Model) saveSessionNow() {
	if m.SessionName == "" {
		return
	}
	lines := m.sessionLines()
	if err := SaveSession(m.SessionName, lines); err == nil {
		m.LastSavedText = strings.Join(lines, "\n")
	}
}

func (m *Model) quit() tea.Cmd {
	m.saveSessionNow()
	return tea.Quit
}

// resolveStartupSession decides which session to open, creating none on disk yet.
func resolveStartupSession(name string, wantNew bool) string {
	if name != "" {
		return sanitizeName(name)
	}
	if wantNew {
		return NewSessionName()
	}
	if last, ok := LastSessionName(); ok {
		return last
	}
	return NewSessionName()
}

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	showSessions := flag.Bool("s", false, "Open the session picker on startup")
	newSession := flag.Bool("n", false, "Start a new session")
	showHelp := flag.Bool("help", false, "Print the help text")
	flag.BoolVar(showHelp, "h", false, "Print the help text")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: nasc [-s] [-n] [--help] [--version] [session name]")
	}
	flag.Parse()

	if *showHelp {
		fmt.Print(helpText)
		return
	}

	if *showVersion {
		fmt.Println(version)
		return
	}

	go func() {
		_ = UpdateExchangeRates()
	}()

	initialInput := readStdin()

	model := InitialModel()
	if initialInput != "" {
		// Piped input stays a throwaway scratch sheet and is never persisted.
		_ = model.addMultipleInputs(initialInput)
	} else {
		name := resolveStartupSession(flag.Arg(0), *newSession)
		lines, err := LoadSession(name)
		if err != nil {
			lines = []string{""}
		}
		_ = model.loadSessionContent(name, lines)
		if *showSessions && len(ListSessions()) > 0 {
			_, _ = model.openPicker()
		}
	}

	opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}
	if initialInput != "" {
		if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
			opts = append(opts, tea.WithInput(tty))
		}
	}

	p := tea.NewProgram(model, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
