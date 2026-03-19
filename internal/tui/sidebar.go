package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atask/atask/internal/client"
)

// SidebarItemKind identifies the type of a sidebar list item.
type SidebarItemKind int

const (
	ItemView          SidebarItemKind = iota
	ItemArea
	ItemProject
	ItemTag
	ItemSectionHeader
)

// SidebarItem represents a single row in the sidebar list.
type SidebarItem struct {
	Label    string
	Kind     SidebarItemKind
	ID       string
	Count    int
	Indent   int
	Expanded bool
}

// viewIcons maps view IDs to their display icons.
var viewIcons = map[string]string{
	"inbox":    "📥",
	"today":    "⭐",
	"upcoming": "📅",
	"someday":  "💤",
	"logbook":  "📓",
}

// Sidebar is the left-hand navigation pane model.
type Sidebar struct {
	items    []SidebarItem
	cursor   int
	height   int
	width    int
	offset   int // scroll offset for long lists
	focused  bool

	// internal state for rebuild
	areas      []client.Area
	projects   []client.Project
	tags       []client.Tag
	taskCounts map[string]int
	expanded   map[string]bool // area ID → expanded
}

// NewSidebar creates a Sidebar with default size and the standard views pre-populated.
func NewSidebar() Sidebar {
	s := Sidebar{
		width:    SidebarWidth,
		expanded: make(map[string]bool),
	}
	s.rebuildItems()
	return s
}

// SetSize updates the sidebar dimensions.
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetFocused marks whether the sidebar has keyboard focus.
func (s *Sidebar) SetFocused(focused bool) {
	s.focused = focused
}

// SetData rebuilds the item list from the given domain data.
func (s *Sidebar) SetData(areas []client.Area, projects []client.Project, tags []client.Tag, taskCounts map[string]int) {
	s.areas = areas
	s.projects = projects
	s.tags = tags
	s.taskCounts = taskCounts
	s.rebuildItems()
}

// rebuildItems reconstructs s.items from the stored domain data.
func (s *Sidebar) rebuildItems() {
	counts := s.taskCounts

	s.items = nil

	// Standard views
	s.items = append(s.items, SidebarItem{Label: "Inbox", Kind: ItemView, ID: "inbox", Count: counts["inbox"]})
	s.items = append(s.items, SidebarItem{Label: "Today", Kind: ItemView, ID: "today", Count: counts["today"]})
	s.items = append(s.items, SidebarItem{Label: "Upcoming", Kind: ItemView, ID: "upcoming", Count: counts["upcoming"]})
	s.items = append(s.items, SidebarItem{Label: "Someday", Kind: ItemView, ID: "someday", Count: counts["someday"]})
	s.items = append(s.items, SidebarItem{Label: "Logbook", Kind: ItemView, ID: "logbook"})

	// Areas section
	if len(s.areas) > 0 {
		s.items = append(s.items, SidebarItem{Label: "AREAS", Kind: ItemSectionHeader})
		for _, area := range s.areas {
			expanded := s.expanded[area.ID]
			s.items = append(s.items, SidebarItem{
				Label:    area.Title,
				Kind:     ItemArea,
				ID:       area.ID,
				Expanded: expanded,
			})
			if expanded {
				for _, p := range s.projects {
					if p.AreaID != nil && *p.AreaID == area.ID {
						s.items = append(s.items, SidebarItem{
							Label:  p.Title,
							Kind:   ItemProject,
							ID:     p.ID,
							Indent: 1,
						})
					}
				}
			}
		}
	}

	// Projects without an area
	var orphanProjects []client.Project
	for _, p := range s.projects {
		if p.AreaID == nil {
			orphanProjects = append(orphanProjects, p)
		}
	}
	if len(orphanProjects) > 0 {
		s.items = append(s.items, SidebarItem{Label: "PROJECTS", Kind: ItemSectionHeader})
		for _, p := range orphanProjects {
			s.items = append(s.items, SidebarItem{Label: p.Title, Kind: ItemProject, ID: p.ID})
		}
	}

	// Tags section
	if len(s.tags) > 0 {
		s.items = append(s.items, SidebarItem{Label: "TAGS", Kind: ItemSectionHeader})
		for _, tag := range s.tags {
			s.items = append(s.items, SidebarItem{Label: tag.Title, Kind: ItemTag, ID: tag.ID})
		}
	}

	// Clamp cursor so it stays on a navigable item.
	s.clampCursor()
}

