package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const pickerVisibleItems = 8

type pickerMode int

const (
	pickerBrowse pickerMode = iota
	pickerNew
	pickerRename
	pickerConfirmDelete
)

type PickerModel struct {
	Query     textinput.Model
	NameInput textinput.Model
	Sessions  []Session
	Filtered  []int
	Selected  int
	Mode      pickerMode
	Status    string
}

func newPicker() PickerModel {
	query := textinput.New()
	query.Prompt = "> "
	query.CharLimit = maxSessionName

	nameInput := textinput.New()
	nameInput.Prompt = ""
	nameInput.CharLimit = maxSessionName

	return PickerModel{Query: query, NameInput: nameInput}
}

func (p *PickerModel) reload() {
	p.Sessions = ListSessions()
	p.applyFilter()
}

func (p *PickerModel) applyFilter() {
	query := p.Query.Value()

	type scored struct {
		index int
		score int
	}
	var matches []scored
	for i, session := range p.Sessions {
		score, ok := fuzzyMatch(query, session.Name)
		if ok {
			matches = append(matches, scored{index: i, score: score})
		}
	}

	sort.SliceStable(matches, func(a, b int) bool {
		return matches[a].score > matches[b].score
	})

	p.Filtered = p.Filtered[:0]
	for _, match := range matches {
		p.Filtered = append(p.Filtered, match.index)
	}
	if p.Selected >= len(p.Filtered) {
		p.Selected = len(p.Filtered) - 1
	}
	if p.Selected < 0 {
		p.Selected = 0
	}
}

func (p PickerModel) selectedSession() (Session, bool) {
	if p.Selected < 0 || p.Selected >= len(p.Filtered) {
		return Session{}, false
	}
	return p.Sessions[p.Filtered[p.Selected]], true
}

// fuzzyMatch scores candidate against a subsequence query. Matches early in the
// name, at word starts, and in unbroken runs score higher.
func fuzzyMatch(query, candidate string) (int, bool) {
	if query == "" {
		return 0, true
	}

	q := []rune(strings.ToLower(query))
	c := []rune(strings.ToLower(candidate))

	score := 0
	matched := 0
	previous := -1

	for i := 0; i < len(c) && matched < len(q); i++ {
		if c[i] != q[matched] {
			continue
		}
		if matched > 0 && i == previous+1 {
			score += 10
		}
		if i == 0 || isNameSeparator(c[i-1]) {
			score += 15
		}
		score -= i - previous - 1
		previous = i
		matched++
	}

	if matched < len(q) {
		return 0, false
	}
	return score, true
}

func isNameSeparator(r rune) bool {
	return r == ' ' || r == '-' || r == '_' || r == '.'
}

func relativeTime(t time.Time) string {
	elapsed := time.Since(t)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	case elapsed < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(elapsed.Hours()/24))
	}
	return t.Format("Jan 02")
}

func (m *Model) openPicker() (tea.Model, tea.Cmd) {
	m.ShowPicker = true
	m.Picker.Mode = pickerBrowse
	m.Picker.Status = ""
	m.Picker.Query.SetValue("")
	m.Picker.Query.Focus()
	m.Picker.Selected = 0
	m.Picker.reload()
	return *m, textinput.Blink
}

func (m *Model) closePicker() (tea.Model, tea.Cmd) {
	m.ShowPicker = false
	m.Picker.Mode = pickerBrowse
	m.Picker.Query.Blur()
	m.Picker.NameInput.Blur()
	return *m, textinput.Blink
}

func (m *Model) startNewSession(name string) (tea.Model, tea.Cmd) {
	m.saveSessionNow()
	cmd := m.loadSessionContent(name, []string{""})
	model, _ := m.closePicker()
	return model, tea.Batch(cmd, textinput.Blink)
}

func (m *Model) switchToSession(name string) tea.Cmd {
	if name == m.SessionName {
		return nil
	}
	m.saveSessionNow()
	lines, err := LoadSession(name)
	if err != nil {
		lines = []string{""}
	}
	return m.loadSessionContent(name, lines)
}

