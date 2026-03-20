package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atask/atask/internal/client"
)

// View implements tea.Model. It applies a 50 ms render-rate limit, computes
// the three-pane layout, composes header/content/footer and overlays.
func (m Model) View() tea.View {
	// Rate-limit: skip if last render was <50 ms ago and cache is warm.
	now := time.Now().UnixMilli()
	if m.renderCache != "" && now-m.lastRender < 50 {
		return tea.NewView(m.renderCache)
	}

	// Guard against zero dimensions (before first WindowSizeMsg).
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}

	// ---- Dimensions ---------------------------------------------------------
	contentHeight := m.height - HeaderHeight - FooterHeight - BorderRows
	if contentHeight < 1 {
		contentHeight = 1
	}
	remaining := m.width - SidebarWidth - (NumPanes * BorderCols)
	if remaining < 2 {
		remaining = 2
	}
	listWidth := remaining * 3 / 10
	if listWidth < 4 {
		listWidth = 4
	}
	detailWidth := remaining - listWidth

	// ---- Panes --------------------------------------------------------------
	sidebarContent := m.renderSidebar(SidebarWidth, contentHeight)
	listContent := m.renderList(listWidth, contentHeight)
	detailContent := m.renderDetail(detailWidth, contentHeight)

	sidebar := renderPane(sidebarContent, SidebarWidth, contentHeight, m.focusedPane == SidebarPane)
	list := renderPane(listContent, listWidth, contentHeight, m.focusedPane == ListPane)
	detail := renderPane(detailContent, detailWidth, contentHeight, m.focusedPane == DetailPane)

	// ---- Compose ------------------------------------------------------------
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, list, detail)

	header := m.renderHeader()
	footer := m.renderFooter()

	result := lipgloss.JoinVertical(lipgloss.Left, header, content, footer)

	// ---- Overlays -----------------------------------------------------------
	switch {
	case m.palette != nil:
		result = m.renderOverlay(result, m.renderPaletteOverlay())
	case m.search != nil:
		result = m.renderOverlay(result, m.renderSearchOverlay())
	case m.showHelp:
		result = m.renderOverlay(result, m.renderHelpOverlay())
	case m.confirm != nil:
		result = m.renderOverlay(result, m.renderConfirmOverlay())
	case m.picker != nil:
		result = m.renderOverlay(result, m.renderPickerOverlay())
	case m.schedule != nil:
		result = m.renderOverlay(result, m.renderScheduleOverlay())
	case m.inputPrompt != nil:
		result = m.renderOverlay(result, m.renderInputOverlay())
	}

	// Cache and return.
	m.renderCache = result
	m.lastRender = now

	return tea.NewView(result)
}

// renderHeader renders the single-line header bar.
// Left: statusContext (bold cyan). Middle: error or flash message. Padded to m.width.
func (m Model) renderHeader() string {
	left := HeaderBarStyle.Render(m.statusContext)

	var mid string
	switch {
	case m.statusErr != "":
		mid = ErrorTextStyle.Render(m.statusErr)
	case m.statusFlash != "":
		mid = MutedStyle.Render(m.statusFlash)
	}

	leftWidth := lipgloss.Width(left)
	midWidth := lipgloss.Width(mid)
	rightPad := m.width - leftWidth - midWidth
	if rightPad < 1 {
		rightPad = 1
	}
	half := rightPad / 2
	return left + strings.Repeat(" ", half) + mid + strings.Repeat(" ", rightPad-half)
}

// renderFooter renders the single-line key-hint bar.
func (m Model) renderFooter() string {
	hints := "[Tab] focus  [/] search  [:] cmd  [?] help  [q] quit"
	return padRight(FooterBarStyle.Render(hints), m.width)
}

