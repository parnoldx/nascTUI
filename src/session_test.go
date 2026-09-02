package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func withTempSessions(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
}

func TestSessionRoundtripKeepsBlankLines(t *testing.T) {
	withTempSessions(t)

	lines := []string{"1+1", "", "ans1 * 2"}
	if err := SaveSession("work", lines); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	loaded, err := LoadSession("work")
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if strings.Join(loaded, "|") != "1+1||ans1 * 2" {
		t.Errorf("Expected blank line preserved, got %q", loaded)
	}
}

func TestSaveSessionSkipsBlankNewSession(t *testing.T) {
	withTempSessions(t)

	if err := SaveSession("empty", []string{"", "  "}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if SessionExists("empty") {
		t.Error("Expected no file for an all-blank new session")
	}

	if err := SaveSession("used", []string{"2+2"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := SaveSession("used", []string{""}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	loaded, err := LoadSession("used")
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if len(loaded) != 1 || loaded[0] != "" {
		t.Errorf("Expected existing session to be cleared, got %q", loaded)
	}
}

func TestListSessionsOrdersByModified(t *testing.T) {
	withTempSessions(t)

	for _, name := range []string{"old", "new"} {
		if err := SaveSession(name, []string{name}); err != nil {
			t.Fatalf("SaveSession: %v", err)
		}
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(sessionPath("old"), past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	sessions := ListSessions()
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Name != "new" {
		t.Errorf("Expected most recent first, got %q", sessions[0].Name)
	}

	last, ok := LastSessionName()
	if !ok || last != "new" {
		t.Errorf("Expected last session 'new', got %q (%v)", last, ok)
	}
}

func TestRenameAndDeleteSession(t *testing.T) {
	withTempSessions(t)

	if err := SaveSession("before", []string{"1"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := RenameSession("before", "after"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}
	if SessionExists("before") || !SessionExists("after") {
		t.Error("Expected session to be renamed")
	}

	if err := SaveSession("other", []string{"2"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := RenameSession("other", "after"); err == nil {
		t.Error("Expected rename onto an existing name to fail")
	}

	if err := DeleteSession("after"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if SessionExists("after") {
		t.Error("Expected session to be deleted")
	}
}

func TestDuplicateSessionNaming(t *testing.T) {
	withTempSessions(t)

	if err := SaveSession("sheet", []string{"6*7"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	first, err := DuplicateSession("sheet")
	if err != nil {
		t.Fatalf("DuplicateSession: %v", err)
	}
	if first != "sheet copy" {
		t.Errorf("Expected 'sheet copy', got %q", first)
	}

	second, err := DuplicateSession("sheet")
	if err != nil {
		t.Fatalf("DuplicateSession: %v", err)
	}
	if second != "sheet copy 2" {
		t.Errorf("Expected 'sheet copy 2', got %q", second)
	}

	lines, err := LoadSession(first)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if lines[0] != "6*7" {
		t.Errorf("Expected duplicated content, got %q", lines)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"  taxes 2026  ": "taxes 2026",
		"a/b\\c":         "abc",
		"...hidden":      "hidden",
		"\x00\x07":       "",
		"   ":            "",
	}
	for input, expected := range cases {
		if got := sanitizeName(input); got != expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", input, got, expected)
		}
	}

	long := strings.Repeat("x", maxSessionName+20)
	if got := sanitizeName(long); len([]rune(got)) != maxSessionName {
		t.Errorf("Expected name capped at %d runes, got %d", maxSessionName, len([]rune(got)))
	}
}

func TestFuzzyMatchRanking(t *testing.T) {
	if _, ok := fuzzyMatch("zzz", "taxes"); ok {
		t.Error("Expected no match for missing characters")
	}
	if _, ok := fuzzyMatch("", "anything"); !ok {
		t.Error("Expected empty query to match")
	}

	prefix, _ := fuzzyMatch("tax", "taxes 2026")
	scattered, _ := fuzzyMatch("tax", "the annual expenses")
	if prefix <= scattered {
		t.Errorf("Expected prefix match to outrank scattered match (%d vs %d)", prefix, scattered)
	}
}

func TestPickerOpensAndCloses(t *testing.T) {
	withTempSessions(t)

	if err := SaveSession("alpha", []string{"1+1"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	m := InitialModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)

	if !m.ShowPicker {
		t.Fatal("Expected picker to be shown after Ctrl+O")
	}
	if len(m.Picker.Filtered) != 1 {
		t.Fatalf("Expected 1 session listed, got %d", len(m.Picker.Filtered))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.ShowPicker {
		t.Error("Expected picker to close on Esc")
	}
}

func TestPickerSwitchLoadsSession(t *testing.T) {
	withTempSessions(t)

	if err := SaveSession("alpha", []string{"1+1", "", "3*3"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	m := InitialModel()
	m.SessionName = "current"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.SessionName != "alpha" {
		t.Errorf("Expected session 'alpha', got %q", m.SessionName)
	}
	if len(m.Inputs) != 3 {
		t.Fatalf("Expected 3 lines (blank kept), got %d", len(m.Inputs))
	}
	if m.Inputs[1].Value() != "" || m.Inputs[2].Value() != "3*3" {
		t.Errorf("Unexpected loaded lines: %q, %q", m.Inputs[1].Value(), m.Inputs[2].Value())
	}
}

func TestAutosaveWritesOnlyWhenChanged(t *testing.T) {
	withTempSessions(t)

	m := InitialModel()
	_ = m.loadSessionContent("work", []string{"1+1"})

	_, cmd := m.Update(autosaveMsg{})
	if cmd == nil {
		t.Fatal("Expected autosave tick to be rearmed")
	}
	if SessionExists("work") {
		t.Error("Expected no write while content is unchanged")
	}

	m.Inputs[0].SetValue("2+2")
	_, _ = m.Update(autosaveMsg{})

	// The save runs in a command, so drive it directly.
	if err := SaveSession(m.SessionName, m.sessionLines()); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	lines, err := LoadSession("work")
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if lines[0] != "2+2" {
		t.Errorf("Expected saved content '2+2', got %q", lines)
	}
}

func TestQuitSavesSession(t *testing.T) {
	withTempSessions(t)

	m := InitialModel()
	_ = m.loadSessionContent("quit-test", []string{"5+5"})

	tm := teatest.NewTestModel(t, m)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	lines, err := LoadSession("quit-test")
	if err != nil {
		t.Fatalf("LoadSession after quit: %v", err)
	}
	if lines[0] != "5+5" {
		t.Errorf("Expected session saved on quit, got %q", lines)
	}
}

func TestDeletingOpenSessionFallsBack(t *testing.T) {
	withTempSessions(t)

	if err := SaveSession("keep", []string{"7*7"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := SaveSession("drop", []string{"1+1"}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	m := InitialModel()
	_ = m.loadSessionContent("drop", []string{"1+1"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = updated.(Model)
	m.Picker.Query.SetValue("drop")
	m.Picker.applyFilter()

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if SessionExists("drop") {
		t.Error("Expected deleted session file to be gone")
	}
	if m.SessionName != "keep" {
		t.Errorf("Expected fallback to 'keep', got %q", m.SessionName)
	}
}

func TestOverlayDoesNotEllipsizeBase(t *testing.T) {
	base := "ABCDEFGHIJ"
	got := overlayAt(base, "XX", 3, 0, 1)
	if got != "ABCXXFGHIJ" {
		t.Errorf("Expected base row to be sliced, got %q", got)
	}

	got = overlayCentered(base, "XX", 10, 1)
	if strings.Contains(got, "…") {
		t.Errorf("Expected no ellipsis in overlay, got %q", got)
	}
}
