package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// StatusBar renders a one-line status strip at the bottom of the TUI.
type StatusBar struct {
	width   int
	context string // e.g., "Inbox", "Today — 5 tasks"
	flash   string // temporary message
	err     string // error message
}

// keyHints is the fixed key hint string shown on the right side of the status bar.
const keyHints = "Tab focus  / search  : command  ? help  q quit"

// View renders: [context]          [flash/error]          [key hints]
func (s StatusBar) View() string {
	left := s.context
	if left == "" {
		left = "Inbox"
	}

	// Middle: show error (styled) or flash, preferring error.
	var middle string
	switch {
	case s.err != "":
		middle = ErrorStyle.Render(s.err)
	case s.flash != "":
		middle = s.flash
	}

	right := keyHints

	leftWidth := lipgloss.Width(left)
	middleWidth := lipgloss.Width(middle)
	rightWidth := lipgloss.Width(right)

	// Calculate padding to distribute remaining space.
	totalFixed := leftWidth + middleWidth + rightWidth
	remaining := s.width - totalFixed
	if remaining < 0 {
		remaining = 0
	}

	var leftPad, rightPad int
	if middleWidth > 0 {
		// Distribute space evenly on both sides of the middle section.
		leftPad = remaining / 2
		rightPad = remaining - leftPad
	} else {
		// No middle content: push right content to the far right.
		leftPad = s.width - leftWidth - rightWidth
		if leftPad < 1 {
			leftPad = 1
		}
		rightPad = 0
	}

	content := left +
		strings.Repeat(" ", leftPad) +
		middle +
		strings.Repeat(" ", rightPad) +
		right

	return StatusBarStyle.
		Width(s.width).
		Render(content)
}