// clampCursor ensures cursor is within bounds and not on a section header.
func (s *Sidebar) clampCursor() {
	if len(s.items) == 0 {
		s.cursor = 0
		return
	}
	if s.cursor >= len(s.items) {
		s.cursor = len(s.items) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
	// Move off a section header downward first, then upward.
	for s.cursor < len(s.items) && s.items[s.cursor].Kind == ItemSectionHeader {
		s.cursor++
	}
	if s.cursor >= len(s.items) {
		s.cursor = len(s.items) - 1
		for s.cursor > 0 && s.items[s.cursor].Kind == ItemSectionHeader {
			s.cursor--
		}
	}
}

// SelectedItem returns the currently highlighted sidebar item.
func (s Sidebar) SelectedItem() SidebarItem {
	if len(s.items) == 0 || s.cursor >= len(s.items) {
		return SidebarItem{}
	}
	return s.items[s.cursor]
}

// Update handles key presses when the sidebar is focused.
func (s Sidebar) Update(msg tea.KeyPressMsg) (Sidebar, tea.Cmd) {
	switch {
	case isRune(msg, 'j') || isDown(msg):
		s.moveCursor(1)

	case isRune(msg, 'k') || isUp(msg):
		s.moveCursor(-1)

	case isEnter(msg):
		return s.handleEnter()
	}

	s.adjustOffset()
	return s, nil
}

// moveCursor moves the cursor by delta, skipping section headers.
func (s *Sidebar) moveCursor(delta int) {
	next := s.cursor + delta
	for next >= 0 && next < len(s.items) {
		if s.items[next].Kind != ItemSectionHeader {
			s.cursor = next
			return
		}
		next += delta
	}
}

// handleEnter processes the Enter key on the current item.
func (s Sidebar) handleEnter() (Sidebar, tea.Cmd) {
	if len(s.items) == 0 || s.cursor >= len(s.items) {
		return s, nil
	}
	item := s.items[s.cursor]
	switch item.Kind {
	case ItemView:
		return s, func() tea.Msg { return ViewSelectedMsg{View: item.ID} }

	case ItemArea:
		s.expanded[item.ID] = !s.expanded[item.ID]
		s.rebuildItems()
		return s, func() tea.Msg { return AreaSelectedMsg{ID: item.ID} }

	case ItemProject:
		return s, func() tea.Msg { return ProjectSelectedMsg{ID: item.ID} }

	case ItemTag:
		return s, func() tea.Msg { return TagSelectedMsg{ID: item.ID} }
	}
	return s, nil
}

// adjustOffset updates the scroll offset so the cursor is always visible.
func (s *Sidebar) adjustOffset() {
	if s.height <= 0 {
		return
	}
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+s.height {
		s.offset = s.cursor - s.height + 1
	}
}

// View renders the sidebar as a string.
func (s Sidebar) View() string {
	if s.height <= 0 {
		return ""
	}

	var border lipgloss.Style
	if s.focused {
		border = FocusedBorder
	} else {
		border = BlurredBorder
	}

	// Inner width: subtract border (1 char each side).
	innerWidth := s.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	var lines []string
	visible := s.items
	if s.offset < len(visible) {
		visible = visible[s.offset:]
	} else {
		visible = nil
	}

	for i, item := range visible {
		if i >= s.height {
			break
		}
		absIdx := i + s.offset
		line := s.renderItem(item, absIdx == s.cursor, innerWidth)
		lines = append(lines, line)
	}

	// Pad with empty lines so the border box has a consistent height.
	for len(lines) < s.height {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	content := strings.Join(lines, "\n")
	return border.
		Width(innerWidth).
		Height(s.height).
		Render(content)
}

// renderItem formats a single sidebar row.
func (s Sidebar) renderItem(item SidebarItem, selected bool, width int) string {
	switch item.Kind {
	case ItemSectionHeader:
		label := SectionHeader.Render(item.Label)
		return lipgloss.NewStyle().Width(width).Render(label)
	}

	indent := strings.Repeat("  ", item.Indent)
	icon := iconFor(item)
	label := item.Label

	// Maximum label width: total width minus indent, icon+space, and badge area.
	badgeWidth := 0
	if item.Count > 0 {
		badgeWidth = len(fmt.Sprintf("%d", item.Count)) + 1 // space + digits
	}
	iconWidth := lipgloss.Width(icon)
	available := width - len(indent) - iconWidth - 1 - badgeWidth // 1 for space after icon
	if available < 1 {
		available = 1
	}

	// Truncate label if needed.
	runes := []rune(label)
	if len(runes) > available {
		runes = runes[:available-1]
		label = string(runes) + "…"
	} else {
		label = string(runes)
	}

	left := indent + icon + " " + label

	var line string
	if item.Count > 0 {
		badge := fmt.Sprintf("%d", item.Count)
		// Pad to fill width.
		padLen := width - lipgloss.Width(left) - len(badge)
		if padLen < 1 {
			padLen = 1
		}
		line = left + strings.Repeat(" ", padLen) + badge
	} else {
		line = left
	}

	if selected {
		line = SelectedItem.Width(width).Render(line)
	} else {
		line = lipgloss.NewStyle().Width(width).Render(line)
	}
	return line
}

// iconFor returns the display icon for a sidebar item.
func iconFor(item SidebarItem) string {
	switch item.Kind {
	case ItemView:
		if icon, ok := viewIcons[item.ID]; ok {
			return icon
		}
		return "•"
	case ItemArea:
		if item.Expanded {
			return "▾"
		}
		return "▸"
	case ItemProject:
		return "📁"
	case ItemTag:
		return "🏷"
	}
	return ""
}
