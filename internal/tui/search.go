package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
)

// SearchQueryMsg is emitted on every keystroke so the parent can filter its list.
type SearchQueryMsg struct {
	Query string
}

// Search is the inline search/filter input rendered at the top of the list pane.
type Search struct {
	input  textinput.Model
	active bool
	width  int
}

// NewSearch creates a new Search model sized to the given width.
func NewSearch(width int) Search {
	ti := textinput.New()
	ti.Placeholder = "Search…"
	ti.Focus()
	return Search{
		input:  ti,
		active: true,
		width:  width,
	}
}

// Update handles key messages. Returns the updated Search, a command, and
// whether the search was closed (true = closed / Esc pressed).
func (s Search) Update(msg tea.KeyPressMsg) (Search, tea.Cmd, bool) {
	if isEscape(msg) {
		s.active = false
		s.input.SetValue("")
		// Emit empty query so the parent clears its filter.
		return s, func() tea.Msg { return SearchQueryMsg{Query: ""} }, true
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	queryCmd := func() tea.Msg { return SearchQueryMsg{Query: s.input.Value()} }
	return s, tea.Batch(cmd, queryCmd), false
}

// View renders the search bar.
func (s Search) View() string {
	if !s.active {
		return ""
	}
	return "/ " + s.input.View()
}

// Query returns the current search string.
func (s Search) Query() string {
	return s.input.Value()
}
