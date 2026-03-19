# atask TUI Rewrite — Design Spec

**Date:** 2026-03-19
**Status:** Draft
**Replaces:** `2026-03-19-atask-tui-design.md` (architecture only — same features)

## Overview

Rewrite `internal/tui/` following patterns from metering-tui. Same features as the current TUI, but with a single-model architecture, proper layout math, and polished rendering.

## What Changes

| Aspect | Current | Rewrite |
|--------|---------|---------|
| Model | Sub-models with separate Update/View | Single `Model` struct, render functions |
| Focus | `Sidebar.focused`, `List.focused`, etc. | `focusedPane` enum on Model |
| Key handling | Custom `isRune`/`isKey` helpers | `key.Binding` from bubbles with help integration |
| Layout | Inconsistent border math, double borders | Explicit height pipeline: header → content → status bar |
| Pane rendering | Each pane renders its own border | `renderPane()` applies border uniformly |
| Scroll | Per-pane offset with no indicators | Centralized `applyScroll()` with "↑↓" indicators |
| Render perf | Re-render on every message | Rate-limited render cache for SSE bursts |
| Styling | Lipgloss ANSI codes | Named color palette + pre-rendered characters |

## What Stays the Same

- All features from the original TUI spec (sidebar, list, detail, overlays, SSE, command palette, login)
- `internal/client/` — HTTP client unchanged
- `internal/tui/login.go` — login screen unchanged
- `cmd/atask/main.go` — CLI entry point unchanged

## Architecture

### Single Model

```go
type Model struct {
    client      *client.Client
    width       int
    height      int
    focusedPane Pane  // SidebarPane, ListPane, DetailPane

    // Sidebar state
    sidebarItems []SidebarItem
    sidebarCursor int
    sidebarScroll int
    areaExpanded  map[string]bool

    // List state
    tasks       []client.Task
    listCursor  int
    listScroll  int
    listTitle   string

    // Detail state
    detailTab     DetailTab  // TabNotes, TabChecklist, TabActivity
    checklist     []client.ChecklistItem
    checkCursor   int
    activities    []client.Activity
    detailScroll  int

    // Cached data
    areas     []client.Area
    projects  []client.Project
    tags      []client.Tag
    locations []client.Location
    currentView string  // "inbox", "today", "project:{id}", etc.

    // Overlays (nil = not shown)
    palette    *PaletteState
    search     *SearchState
    help       bool
    confirm    *ConfirmState
    picker     *PickerState
    schedule   *ScheduleState
    inputPrompt *InputState

    // Editing
    editing     bool
    editInput   textinput.Model

    // SSE
    sseEvents   <-chan client.DomainEvent

    // Status bar
    statusContext string
    statusFlash   string
    statusErr     string

    // Render cache
    lastRender time.Time
    renderCache string
}
```

### Layout Pipeline

```
┌─────────────────────────────────────────────────┐
│  Status context                    key hints     │ ← 1 line
├──────────┬──────────────┬───────────────────────┤
│ Sidebar  │  List Pane   │    Detail Pane         │ ← remaining height
│ (fixed   │  (~30%)      │    (~70%)              │
│  22 col) │              │                        │
│          │              │                        │
├──────────┴──────────────┴───────────────────────┤
│  [Tab] focus  [/] search  [:] cmd  [?] help     │ ← 1 line
└─────────────────────────────────────────────────┘
```

Height calculation:
```go
const headerHeight = 1
const footerHeight = 1
contentHeight = height - headerHeight - footerHeight - (2 * borderRows)
```

Width calculation:
```go
const sidebarWidth = 22
const numPanes = 3
const borderCols = 2  // per pane
totalBorderCols = numPanes * borderCols
remaining = width - sidebarWidth - totalBorderCols
listWidth = remaining * 3 / 10
detailWidth = remaining - listWidth
```

### Render Functions (not methods on sub-models)

```go
func (m Model) View() tea.View
func (m Model) renderHeader() string
func (m Model) renderSidebar(width, height int) string
func (m Model) renderList(width, height int) string
func (m Model) renderDetail(width, height int) string
func (m Model) renderFooter() string
func (m Model) renderPane(content string, width, height int, focused bool) string
func applyScroll(content string, offset, height int) (string, string)  // returns (visible, indicator)
```

Each render function receives explicit dimensions. `renderPane` wraps content with focused/blurred border. `applyScroll` clips content and adds scroll indicators.

### Key Bindings with `key.Binding`

