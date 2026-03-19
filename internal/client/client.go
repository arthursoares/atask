package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Task mirrors the domain.Task JSON response.
type Task struct {
	ID          string   `json:"ID"`
	Title       string   `json:"Title"`
	Notes       string   `json:"Notes"`
	Status      int      `json:"Status"`
	Schedule    int      `json:"Schedule"`
	StartDate   *string  `json:"StartDate"`
	Deadline    *string  `json:"Deadline"`
	CompletedAt *string  `json:"CompletedAt"`
	Index       int      `json:"Index"`
	TodayIndex  *int     `json:"TodayIndex"`
	ProjectID   *string  `json:"ProjectID"`
	SectionID   *string  `json:"SectionID"`
	AreaID      *string  `json:"AreaID"`
	LocationID  *string  `json:"LocationID"`
	Tags        []string `json:"Tags"`
	CreatedAt   string   `json:"CreatedAt"`
	UpdatedAt   string   `json:"UpdatedAt"`
}

// Project mirrors the domain.Project JSON response.
type Project struct {
	ID           string   `json:"ID"`
	Title        string   `json:"Title"`
	Notes        string   `json:"Notes"`
	Status       int      `json:"Status"`
	Schedule     int      `json:"Schedule"`
	StartDate    *string  `json:"StartDate"`
	Deadline     *string  `json:"Deadline"`
	CompletedAt  *string  `json:"CompletedAt"`
	Index        int      `json:"Index"`
	AreaID       *string  `json:"AreaID"`
	Tags         []string `json:"Tags"`
	AutoComplete bool     `json:"AutoComplete"`
	CreatedAt    string   `json:"CreatedAt"`
	UpdatedAt    string   `json:"UpdatedAt"`
}

// Area mirrors the domain.Area JSON response.
type Area struct {
	ID        string `json:"ID"`
	Title     string `json:"Title"`
	Index     int    `json:"Index"`
	Archived  bool   `json:"Archived"`
	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

// Tag mirrors the domain.Tag JSON response.
type Tag struct {
	ID        string  `json:"ID"`
	Title     string  `json:"Title"`
	ParentID  *string `json:"ParentID"`
	Shortcut  *string `json:"Shortcut"`
	Index     int     `json:"Index"`
	CreatedAt string  `json:"CreatedAt"`
	UpdatedAt string  `json:"UpdatedAt"`
}

// Location mirrors the domain.Location JSON response.
type Location struct {
	ID        string   `json:"ID"`
	Name      string   `json:"Name"`
	Latitude  *float64 `json:"Latitude"`
	Longitude *float64 `json:"Longitude"`
	Radius    *int     `json:"Radius"`
	Address   *string  `json:"Address"`
	CreatedAt string   `json:"CreatedAt"`
	UpdatedAt string   `json:"UpdatedAt"`
}

// ChecklistItem mirrors the domain.ChecklistItem JSON response.
type ChecklistItem struct {
	ID        string `json:"ID"`
	Title     string `json:"Title"`
	Status    int    `json:"Status"`
	TaskID    string `json:"TaskID"`
	Index     int    `json:"Index"`
	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

// Activity mirrors the domain.Activity JSON response.
type Activity struct {
	ID        string `json:"ID"`
	TaskID    string `json:"TaskID"`
	ActorID   string `json:"ActorID"`
	ActorType string `json:"ActorType"`
	Type      string `json:"Type"`
	Content   string `json:"Content"`
	CreatedAt string `json:"CreatedAt"`
}

// Section mirrors the domain.Section JSON response.
type Section struct {
	ID        string `json:"ID"`
	Title     string `json:"Title"`
	ProjectID string `json:"ProjectID"`
	Index     int    `json:"Index"`
	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

// Client is an HTTP client for the atask REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a new Client with the given base URL and token.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

// SetToken updates the authentication token used by the client.
func (c *Client) SetToken(token string) {
	c.token = token
}

// doJSON sends a request and decodes the response body directly into result.
func (c *Client) doJSON(ctx context.Context, method, path string, body, result any) error {
	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error != "" {
			return fmt.Errorf("api error %d: %s", resp.StatusCode, errBody.Error)
		}
		return fmt.Errorf("api error %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// eventEnvelope is the wire format for event responses.
type eventEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// doEvent sends a mutation request, decodes the event envelope, and extracts the data field into result.
func (c *Client) doEvent(ctx context.Context, method, path string, body, result any) error {
	var envelope eventEnvelope
	if err := c.doJSON(ctx, method, path, body, &envelope); err != nil {
		return err
	}

	if result != nil && envelope.Data != nil {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("decode event data: %w", err)
		}
	}

	return nil
}

// --- Auth ---

// Register creates a new user account and immediately logs in, returning a token.
func (c *Client) Register(ctx context.Context, email, password, name string) (string, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	}
	if err := c.doJSON(ctx, http.MethodPost, "/auth/register", body, nil); err != nil {
		return "", err
	}
	return c.Login(ctx, email, password)
}

// Login authenticates and returns a JWT token.
func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/auth/login", body, &resp); err != nil {
		return "", err
	}
	return resp.Token, nil
}

// GetMe returns the current user profile. Useful for validating stored tokens.
func (c *Client) GetMe(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	if err := c.doJSON(ctx, http.MethodGet, "/auth/me", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// --- Views ---

// ListInbox returns tasks in the inbox view.
func (c *Client) ListInbox(ctx context.Context) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/views/inbox", nil, &tasks)
}

// ListToday returns tasks in the today view.
func (c *Client) ListToday(ctx context.Context) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/views/today", nil, &tasks)
}

// ListUpcoming returns tasks in the upcoming view.
func (c *Client) ListUpcoming(ctx context.Context) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/views/upcoming", nil, &tasks)
}