func (m *Model) handlePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Picker.Mode {
	case pickerNew:
		return m.handlePickerNameKeys(msg, m.acceptNewName)
	case pickerRename:
		return m.handlePickerNameKeys(msg, m.acceptRename)
	case pickerConfirmDelete:
		return m.handlePickerDeleteKeys(msg)
	}
	return m.handlePickerBrowseKeys(msg)
}

func (m *Model) handlePickerBrowseKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return *m, m.quit()

	case tea.KeyEsc:
		return m.closePicker()

	case tea.KeyEnter:
		session, ok := m.Picker.selectedSession()
		if !ok {
			return *m, nil
		}
		cmd := m.switchToSession(session.Name)
		model, _ := m.closePicker()
		return model, tea.Batch(cmd, textinput.Blink)

	case tea.KeyUp:
		if m.Picker.Selected > 0 {
			m.Picker.Selected--
		}
		return *m, nil

	case tea.KeyDown:
		if m.Picker.Selected < len(m.Picker.Filtered)-1 {
			m.Picker.Selected++
		}
		return *m, nil

	case tea.KeyCtrlN:
		return m.openNamePrompt(pickerNew, NewSessionName())

	case tea.KeyCtrlR:
		session, ok := m.Picker.selectedSession()
		if !ok {
			return *m, nil
		}
		return m.openNamePrompt(pickerRename, session.Name)

	case tea.KeyCtrlU:
		session, ok := m.Picker.selectedSession()
		if !ok {
			return *m, nil
		}
		copyName, err := DuplicateSession(session.Name)
		if err != nil {
			m.Picker.Status = "could not duplicate: " + err.Error()
			return *m, nil
		}
		m.Picker.Status = "duplicated as " + copyName
		m.Picker.reload()
		return *m, nil

	case tea.KeyCtrlD:
		if _, ok := m.Picker.selectedSession(); !ok {
			return *m, nil
		}
		m.Picker.Mode = pickerConfirmDelete
		return *m, nil
	}

	var cmd tea.Cmd
	m.Picker.Query, cmd = m.Picker.Query.Update(msg)
	m.Picker.Status = ""
	m.Picker.applyFilter()
	return *m, cmd
}

func (m *Model) openNamePrompt(mode pickerMode, value string) (tea.Model, tea.Cmd) {
	m.Picker.Mode = mode
	m.Picker.Status = ""
	m.Picker.NameInput.SetValue(value)
	m.Picker.NameInput.CursorEnd()
	m.Picker.NameInput.Focus()
	return *m, textinput.Blink
}

func (m *Model) handlePickerNameKeys(msg tea.KeyMsg, accept func(string) (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return *m, m.quit()

	case tea.KeyEsc:
		m.Picker.Mode = pickerBrowse
		m.Picker.NameInput.Blur()
		return *m, textinput.Blink

	case tea.KeyEnter:
		name := sanitizeName(m.Picker.NameInput.Value())
		if name == "" {
			m.Picker.Status = "name cannot be empty"
			return *m, nil
		}
		return accept(name)
	}

	var cmd tea.Cmd
	m.Picker.NameInput, cmd = m.Picker.NameInput.Update(msg)
	return *m, cmd
}

func (m *Model) acceptNewName(name string) (tea.Model, tea.Cmd) {
	if SessionExists(name) {
		m.Picker.Status = "session already exists"
		return *m, nil
	}
	return m.startNewSession(name)
}

func (m *Model) acceptRename(name string) (tea.Model, tea.Cmd) {
	session, ok := m.Picker.selectedSession()
	if !ok {
		m.Picker.Mode = pickerBrowse
		return *m, nil
	}
	if err := RenameSession(session.Name, name); err != nil {
		m.Picker.Status = "could not rename: " + err.Error()
		return *m, nil
	}
	if m.SessionName == session.Name {
		m.SessionName = name
	}

	m.Picker.Mode = pickerBrowse
	m.Picker.NameInput.Blur()
	m.Picker.reload()
	return *m, textinput.Blink
}

