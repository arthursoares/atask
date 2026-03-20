package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// applyScroll clips lines to visible window, returns visible lines + scroll indicator.
func applyScroll(lines []string, offset, visibleHeight int) ([]string, string) {
	total := len(lines)
	if total <= visibleHeight {
		for len(lines) < visibleHeight {
			lines = append(lines, "")
		}
		return lines, ""
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total-visibleHeight {
		offset = total - visibleHeight
	}
	visible := lines[offset : offset+visibleHeight]
	indicator := fmt.Sprintf("%d–%d of %d", offset+1, offset+visibleHeight, total)
	if offset > 0 {
		indicator = "↑ " + indicator
	}
	if offset+visibleHeight < total {
		indicator += " ↓"
	}
	return visible, MutedStyle.Render(indicator)
}

// renderPane wraps content in a focused or blurred border.
func renderPane(content string, width, height int, focused bool) string {
	border := BlurredBorder
	if focused {
		border = FocusedBorder
	}
	return border.Width(width).Height(height).Render(content)
}

// truncate truncates s to maxWidth, adding "…" if needed.
func truncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxWidth-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// padRight pads s with spaces to exactly width.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