// ListSomeday returns tasks in the someday view.
func (c *Client) ListSomeday(ctx context.Context) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/views/someday", nil, &tasks)
}

// ListLogbook returns completed/cancelled tasks.
func (c *Client) ListLogbook(ctx context.Context) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/views/logbook", nil, &tasks)
}

// --- Tasks ---

// CreateTask creates a new task with the given title.
func (c *Client) CreateTask(ctx context.Context, title string) (*Task, error) {
	body := map[string]string{"title": title}
	var task Task
	return &task, c.doEvent(ctx, http.MethodPost, "/tasks", body, &task)
}

// GetTask fetches a task by ID.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
	var task Task
	return &task, c.doJSON(ctx, http.MethodGet, "/tasks/"+id, nil, &task)
}

// CompleteTask marks a task as complete.
func (c *Client) CompleteTask(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodPost, "/tasks/"+id+"/complete", nil, nil)
}

// CancelTask marks a task as cancelled.
func (c *Client) CancelTask(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodPost, "/tasks/"+id+"/cancel", nil, nil)
}

// DeleteTask soft-deletes a task.
func (c *Client) DeleteTask(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodDelete, "/tasks/"+id, nil, nil)
}

// UpdateTaskTitle updates the title of a task.
func (c *Client) UpdateTaskTitle(ctx context.Context, id, title string) error {
	body := map[string]string{"title": title}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/title", body, nil)
}

// UpdateTaskNotes updates the notes of a task.
func (c *Client) UpdateTaskNotes(ctx context.Context, id, notes string) error {
	body := map[string]string{"notes": notes}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/notes", body, nil)
}

// UpdateTaskSchedule updates the schedule of a task (e.g. "inbox", "anytime", "someday").
func (c *Client) UpdateTaskSchedule(ctx context.Context, id, schedule string) error {
	body := map[string]string{"schedule": schedule}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/schedule", body, nil)
}

// SetTaskStartDate sets or clears the start date of a task.
func (c *Client) SetTaskStartDate(ctx context.Context, id string, date *string) error {
	body := map[string]*string{"date": date}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/start-date", body, nil)
}

// SetTaskDeadline sets or clears the deadline of a task.
func (c *Client) SetTaskDeadline(ctx context.Context, id string, date *string) error {
	body := map[string]*string{"date": date}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/deadline", body, nil)
}

// MoveTaskToProject moves a task to a project (or removes from project if projectID is nil).
func (c *Client) MoveTaskToProject(ctx context.Context, id string, projectID *string) error {
	body := map[string]*string{"id": projectID}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/project", body, nil)
}

// MoveTaskToSection moves a task to a section (or removes from section if sectionID is nil).
func (c *Client) MoveTaskToSection(ctx context.Context, id string, sectionID *string) error {
	body := map[string]*string{"id": sectionID}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/section", body, nil)
}

