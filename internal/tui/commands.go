package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atask/atask/internal/client"
)

// --- Data loading ---

func (m Model) cmdLoadInbox() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListInbox(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadToday() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListToday(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadUpcoming() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListUpcoming(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadSomeday() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListSomeday(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadLogbook() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListLogbook(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadProjectTasks(projectID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListTasksByProject(context.Background(), projectID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadAreaTasks(areaID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListTasksByArea(context.Background(), areaID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadAreas() tea.Cmd {
	return func() tea.Msg {
		areas, err := m.client.ListAreas(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return AreasLoadedMsg{Areas: areas}
	}
}

func (m Model) cmdLoadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.client.ListProjects(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func (m Model) cmdLoadTags() tea.Cmd {
	return func() tea.Msg {
		tags, err := m.client.ListTags(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TagsLoadedMsg{Tags: tags}
	}
}

func (m Model) cmdLoadLocations() tea.Cmd {
	return func() tea.Msg {
		locations, err := m.client.ListLocations(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return LocationsLoadedMsg{Locations: locations}
	}
}

func (m Model) cmdLoadChecklist(taskID string) tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.ListChecklistItems(context.Background(), taskID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ChecklistLoadedMsg{Items: items}
	}
}

func (m Model) cmdLoadActivities(taskID string) tea.Cmd {
	return func() tea.Msg {
		activities, err := m.client.ListActivities(context.Background(), taskID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ActivitiesLoadedMsg{Activities: activities}
	}
}

// --- Task mutations ---

func (m Model) cmdCompleteTask(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.CompleteTask(context.Background(), id); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (m Model) cmdCancelTask(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.CancelTask(context.Background(), id); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (m Model) cmdDeleteTask(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteTask(context.Background(), id); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (m Model) cmdCreateTask(title string) tea.Cmd {
	return func() tea.Msg {
		task, err := m.client.CreateTask(context.Background(), title)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TaskCreatedMsg{Task: *task}
	}
}

func (m Model) cmdUpdateTitle(id, title string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.UpdateTaskTitle(context.Background(), id, title); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (m Model) cmdUpdateSchedule(id, schedule string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.UpdateTaskSchedule(context.Background(), id, schedule); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (m Model) cmdMoveToProject(id string, projectID *string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.MoveTaskToProject(context.Background(), id, projectID); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

// --- Detail mutations ---

func (m Model) cmdAddComment(taskID, content string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.client.AddActivity(context.Background(), taskID, "human", "comment", content); err != nil {
			return ErrorMsg{Err: err}
		}
		return DetailRefreshMsg{TaskID: taskID}
	}
}

func (m Model) cmdCompleteCheckItem(taskID, itemID string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.CompleteChecklistItem(context.Background(), taskID, itemID); err != nil {
			return ErrorMsg{Err: err}
		}
		return DetailRefreshMsg{TaskID: taskID}
	}
}

func (m Model) cmdUncompleteCheckItem(taskID, itemID string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.UncompleteChecklistItem(context.Background(), taskID, itemID); err != nil {
			return ErrorMsg{Err: err}
		}
		return DetailRefreshMsg{TaskID: taskID}
	}
}

func (m Model) cmdAddCheckItem(taskID, title string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.client.AddChecklistItem(context.Background(), taskID, title); err != nil {
			return ErrorMsg{Err: err}
		}
		return DetailRefreshMsg{TaskID: taskID}
	}
}

func (m Model) cmdDeleteCheckItem(taskID, itemID string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteChecklistItem(context.Background(), taskID, itemID); err != nil {
			return ErrorMsg{Err: err}
		}
		return DetailRefreshMsg{TaskID: taskID}
	}
}

// --- Routing ---

// refreshCurrentView dispatches to the correct load command based on m.currentView.
// Project views are encoded as "project:{id}" and area views as "area:{id}".
func (m Model) refreshCurrentView() tea.Cmd {
	switch {
	case m.currentView == viewInbox:
		return m.cmdLoadInbox()
	case m.currentView == viewToday:
		return m.cmdLoadToday()
	case m.currentView == viewUpcoming:
		return m.cmdLoadUpcoming()
	case m.currentView == viewSomeday:
		return m.cmdLoadSomeday()
	case m.currentView == viewLogbook:
		return m.cmdLoadLogbook()
	case strings.HasPrefix(m.currentView, viewProject+":"):
		id := strings.TrimPrefix(m.currentView, viewProject+":")
		return m.cmdLoadProjectTasks(id)
	case strings.HasPrefix(m.currentView, viewArea+":"):
		id := strings.TrimPrefix(m.currentView, viewArea+":")
		return m.cmdLoadAreaTasks(id)
	}
	return nil
}

// refreshDetail batches checklist and activity loads for the currently selected task.
func (m Model) refreshDetail() tea.Cmd {
	if m.selectedTask == nil {
		return nil
	}
	id := m.selectedTask.ID
	return tea.Batch(
		m.cmdLoadChecklist(id),
		m.cmdLoadActivities(id),
	)
}

// --- SSE ---

func (m Model) cmdStartSSE() tea.Cmd {
	return func() tea.Msg {
		events, err := m.client.SubscribeEvents(context.Background(), "*")
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return SSEStartedMsg{Events: events}
	}
}

// cmdListenSSE reads one event from the channel and returns it as an SSEEventMsg.
// Call this again from Update after each SSEEventMsg to keep listening.
func cmdListenSSE(events <-chan client.DomainEvent) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-events
		if !ok {
			return SSEDisconnectedMsg{}
		}
		return SSEEventMsg{Event: evt}
	}
}

// handleSSEEvent decides what to refresh based on the event type prefix.
func (m Model) handleSSEEvent(evt client.DomainEvent) tea.Cmd {
	t := evt.Type
	switch {
	case strings.HasPrefix(t, "task."):
		return m.refreshCurrentView()
	case strings.HasPrefix(t, "checklist."), strings.HasPrefix(t, "activity."):
		return m.refreshDetail()
	case strings.HasPrefix(t, "project."):
		return m.cmdLoadProjects()
	case strings.HasPrefix(t, "area."):
		return m.cmdLoadAreas()
	case strings.HasPrefix(t, "tag."):
		return m.cmdLoadTags()
	case strings.HasPrefix(t, "location."):
		return m.cmdLoadLocations()
	default:
		return nil
	}
}
