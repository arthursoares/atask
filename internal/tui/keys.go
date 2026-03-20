package tui

import "charm.land/bubbles/v2/key"

var Keys = struct {
	Up, Down      key.Binding
	Enter, Escape key.Binding
	Tab, ShiftTab key.Binding
	Top, Bottom   key.Binding

	New, Edit, Complete key.Binding
	Cancel, Delete      key.Binding
	Schedule, Move, Tag key.Binding
	Location, Comment   key.Binding

	Palette, Search, Help key.Binding
	Refresh, Quit         key.Binding

	Tab1, Tab2, Tab3 key.Binding
}{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "select")),
	Escape:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
	ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev pane")),
	Top:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
	Bottom:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),

	New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Complete: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "complete")),
	Cancel:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "cancel")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Schedule: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "schedule")),
	Move:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move")),
	Tag:      key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tag")),
	Location: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "location")),
	Comment:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "comment")),

	Palette: key.NewBinding(key.WithKeys(":", "ctrl+p"), key.WithHelp(":/^P", "commands")),
	Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),

	Tab1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "notes")),
	Tab2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "checklist")),
	Tab3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "activity")),
}