// renderSidebar renders the sidebar item list into width×height.
func (m Model) renderSidebar(width, height int) string {
	lines := make([]string, 0, len(m.sidebarItems))

	for i, item := range m.sidebarItems {
		var line string
		switch item.Kind {
		case "header":
			label := MutedStyle.Render(strings.ToUpper(item.Label))
			line = padRight(truncate(label, width), width)

		case "view":
			icon := ViewIcons[item.ID]
			if icon == "" {
				icon = "  "
			}
			label := icon + " " + item.Label
			badge := ""
			if item.Count > 0 {
				badge = MutedStyle.Render(fmt.Sprintf("%d", item.Count))
			}
			badgeWidth := lipgloss.Width(badge)
			titleAvail := width - badgeWidth
			if titleAvail < 1 {
				titleAvail = 1
			}
			truncLabel := truncate(label, titleAvail)
			line = padRight(truncLabel, titleAvail) + badge

		case "area":
			chevron := "▸"
			if item.Expanded {
				chevron = "▾"
			}
			line = padRight(truncate(chevron+" "+item.Label, width), width)

		case "project":
			line = padRight(truncate("  "+IconProject+" "+item.Label, width), width)

		case "tag":
			line = padRight(truncate(IconTag+" "+item.Label, width), width)

		default:
			line = padRight(truncate(item.Label, width), width)
		}

		if i == m.sidebarCursor {
			line = SelectedStyle.Width(width).Render(line)
		}

		lines = append(lines, line)
	}

	visible, _ := applyScroll(lines, m.sidebarScroll, height)
	return strings.Join(visible, "\n")
}

// renderList renders the task list pane into width×height.
func (m Model) renderList(width, height int) string {
	// Title bar (counts against height).
	count := len(m.tasks)
	noun := "tasks"
	if count == 1 {
		noun = "task"
	}
	titleText := fmt.Sprintf("%s — %d %s", m.listTitle, count, noun)
	titleBar := padRight(TitleStyle.Render(truncate(titleText, width)), width)

	taskHeight := height - 1 // subtract title bar
	if taskHeight < 0 {
		taskHeight = 0
	}

	lines := make([]string, 0, len(m.tasks))
	for i, task := range m.tasks {
		var line string

		if m.editing && i == m.listCursor {
			// Inline edit mode: show the text input instead of the title.
			line = padRight(m.editInput.View(), width)
		} else {
			// Status icon.
			var icon string
			switch task.Status {
			case 1:
				icon = CheckDone
			case 2:
				icon = CheckCancel
			default:
				icon = CheckOpen
			}

			// Right-side decoration: deadline or project icon.
			var right string
			if task.Deadline != nil && *task.Deadline != "" {
				dl := formatDeadline(*task.Deadline)
				if isOverdue(*task.Deadline) {
					right = OverdueStyle.Render(dl)
				} else {
					right = MutedStyle.Render(dl)
				}
			} else if task.ProjectID != nil && *task.ProjectID != "" {
				right = IconFolder
			}

			iconWidth := lipgloss.Width(icon)
			rightWidth := lipgloss.Width(right)
			// 1 space between icon and title, 1 space before right.
			titleAvail := width - iconWidth - 1 - rightWidth - 1
			if titleAvail < 1 {
				titleAvail = 1
			}
			title := padRight(truncate(task.Title, titleAvail), titleAvail)

			line = icon + " " + title + " " + right

			if task.Status == 1 || task.Status == 2 {
				line = MutedStyle.Render(padRight(line, width))
			} else {
				line = padRight(line, width)
			}
		}

		if i == m.listCursor && !m.editing {
			line = SelectedStyle.Width(width).Render(line)
		}

		lines = append(lines, line)
	}

	visible, scrollIndicator := applyScroll(lines, m.listScroll, taskHeight)
	visibleLines := strings.Join(visible, "\n")
	if scrollIndicator != "" {
		// Append scroll indicator on its own line, replacing the last line.
		parts := strings.Split(visibleLines, "\n")
		if len(parts) > 0 {
			parts[len(parts)-1] = padRight(scrollIndicator, width)
			visibleLines = strings.Join(parts, "\n")
		}
	}

	return titleBar + "\n" + visibleLines
}