// MoveTaskToArea moves a task to an area (or removes from area if areaID is nil).
func (c *Client) MoveTaskToArea(ctx context.Context, id string, areaID *string) error {
	body := map[string]*string{"id": areaID}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/area", body, nil)
}

// SetTaskLocation sets or clears the location of a task.
func (c *Client) SetTaskLocation(ctx context.Context, id string, locationID *string) error {
	body := map[string]*string{"id": locationID}
	return c.doEvent(ctx, http.MethodPut, "/tasks/"+id+"/location", body, nil)
}

// AddTaskTag adds a tag to a task.
func (c *Client) AddTaskTag(ctx context.Context, taskID, tagID string) error {
	return c.doEvent(ctx, http.MethodPost, "/tasks/"+taskID+"/tags/"+tagID, nil, nil)
}

// RemoveTaskTag removes a tag from a task.
func (c *Client) RemoveTaskTag(ctx context.Context, taskID, tagID string) error {
	return c.doEvent(ctx, http.MethodDelete, "/tasks/"+taskID+"/tags/"+tagID, nil, nil)
}

// ListTasksByProject returns tasks belonging to a project.
func (c *Client) ListTasksByProject(ctx context.Context, projectID string) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/tasks?project_id="+projectID, nil, &tasks)
}

// ListTasksByArea returns tasks belonging to an area.
func (c *Client) ListTasksByArea(ctx context.Context, areaID string) ([]Task, error) {
	var tasks []Task
	return tasks, c.doJSON(ctx, http.MethodGet, "/tasks?area_id="+areaID, nil, &tasks)
}

// --- Projects ---

// CreateProject creates a new project with the given title.
func (c *Client) CreateProject(ctx context.Context, title string) (*Project, error) {
	body := map[string]string{"title": title}
	var project Project
	return &project, c.doEvent(ctx, http.MethodPost, "/projects", body, &project)
}

// ListProjects returns all projects.
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	return projects, c.doJSON(ctx, http.MethodGet, "/projects", nil, &projects)
}

// GetProject fetches a project by ID.
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var project Project
	return &project, c.doJSON(ctx, http.MethodGet, "/projects/"+id, nil, &project)
}

// CompleteProject marks a project as complete.
func (c *Client) CompleteProject(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodPost, "/projects/"+id+"/complete", nil, nil)
}

// CancelProject marks a project as cancelled.
func (c *Client) CancelProject(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodPost, "/projects/"+id+"/cancel", nil, nil)
}

// DeleteProject soft-deletes a project.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodDelete, "/projects/"+id, nil, nil)
}

// UpdateProjectTitle updates the title of a project.
func (c *Client) UpdateProjectTitle(ctx context.Context, id, title string) error {
	body := map[string]string{"title": title}
	return c.doEvent(ctx, http.MethodPut, "/projects/"+id+"/title", body, nil)
}

// --- Areas ---

// CreateArea creates a new area with the given title.
func (c *Client) CreateArea(ctx context.Context, title string) (*Area, error) {
	body := map[string]string{"title": title}
	var area Area
	return &area, c.doEvent(ctx, http.MethodPost, "/areas", body, &area)
}

// ListAreas returns all non-archived areas.
func (c *Client) ListAreas(ctx context.Context) ([]Area, error) {
	var areas []Area
	return areas, c.doJSON(ctx, http.MethodGet, "/areas", nil, &areas)
}

// RenameArea updates the title of an area.
func (c *Client) RenameArea(ctx context.Context, id, title string) error {
	body := map[string]string{"title": title}
	return c.doEvent(ctx, http.MethodPut, "/areas/"+id, body, nil)
}

// ArchiveArea archives an area.
func (c *Client) ArchiveArea(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodPost, "/areas/"+id+"/archive", nil, nil)
}

// DeleteArea soft-deletes an area. If cascade is true, also deletes child tasks and projects.
func (c *Client) DeleteArea(ctx context.Context, id string, cascade bool) error {
	path := "/areas/" + id
	if cascade {
		path += "?cascade=true"
	}
	return c.doEvent(ctx, http.MethodDelete, path, nil, nil)
}

// --- Tags ---

// CreateTag creates a new tag with the given title.
func (c *Client) CreateTag(ctx context.Context, title string) (*Tag, error) {
	body := map[string]string{"title": title}
	var tag Tag
	return &tag, c.doEvent(ctx, http.MethodPost, "/tags", body, &tag)
}

