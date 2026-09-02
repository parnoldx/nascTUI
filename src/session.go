package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	sessionExt     = ".nasc"
	maxSessionName = 64
)

type Session struct {
	Name     string
	Modified time.Time
}

func sessionsDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "nasc-tui", "sessions")
}

func sessionPath(name string) string {
	return filepath.Join(sessionsDir(), name+sessionExt)
}

// sanitizeName turns user input into something safe to use as a file name.
// Returns "" when nothing usable is left.
func sanitizeName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < ' ' || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)

	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimLeft(cleaned, ".")
	cleaned = strings.TrimSpace(cleaned)

	if len([]rune(cleaned)) > maxSessionName {
		cleaned = string([]rune(cleaned)[:maxSessionName])
		cleaned = strings.TrimSpace(cleaned)
	}
	return cleaned
}

func ListSessions() []Session {
	entries, err := os.ReadDir(sessionsDir())
	if err != nil {
		return nil
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), sessionExt) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		sessions = append(sessions, Session{
			Name:     strings.TrimSuffix(entry.Name(), sessionExt),
			Modified: info.ModTime(),
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	return sessions
}

func SessionExists(name string) bool {
	_, err := os.Stat(sessionPath(name))
	return err == nil
}

// LoadSession returns the stored input lines. Blank lines in the middle are kept
// because ans1/ans2 references address lines by position.
func LoadSession(name string) ([]string, error) {
	data, err := os.ReadFile(sessionPath(name))
	if err != nil {
		return nil, err
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines, nil
}

// SaveSession writes the lines atomically. An all-blank session is not created,
// so sessions the user never typed into leave no file behind.
func SaveSession(name string, lines []string) error {
	if name == "" {
		return nil
	}
	if isBlank(lines) && !SessionExists(name) {
		return nil
	}

	dir := sessionsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	temp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()

	content := strings.Join(lines, "\n") + "\n"
	if _, err := temp.WriteString(content); err != nil {
		temp.Close()
		os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, sessionPath(name)); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}

func RenameSession(oldName, newName string) error {
	if oldName == newName {
		return nil
	}
	if newName == "" {
		return fmt.Errorf("empty session name")
	}
	if SessionExists(newName) {
		return fmt.Errorf("session %q already exists", newName)
	}
	return os.Rename(sessionPath(oldName), sessionPath(newName))
}

func DuplicateSession(name string) (string, error) {
	data, err := os.ReadFile(sessionPath(name))
	if err != nil {
		return "", err
	}

	copyName := uniqueName(name + " copy")
	if err := os.WriteFile(sessionPath(copyName), data, 0o644); err != nil {
		return "", err
	}
	return copyName, nil
}

func DeleteSession(name string) error {
	return os.Remove(sessionPath(name))
}

func LastSessionName() (string, bool) {
	sessions := ListSessions()
	if len(sessions) == 0 {
		return "", false
	}
	return sessions[0].Name, true
}

func NewSessionName() string {
	return uniqueName(time.Now().Format("2006-01-02"))
}

func uniqueName(base string) string {
	if !SessionExists(base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s %d", base, i)
		if !SessionExists(candidate) {
			return candidate
		}
	}
}

func isBlank(lines []string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return false
		}
	}
	return true
}