// renderDetail renders the detail pane into width×height.
func (m Model) renderDetail(width, height int) string {
	if m.selectedTask == nil {
		placeholder := MutedStyle.Render("Select a task to view details")
		padTop := height / 2
		padLeft := (width - lipgloss.Width(placeholder)) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		lines := make([]string, height)
		for i := range lines {
			lines[i] = strings.Repeat(" ", width)
		}
		if padTop < height {
			placeholderW := lipgloss.Width(placeholder)
			right := width - padLeft - placeholderW
			if right < 0 {
				right = 0
			}
			lines[padTop] = strings.Repeat(" ", padLeft) + placeholder + strings.Repeat(" ", right)
		}
		return strings.Join(lines, "\n")
	}

	task := m.selectedTask

	// 1. Title line.
	titleLine := padRight(TitleStyle.Render(truncate(task.Title, width)), width)

	// 2. Metadata line.
	var meta []string
	if task.ProjectID != nil && *task.ProjectID != "" {
		meta = append(meta, IconFolder+" "+m.projectTitle(*task.ProjectID))
	}
	for _, tagID := range task.Tags {
		meta = append(meta, IconTag+" "+m.tagTitle(tagID))
	}
	if task.Deadline != nil && *task.Deadline != "" {
		dl := formatDeadline(*task.Deadline)
		if isOverdue(*task.Deadline) {
			meta = append(meta, OverdueStyle.Render("due:"+dl))
		} else {
			meta = append(meta, MutedStyle.Render("due:"+dl))
		}
	}
	if task.LocationID != nil && *task.LocationID != "" {
		meta = append(meta, MutedStyle.Render("📍 "+m.locationName(*task.LocationID)))
	}
	metaLine := padRight(MutedStyle.Render(truncate(strings.Join(meta, "  "), width)), width)

	// 3. Tab bar.
	tabBar := m.renderDetailTabBar(width)

	// Lines consumed by fixed header rows (title + meta + tab bar).
	const headerLines = 3

	// Latest activity footer: always visible, 1 line.
	var latestActivityLine string
	if len(m.activities) > 0 {
		latestActivityLine = padRight(renderActivityEntry(m.activities[len(m.activities)-1], width), width)
	}
	footerLines := 0
	if latestActivityLine != "" {
		footerLines = 1
	}

	tabContentHeight := height - headerLines - footerLines
	if tabContentHeight < 0 {
		tabContentHeight = 0
	}

	// 4. Tab content.
	tabContent := m.renderDetailTabContent(width, tabContentHeight)

	var sb strings.Builder
	sb.WriteString(titleLine)
	sb.WriteByte('\n')
	sb.WriteString(metaLine)
	sb.WriteByte('\n')
	sb.WriteString(tabBar)
	sb.WriteByte('\n')
	sb.WriteString(tabContent)
	if latestActivityLine != "" {
		sb.WriteByte('\n')
		sb.WriteString(latestActivityLine)
	}
	return sb.String()
}

// renderDetailTabBar renders the [Notes] [Checklist (x/y)] [Activity (n)] tab bar.
func (m Model) renderDetailTabBar(width int) string {
	// Checklist badge.
	done := 0
	for _, item := range m.checklist {
		if item.Status == 1 {
			done++
		}
	}
	checkBadge := ""
	if len(m.checklist) > 0 {
		checkBadge = fmt.Sprintf(" (%d/%d)", done, len(m.checklist))
	}

	actBadge := ""
	if len(m.activities) > 0 {
		actBadge = fmt.Sprintf(" (%d)", len(m.activities))
	}

	tabs := []struct {
		label string
		tab   DetailTab
	}{
		{"Notes", TabNotes},
		{"Checklist" + checkBadge, TabChecklist},
		{"Activity" + actBadge, TabActivity},
	}

	var parts []string
	for _, t := range tabs {
		label := "[" + t.label + "]"
		if m.detailTab == t.tab {
			parts = append(parts, ActiveTabStyle.Render(label))
		} else {
			parts = append(parts, InactiveTabStyle.Render(label))
		}
	}
	return padRight(strings.Join(parts, " "), width)
}