// ListTags returns all tags.
func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	return tags, c.doJSON(ctx, http.MethodGet, "/tags", nil, &tags)
}

// RenameTag updates the title of a tag.
func (c *Client) RenameTag(ctx context.Context, id, title string) error {
	body := map[string]string{"title": title}
	return c.doEvent(ctx, http.MethodPut, "/tags/"+id, body, nil)
}

// DeleteTag deletes a tag.
func (c *Client) DeleteTag(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodDelete, "/tags/"+id, nil, nil)
}

// --- Locations ---

// CreateLocation creates a new location with the given name.
func (c *Client) CreateLocation(ctx context.Context, name string) (*Location, error) {
	body := map[string]string{"name": name}
	var location Location
	return &location, c.doEvent(ctx, http.MethodPost, "/locations", body, &location)
}

// ListLocations returns all locations.
func (c *Client) ListLocations(ctx context.Context) ([]Location, error) {
	var locations []Location
	return locations, c.doJSON(ctx, http.MethodGet, "/locations", nil, &locations)
}

// DeleteLocation deletes a location.
func (c *Client) DeleteLocation(ctx context.Context, id string) error {
	return c.doEvent(ctx, http.MethodDelete, "/locations/"+id, nil, nil)
}

// --- Sections ---

// CreateSection creates a new section in a project.
func (c *Client) CreateSection(ctx context.Context, projectID, title string) (*Section, error) {
	body := map[string]string{"title": title}
	var section Section
	return &section, c.doEvent(ctx, http.MethodPost, "/projects/"+projectID+"/sections", body, &section)
}

// ListSections returns all sections in a project.
func (c *Client) ListSections(ctx context.Context, projectID string) ([]Section, error) {
	var sections []Section
	return sections, c.doJSON(ctx, http.MethodGet, "/projects/"+projectID+"/sections", nil, &sections)
}

// --- Checklist ---

// AddChecklistItem adds a checklist item to a task.
func (c *Client) AddChecklistItem(ctx context.Context, taskID, title string) (*ChecklistItem, error) {
	body := map[string]string{"title": title}
	var item ChecklistItem
	return &item, c.doEvent(ctx, http.MethodPost, "/tasks/"+taskID+"/checklist", body, &item)
}

// ListChecklistItems returns all checklist items for a task.
func (c *Client) ListChecklistItems(ctx context.Context, taskID string) ([]ChecklistItem, error) {
	var items []ChecklistItem
	return items, c.doJSON(ctx, http.MethodGet, "/tasks/"+taskID+"/checklist", nil, &items)
}

// CompleteChecklistItem marks a checklist item as complete.
func (c *Client) CompleteChecklistItem(ctx context.Context, taskID, itemID string) error {
	return c.doEvent(ctx, http.MethodPost, "/tasks/"+taskID+"/checklist/"+itemID+"/complete", nil, nil)
}

// UncompleteChecklistItem marks a checklist item as incomplete.
func (c *Client) UncompleteChecklistItem(ctx context.Context, taskID, itemID string) error {
	return c.doEvent(ctx, http.MethodPost, "/tasks/"+taskID+"/checklist/"+itemID+"/uncomplete", nil, nil)
}

// DeleteChecklistItem removes a checklist item from a task.
func (c *Client) DeleteChecklistItem(ctx context.Context, taskID, itemID string) error {
	return c.doEvent(ctx, http.MethodDelete, "/tasks/"+taskID+"/checklist/"+itemID, nil, nil)
}

// --- Activity ---

// AddActivity records an activity on a task.
func (c *Client) AddActivity(ctx context.Context, taskID, actorType, activityType, content string) (*Activity, error) {
	body := map[string]string{
		"actor_type": actorType,
		"type":       activityType,
		"content":    content,
	}
	var activity Activity
	return &activity, c.doEvent(ctx, http.MethodPost, "/tasks/"+taskID+"/activity", body, &activity)
}

// ListActivities returns all activities for a task.
func (c *Client) ListActivities(ctx context.Context, taskID string) ([]Activity, error) {
	var activities []Activity
	return activities, c.doJSON(ctx, http.MethodGet, "/tasks/"+taskID+"/activity", nil, &activities)
}