func (m *Model) handlePickerDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return *m, m.quit()
	}

	switch strings.ToLower(msg.String()) {
	case "y":
		return m.deleteSelectedSession()
	default:
		m.Picker.Mode = pickerBrowse
		return *m, nil
	}
}

// deleteSelectedSession removes the file. Deleting the open session moves the
// user to the next most recent one so they never end up editing a ghost.
func (m *Model) deleteSelectedSession() (tea.Model, tea.Cmd) {
	m.Picker.Mode = pickerBrowse

	session, ok := m.Picker.selectedSession()
	if !ok {
		return *m, nil
	}
	if err := DeleteSession(session.Name); err != nil {
		m.Picker.Status = "could not delete: " + err.Error()
		return *m, nil
	}

	m.Picker.Status = "deleted " + session.Name
	m.Picker.reload()

	if m.SessionName != session.Name {
		return *m, nil
	}

	m.SessionName = ""
	if next, exists := LastSessionName(); exists {
		return *m, m.switchToSession(next)
	}
	return *m, m.loadSessionContent(NewSessionName(), []string{""})
}

func (m Model) pickerWidth() int {
	width := min(56, m.Width-6)
	if width < 24 {
		width = 24
	}
	return width
}

func (m Model) renderPicker(baseView string) string {
	width := m.pickerWidth()
	m.Picker.Query.Width = width - 2
	m.Picker.NameInput.Width = width - 14

	var lines []string
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.Theme.focusedColor).
		Render("Sessions (type to filter)")
	lines = append(lines, title, "")

	switch m.Picker.Mode {
	case pickerNew:
		lines = append(lines, "New session: "+m.Picker.NameInput.View())
	case pickerRename:
		lines = append(lines, "Rename to: "+m.Picker.NameInput.View())
	case pickerConfirmDelete:
		session, _ := m.Picker.selectedSession()
		lines = append(lines, "Delete \""+session.Name+"\"? (y/N)")
	default:
		lines = append(lines, m.Picker.Query.View())
	}

	lines = append(lines, "")
	lines = append(lines, m.renderPickerList(width)...)
	lines = append(lines, "")

	footer := m.Picker.Status
	if footer == "" {
		footer = "^N new  ^R rename  ^U dup  ^D del  Esc close"
	}
	lines = append(lines, lipgloss.NewStyle().
		Foreground(m.Theme.gutterColor).
		Render(truncateVisual(footer, width)))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.borderColor).
		Background(m.Theme.popupBg).
		Foreground(m.Theme.unfocusedColor).
		Padding(1, 2).
		Width(width + 4).
		Render(strings.Join(lines, "\n"))

	return overlayCentered(baseView, box, m.Width, m.Height)
}

func (m Model) renderPickerList(width int) []string {
	if len(m.Picker.Filtered) == 0 {
		return []string{lipgloss.NewStyle().
			Foreground(m.Theme.gutterColor).
			Render("no sessions")}
	}

	start := m.Picker.Selected - pickerVisibleItems/2
	if start < 0 {
		start = 0
	}
	end := start + pickerVisibleItems
	if end > len(m.Picker.Filtered) {
		end = len(m.Picker.Filtered)
		start = end - pickerVisibleItems
		if start < 0 {
			start = 0
		}
	}

	var rendered []string
	for i := start; i < end; i++ {
		session := m.Picker.Sessions[m.Picker.Filtered[i]]
		stamp := relativeTime(session.Modified)
		nameWidth := width - lipgloss.Width(stamp) - 3
		if nameWidth < 1 {
			nameWidth = 1
		}
		row := padOrTrimVisual(truncateVisual(session.Name, nameWidth), nameWidth) + " " + stamp

		style := lipgloss.NewStyle().Foreground(m.Theme.completionFg)
		marker := "  "
		if i == m.Picker.Selected {
			style = lipgloss.NewStyle().
				Foreground(m.Theme.focusedColor).
				Background(m.Theme.completionSelBg).
				Bold(true)
			marker = "▶ "
		}
		rendered = append(rendered, style.Render(marker+row))
	}
	return rendered
}
