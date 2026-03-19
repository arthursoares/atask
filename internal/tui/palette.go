package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Command represents a single entry in the command palette.
type Command struct {
	Name     string
	Category string // "Task", "Project", "Area", "Navigation", "System"
	Action   func() tea.Cmd
}

// Palette is the command palette overlay model.
type Palette struct {
	input    textinput.Model
	commands []Command
	filtered []Command
	cursor   int
	height   int
	width    int
}

// paletteWidth is the fixed inner width of the palette box.
const paletteWidth = 40

// paletteMaxItems is the maximum number of command rows shown at once.
const paletteMaxItems = 8

// NewPalette constructs a Palette pre-loaded with the given commands.
func NewPalette(commands []Command, width, height int) Palette {
	ti := textinput.New()
	ti.Placeholder = "Type a command…"
	ti.Focus()

	p := Palette{
		input:    ti,
		commands: commands,
		width:    width,
		height:   height,
	}
	p.filtered = commands
	return p
}

// fuzzyMatch reports whether query is a case-insensitive subsequence of s.
func fuzzyMatch(s, query string) bool {
	s = strings.ToLower(s)
	query = strings.ToLower(query)
	si := 0
	for qi := 0; qi < len(query); qi++ {
		found := false
		for si < len(s) {
			if s[si] == query[qi] {
				si++
				found = true
				break
			}
			si++
		}
		if !found {
			return false
		}
	}
	return true
}

// filter rebuilds p.filtered based on the current input value.
func (p *Palette) filter() {
	q := p.input.Value()
	if q == "" {
		p.filtered = p.commands
		return
	}
	p.filtered = p.filtered[:0]
	for _, cmd := range p.commands {
		if fuzzyMatch(cmd.Name, q) || fuzzyMatch(cmd.Category, q) {
			p.filtered = append(p.filtered, cmd)
		}
	}
}

// Update handles a key press for the palette. Returns the updated palette,
// an optional tea.Cmd to run, and a bool indicating whether the palette should
// be closed (true = close).
func (p Palette) Update(msg tea.KeyPressMsg) (Palette, tea.Cmd, bool) {
	switch {
	case isEscape(msg):
		return p, nil, true

	case isEnter(msg):
		if len(p.filtered) == 0 {
			return p, nil, true
		}
		selected := p.filtered[p.cursor]
		var cmd tea.Cmd
		if selected.Action != nil {
			cmd = selected.Action()
		}
		return p, cmd, true

	case isRune(msg, 'j') || isDown(msg):
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
		return p, nil, false

	case isRune(msg, 'k') || isUp(msg):
		if p.cursor > 0 {
			p.cursor--
		}
		return p, nil, false

	default:
		var inputCmd tea.Cmd
		p.input, inputCmd = p.input.Update(msg)
		// Rebuild filtered list and reset cursor.
		p.filter()
		p.cursor = 0
		return p, inputCmd, false
	}
}

// View renders the palette as a centered overlay box.
func (p Palette) View() string {
	var b strings.Builder

	// Input row.
	b.WriteString(p.input.View())

	if len(p.filtered) > 0 {
		b.WriteString("\n")
	}

	// Command rows.
	start := 0
	end := len(p.filtered)
	if end > paletteMaxItems {
		// Scroll window so the cursor is always visible.
		if p.cursor >= paletteMaxItems {
			start = p.cursor - paletteMaxItems + 1
		}
		end = start + paletteMaxItems
		if end > len(p.filtered) {
			end = len(p.filtered)
		}
	}

	categoryStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := SelectedItem

	for i, cmd := range p.filtered[start:end] {
		absIdx := start + i
		prefix := "  "
		if absIdx == p.cursor {
			prefix = "> "
		}

		label := categoryStyle.Render(cmd.Category+" → ") + cmd.Name
		line := prefix + label

		if absIdx == p.cursor {
			// Render the whole row highlighted, re-constructing without the
			// category dim since SelectedItem sets background.
			plain := prefix + cmd.Category + " → " + cmd.Name
			line = selectedStyle.Render(plain)
		}

		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}

	// Wrap in the overlay border/padding style at a fixed width.
	inner := lipgloss.NewStyle().Width(paletteWidth).Render(b.String())
	return OverlayStyle.Render(inner)
}