```go
var keys = struct {
    Up, Down, Enter, Escape key.Binding
    Tab, ShiftTab           key.Binding
    Top, Bottom             key.Binding
    New, Edit, Complete     key.Binding
    Cancel, Delete          key.Binding
    Schedule, Move, Tag     key.Binding
    Location, Comment       key.Binding
    Palette, Search, Help   key.Binding
    Refresh, Quit           key.Binding
    Tab1, Tab2, Tab3        key.Binding
}{
    Up:    key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
    Down:  key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
    // ...
}
```

Use `key.Matches(msg, keys.Up)` in Update. Use `help.Model` from bubbles to render the footer key hints.

### Update Flow

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        return m, nil

    case tea.KeyPressMsg:
        // 1. Active overlay intercepts all keys
        if m.palette != nil { return m.updatePalette(msg) }
        if m.search != nil { return m.updateSearch(msg) }
        if m.help { ... }
        if m.confirm != nil { ... }
        if m.picker != nil { ... }
        if m.schedule != nil { ... }
        if m.inputPrompt != nil { ... }
        if m.editing { return m.updateEditing(msg) }

        // 2. Global keys
        if key.Matches(msg, keys.Quit) { return m, tea.Quit }
        if key.Matches(msg, keys.Tab) { m.cycleFocus(1); return m, nil }
        if key.Matches(msg, keys.Palette) { ... }
        if key.Matches(msg, keys.Search) { ... }
        if key.Matches(msg, keys.Help) { ... }
        if key.Matches(msg, keys.Refresh) { return m, m.refreshCurrentView() }

        // 3. Pane-specific keys
        switch m.focusedPane {
        case SidebarPane: return m.updateSidebar(msg)
        case ListPane: return m.updateList(msg)
        case DetailPane: return m.updateDetail(msg)
        }

    // Data messages
    case TasksLoadedMsg: ...
    case AreasLoadedMsg: ...
    case SSEEventMsg: ...
    case ErrorMsg: ...
    }
}
```

### Color Palette

```go
const (
    ColorPrimary   = "#7C3AED"  // Purple (focused borders, agent)
    ColorSecondary = "#38BDF8"  // Cyan (selected items)
    ColorSuccess   = "#22C55E"  // Green (completed)
    ColorWarning   = "#F59E0B"  // Orange (upcoming deadlines)
    ColorError     = "#EF4444"  // Red (overdue, errors)
    ColorMuted     = "#6B7280"  // Gray (dimmed text, borders)
    ColorBg        = "#1E293B"  // Dark blue (selected row bg)
)
```

Pre-rendered characters:
```go
var (
    checkboxOpen   = lipgloss.NewStyle().Foreground(ColorMuted).Render("☐")
    checkboxDone   = lipgloss.NewStyle().Foreground(ColorSuccess).Render("✓")
    checkboxCancel = lipgloss.NewStyle().Foreground(ColorError).Render("✗")
    iconInbox      = "📥"
    iconToday      = "⭐"
    // ...
)
```

### Rate-Limited Rendering

```go
func (m Model) View() tea.View {
    if time.Since(m.lastRender) < 50*time.Millisecond && m.renderCache != "" {
        return tea.NewView(m.renderCache)
    }
    // ... render ...
    m.renderCache = result
    m.lastRender = time.Now()
    return tea.NewView(result)
}
```

### Scroll with Indicators

```go
func applyScroll(lines []string, offset, visibleHeight int) ([]string, string) {
    total := len(lines)
    if total <= visibleHeight {
        return lines, ""
    }
    if offset > total-visibleHeight {
        offset = total - visibleHeight
    }
    visible := lines[offset : offset+visibleHeight]

    indicator := fmt.Sprintf("%d-%d of %d", offset+1, offset+visibleHeight, total)
    if offset > 0 { indicator = "↑ " + indicator }
    if offset+visibleHeight < total { indicator += " ↓" }
    return visible, indicator
}
```

### File Structure

```
internal/tui/
├── model.go          → Model struct, Init, NewModel
├── update.go         → Update, updateSidebar, updateList, updateDetail
├── view.go           → View, renderHeader, renderSidebar, renderList, renderDetail, renderFooter
├── overlay.go        → updatePalette, updateSearch, updatePicker, etc. + render functions
├── commands.go       → all tea.Cmd functions (API calls, SSE, refresh)
├── keys.go           → key.Binding definitions
├── styles.go         → color palette, pre-rendered chars, lipgloss styles
├── messages.go       → all tea.Msg types
├── scroll.go         → applyScroll, renderPane
├── login.go          → login screen (unchanged)
```

9 files total (down from 13). Each file has one clear responsibility.
