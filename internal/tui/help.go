package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const helpContent = `Navigation:
  Tab/Shift+Tab   Cycle pane focus
  j/k             Navigate within pane
  Enter           Select / expand
  Esc             Back / cancel
  g/G             Top / bottom

Actions:
  n               New (context-aware)
  e               Edit / rename
  x               Complete task / toggle checklist
  X               Cancel task
  d               Delete (with confirmation)
  s               Schedule picker
  m               Move to project
  t               Assign tag
  l               Set location
  a               Add comment / archive area

Detail Pane:
  1/2/3           Switch tabs
  Tab             Cycle tabs

Global:
  :  or  Ctrl+P   Command palette
  /               Search / filter
  ?               Help (this screen)
  r               Refresh
  q               Quit`

// Help is a scrollable full-screen overlay showing key binding reference.
type Help struct {
	width  int
	height int
	offset int // scroll offset (lines)
}

// NewHelp creates a new Help overlay sized to the given dimensions.
func NewHelp(width, height int) Help {
	return Help{width: width, height: height}
}

// Update handles key messages. Returns the updated Help, a command, and
// whether the overlay was closed (true = Esc pressed).
func (h Help) Update(msg tea.KeyPressMsg) (Help, tea.Cmd, bool) {
	lines := strings.Split(helpContent, "\n")
	maxOffset := len(lines) - h.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}

	switch {
	case isEscape(msg):
		return h, nil, true
	case isRune(msg, 'j') || isDown(msg):
		if h.offset < maxOffset {
			h.offset++
		}
	case isRune(msg, 'k') || isUp(msg):
		if h.offset > 0 {
			h.offset--
		}
	case isRune(msg, 'g'):
		h.offset = 0
	case isRune(msg, 'G'):
		h.offset = maxOffset
	}
	return h, nil, false
}

// visibleLines returns the number of content lines that fit in the overlay.
func (h Help) visibleLines() int {
	v := h.height - 4 // account for title + padding
	if v < 1 {
		v = 10
	}
	return v
}

// View renders the help overlay content.
func (h Help) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan)
	title := titleStyle.Render("Key Bindings")

	lines := strings.Split(helpContent, "\n")
	visible := h.visibleLines()

	end := h.offset + visible
	if end > len(lines) {
		end = len(lines)
	}
	shown := lines[h.offset:end]

	// Scroll hint
	hint := ""
	if h.offset > 0 || end < len(lines) {
		hint = DimmedItem.Render("  (j/k to scroll, Esc to close)")
	} else {
		hint = DimmedItem.Render("  (Esc to close)")
	}

	parts := []string{title, strings.Join(shown, "\n"), hint}
	return strings.Join(parts, "\n")
}