// renderDetailTabContent renders the scrollable tab body.
func (m Model) renderDetailTabContent(width, height int) string {
	var lines []string

	switch m.detailTab {
	case TabNotes:
		if m.selectedTask == nil || m.selectedTask.Notes == "" {
			lines = []string{MutedStyle.Render("No notes.")}
		} else {
			for _, l := range strings.Split(m.selectedTask.Notes, "\n") {
				lines = append(lines, padRight(truncate(l, width), width))
			}
		}

	case TabChecklist:
		if len(m.checklist) == 0 {
			lines = []string{MutedStyle.Render("No checklist items.")}
		}
		for i, item := range m.checklist {
			var icon string
			if item.Status == 1 {
				icon = CheckDone
			} else {
				icon = CheckOpen
			}
			iconW := lipgloss.Width(icon)
			label := truncate(item.Title, width-iconW-1)
			line := padRight(icon+" "+label, width)
			if i == m.checkCursor {
				line = SelectedStyle.Width(width).Render(line)
			}
			lines = append(lines, line)
		}

	case TabActivity:
		if len(m.activities) == 0 {
			lines = []string{MutedStyle.Render("No activity yet.")}
		}
		for _, a := range m.activities {
			lines = append(lines, padRight(renderActivityEntry(a, width), width))
		}
	}

	visible, _ := applyScroll(lines, m.detailScroll, height)
	return strings.Join(visible, "\n")
}

// renderActivityEntry formats one Activity into a single line ≤ width.
// renderActivityEntry formats one activity entry for display.
func renderActivityEntry(a client.Activity, width int) string {
	var actorIcon string
	if a.ActorType == "agent" || a.ActorType == "bot" {
		actorIcon = AgentStyle.Render("»")
	} else {
		actorIcon = HumanStyle.Render("›")
	}

	ts := ""
	if t, err := time.Parse(time.RFC3339, a.CreatedAt); err == nil {
		ts = MutedStyle.Render(t.Format("Jan 2 15:04"))
	}

	iconW := lipgloss.Width(actorIcon)
	tsW := lipgloss.Width(ts)
	// type + space + content
	typeLabel := a.Type
	textAvail := width - iconW - 1 - tsW - 1
	if textAvail < 1 {
		textAvail = 1
	}
	body := truncate(typeLabel+": "+a.Content, textAvail)
	return actorIcon + " " + padRight(body, textAvail) + " " + ts
}

// renderOverlay centers an already-rendered overlay string on top of the base
// string by replacing lines. The overlay renderers in overlay.go already apply
// OverlayBorder, so no additional border is added here.
func (m Model) renderOverlay(base, overlay string) string {
	overlayLines := strings.Split(overlay, "\n")
	baseLines := strings.Split(base, "\n")

	overlayH := len(overlayLines)
	overlayW := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > overlayW {
			overlayW = w
		}
	}

	// Vertical: start at 1/3 from top.
	vStart := (len(baseLines) - overlayH) / 3
	if vStart < 0 {
		vStart = 0
	}

	// Horizontal: center.
	hPad := (m.width - overlayW) / 2
	if hPad < 0 {
		hPad = 0
	}
	prefix := strings.Repeat(" ", hPad)

	for i, ol := range overlayLines {
		row := vStart + i
		if row >= len(baseLines) {
			break
		}
		baseLines[row] = prefix + ol
	}
	return strings.Join(baseLines, "\n")
}

// ---- Lookup helpers ---------------------------------------------------------

func (m Model) projectTitle(id string) string {
	for _, p := range m.projects {
		if p.ID == id {
			return p.Title
		}
	}
	return id
}

func (m Model) tagTitle(id string) string {
	for _, t := range m.tags {
		if t.ID == id {
			return t.Title
		}
	}
	return id
}

func (m Model) locationName(id string) string {
	for _, l := range m.locations {
		if l.ID == id {
			return l.Name
		}
	}
	return id
}

// ---- Date helpers -----------------------------------------------------------

// formatDeadline returns a short human-readable date from an ISO string.
func formatDeadline(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		t, err = time.Parse("2006-01-02", iso)
		if err != nil {
			return iso
		}
	}
	return t.Format("Jan 2")
}

// isOverdue returns true if the ISO date string is in the past (before today).
func isOverdue(iso string) bool {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		t, err = time.Parse("2006-01-02", iso)
		if err != nil {
			return false
		}
	}
	return t.Before(time.Now().Truncate(24 * time.Hour))
}
