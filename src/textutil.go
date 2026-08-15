package main

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[mK]`)

func isWordBoundary(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '+', '-', '*', '/', '=', '(', ')', '^', '%', ',', '<', '>':
		return true
	}
	return false
}

func insertAtRune(s string, pos int, insert string) (string, int) {
	runes := []rune(s)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	ins := []rune(insert)
	out := make([]rune, 0, len(runes)+len(ins))
	out = append(out, runes[:pos]...)
	out = append(out, ins...)
	out = append(out, runes[pos:]...)
	return string(out), pos + len(ins)
}

func replaceWordAtCursor(s string, cursor int, replacement string) (string, int) {
	runes := []rune(s)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	wordStart := cursor
	for wordStart > 0 && !isWordBoundary(runes[wordStart-1]) {
		wordStart--
	}
	repl := []rune(replacement)
	out := make([]rune, 0, len(runes)-(cursor-wordStart)+len(repl))
	out = append(out, runes[:wordStart]...)
	out = append(out, repl...)
	out = append(out, runes[cursor:]...)
	return string(out), wordStart + len(repl)
}

func currentWordAt(s string, cursor int) string {
	runes := []rune(s)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	wordStart := cursor
	for wordStart > 0 && !isWordBoundary(runes[wordStart-1]) {
		wordStart--
	}
	return string(runes[wordStart:cursor])
}

func stripANSIEscapeCodes(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}

func truncateVisual(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	const ellipsis = "…"
	ellipsisW := lipgloss.Width(ellipsis)
	if maxWidth <= ellipsisW {
		return ellipsis
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw+ellipsisW > maxWidth {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + ellipsis
}

func visualSlice(s string, startCol, endCol int) string {
	if endCol <= startCol || startCol < 0 {
		return ""
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		next := col + rw
		if next > startCol && col < endCol {
			if col >= startCol && next <= endCol {
				b.WriteRune(r)
			}
		}
		col = next
		if col >= endCol {
			break
		}
	}
	return b.String()
}

func padOrTrimVisual(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w > width {
		return truncateVisual(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func runeIndexAtVisual(s string, visualCol int) int {
	if visualCol <= 0 {
		return 0
	}
	col := 0
	i := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if col+rw > visualCol {
			return i
		}
		col += rw
		i++
	}
	return i
}

func containsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func lastRune(s string) (rune, bool) {
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return r, r != utf8.RuneError
}
