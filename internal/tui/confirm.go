package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Confirm is a small centered overlay presenting a yes/no prompt.
type Confirm struct {
	message string
	width   int
	onYes   func() tea.Cmd
}

// NewConfirm creates a new Confirm dialog.
func NewConfirm(message string, width int, onYes func() tea.Cmd) Confirm {
	return Confirm{message: message, width: width, onYes: onYes}
}

// Update handles key messages. Returns the updated Confirm, a command, and
// whether the dialog was closed (true = y/n/Esc pressed).
func (c Confirm) Update(msg tea.KeyPressMsg) (Confirm, tea.Cmd, bool) {
	switch {
	case isRune(msg, 'y'):
		var cmd tea.Cmd
		if c.onYes != nil {
			cmd = c.onYes()
		}
		return c, cmd, true
	case isRune(msg, 'n') || isEscape(msg):
		return c, nil, true
	}
	return c, nil, false
}

// View renders the confirmation dialog content.
func (c Confirm) View() string {
	msgStyle := lipgloss.NewStyle().Bold(true)
	hintStyle := DimmedItem

	prompt := msgStyle.Render(c.message)
	hint := hintStyle.Render("(y) yes   (n) no")

	return prompt + "\n\n" + hint
}
