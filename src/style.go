package main

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	focusedColor    lipgloss.Color
	unfocusedColor  lipgloss.Color
	resultColor     lipgloss.Color
	borderColor     lipgloss.Color
	inputBg         lipgloss.Color
	resultBg        lipgloss.Color
	gutterColor     lipgloss.Color
	ansColor        lipgloss.Color
	completionFg    lipgloss.Color
	completionSelBg lipgloss.Color
	popupBg         lipgloss.Color
}

func newTheme() Theme {
	return Theme{
		focusedColor:    lipgloss.Color("4"),
		unfocusedColor:  lipgloss.Color("7"),
		resultColor:     lipgloss.Color("3"),
		borderColor:     lipgloss.Color("5"),
		inputBg:         lipgloss.Color("0"),
		resultBg:        lipgloss.Color("0"),
		gutterColor:     lipgloss.Color("8"),
		ansColor:        lipgloss.Color("2"),
		completionFg:    lipgloss.Color("7"),
		completionSelBg: lipgloss.Color("8"),
		popupBg:         lipgloss.Color("0"),
	}
}
