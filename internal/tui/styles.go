package tui

import "charm.land/lipgloss/v2"

// Colors
const (
	ColorPrimary   = "#7C3AED" // Purple — focused borders, agent
	ColorSecondary = "#38BDF8" // Cyan — selected items
	ColorSuccess   = "#22C55E" // Green — completed
	ColorWarning   = "#F59E0B" // Orange — upcoming deadlines
	ColorError     = "#EF4444" // Red — overdue, errors
	ColorMuted     = "#6B7280" // Gray — dimmed, unfocused
	ColorBg        = "#1E293B" // Dark blue — selected row bg
)

// Layout
const (
	SidebarWidth = 22
	HeaderHeight = 1
	FooterHeight = 1
	BorderCols   = 2
	BorderRows   = 2
	NumPanes     = 3
)

// Pane borders
var (
	FocusedBorder = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ColorPrimary))
	BlurredBorder = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ColorMuted))
)

// Text styles
var (
	TitleStyle       = lipgloss.NewStyle().Bold(true)
	MutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	ErrorTextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
	SelectedStyle    = lipgloss.NewStyle().Background(lipgloss.Color(ColorBg)).Bold(true)
	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorSecondary)).Underline(true)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	AgentStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary))
	HumanStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary))
	OverdueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
	HeaderBarStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorSecondary))
	FooterBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	OverlayBorder    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ColorSecondary)).Padding(1, 2)
)

// Pre-rendered characters
var (
	CheckOpen   = MutedStyle.Render("☐")
	CheckDone   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)).Render("✓")
	CheckCancel = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError)).Render("✗")
)

// View icons
var ViewIcons = map[string]string{
	"inbox": "📥", "today": "⭐", "upcoming": "📅", "someday": "💤", "logbook": "📓",
}
