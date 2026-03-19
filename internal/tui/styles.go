package tui

import "charm.land/lipgloss/v2"

// SidebarWidth is the fixed width of the sidebar pane.
const SidebarWidth = 22

// FocusedBorder is a rounded border styled with cyan (#39) for the focused pane.
var FocusedBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("39"))

// BlurredBorder is a rounded border styled with dim grey (#240) for unfocused panes.
var BlurredBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240"))

// SectionHeader is a dim, bold style used for section headings in the sidebar.
var SectionHeader = lipgloss.NewStyle().
	Faint(true).
	Bold(true)

// SelectedItem highlights a list item with a dark blue (#236) background and bold text.
var SelectedItem = lipgloss.NewStyle().
	Background(lipgloss.Color("236")).
	Bold(true)

// DimmedItem renders text in a muted grey (#240).
var DimmedItem = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

// SelectedTask highlights a selected task with a dark teal background (#17) and cyan foreground.
var SelectedTask = lipgloss.NewStyle().
	Background(lipgloss.Color("17")).
	Foreground(lipgloss.Cyan)

// CompletedTask renders a task title in grey (#240) with strikethrough.
var CompletedTask = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240")).
	Strikethrough(true)

// OverdueDeadline renders an overdue deadline date in bright red (#196).
var OverdueDeadline = lipgloss.NewStyle().
	Foreground(lipgloss.Color("196"))

// DetailTitle renders the task detail title in bold.
var DetailTitle = lipgloss.NewStyle().
	Bold(true)

// MetadataLine renders metadata fields in a faint/dim style.
var MetadataLine = lipgloss.NewStyle().
	Faint(true)

// ActiveTab renders the active tab label as bold, cyan, and underlined.
var ActiveTab = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Cyan).
	Underline(true)

// InactiveTab renders inactive tab labels as dim.
var InactiveTab = lipgloss.NewStyle().
	Faint(true)

// AgentActor renders the agent actor label in purple (#141).
var AgentActor = lipgloss.NewStyle().
	Foreground(lipgloss.Color("141"))

// HumanActor renders the human actor label in cyan.
var HumanActor = lipgloss.NewStyle().
	Foreground(lipgloss.Cyan)

// StatusBarStyle renders the status bar with a dark background (#235).
var StatusBarStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("235"))

// ErrorStyle renders error messages in red.
var ErrorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Red)

// OverlayStyle renders modal overlays with a rounded cyan border and padding.
var OverlayStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Cyan).
	Padding(1, 2)
