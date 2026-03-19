package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// PickerItem represents a selectable item in the fuzzy picker.
type PickerItem struct {
	ID    string
	Label string
}

// Picker is a fuzzy-search overlay for selecting from a list of items.
type Picker struct {
	title    string
	input    textinput.Model
	items    []PickerItem
	filtered []PickerItem
	cursor   int
	width    int
	height   int
	onSelect func(id string) tea.Cmd
}

// NewPicker creates a new Picker overlay.
func NewPicker(title string, items []PickerItem, width, height int, onSelect func(string) tea.Cmd) Picker {
	ti := textinput.New()
	ti.Placeholder = "Type to filter…"
	ti.Focus()

	p := Picker{
		title:    title,
		input:    ti,
		items:    items,
		width:    width,
		height:   height,
		onSelect: onSelect,
	}
	p.filtered = items
	return p
}

// filter rebuilds the filtered list using case-insensitive substring matching.
func (p *Picker) filter(query string) {
	if query == "" {
		p.filtered = p.items
		p.cursor = 0
		return
	}
	lower := strings.ToLower(query)
	p.filtered = p.filtered[:0]
	for _, item := range p.items {
		if strings.Contains(strings.ToLower(item.Label), lower) {
			p.filtered = append(p.filtered, item)
		}
	}
	p.cursor = 0
}

// Update handles key messages. Returns the updated Picker, a command, and
// whether the picker was closed (true = closed).
func (p Picker) Update(msg tea.KeyPressMsg) (Picker, tea.Cmd, bool) {
	switch {
	case isEscape(msg):
		return p, nil, true

	case isEnter(msg):
		if len(p.filtered) == 0 {
			return p, nil, true
		}
		selected := p.filtered[p.cursor]
		var cmd tea.Cmd
		if p.onSelect != nil {
			cmd = p.onSelect(selected.ID)
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
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		p.filter(p.input.Value())
		return p, cmd, false
	}
}

// View renders the picker overlay content.
func (p Picker) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan)
	title := titleStyle.Render(p.title)

	inputView := p.input.View()

	// Determine how many list rows to show.
	maxRows := p.height - 4 // title + input + borders
	if maxRows < 1 {
		maxRows = 5
	}

	var rows []string
	for i, item := range p.filtered {
		if i >= maxRows {
			break
		}
		line := item.Label
		if i == p.cursor {
			line = SelectedItem.Width(p.width - 4).Render("> " + line)
		} else {
			line = lipgloss.NewStyle().Width(p.width - 4).Render("  " + line)
		}
		rows = append(rows, line)
	}

	if len(rows) == 0 {
		rows = append(rows, DimmedItem.Render("  no matches"))
	}

	parts := []string{title, inputView, strings.Join(rows, "\n")}
	return strings.Join(parts, "\n")
}
