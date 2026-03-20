use dioxus::prelude::*;
use futures_util::StreamExt;

mod api;
mod components;
mod state;
mod views;

use api::client::ApiClient;
use api::sse::{self, SseParsedEvent};
use components::command_palette::CommandPalette;
use components::sidebar::Sidebar;
use components::toolbar::Toolbar;
use state::command::CommandState;
use state::navigation::ActiveView;
use state::tasks::TaskState;
use state::projects::ProjectState;
use views::login::LoginToken;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    let api_url = std::env::var("ATASK_API_URL")
        .unwrap_or_else(|_| "http://localhost:8080".to_string());

    // Load saved credentials
    let saved = state::credentials::load();
    let mut initial_api = ApiClient::new(&api_url);
    if let Some(ref tok) = saved.token {
        initial_api.set_token(tok.clone());
    }
    let api: Signal<ApiClient> = use_signal(|| initial_api);
    let login_token = LoginToken(use_signal(|| saved.token));
    let task_state: Signal<TaskState> = use_signal(|| TaskState::default());
    let project_state: Signal<ProjectState> = use_signal(|| ProjectState::default());
    let active_view = use_signal(|| ActiveView::Today);
    let selected_task_id: Signal<Option<String>> = use_signal(|| None);
    let command_state: Signal<CommandState> = use_signal(|| CommandState::default());
    let selected_list_index: Signal<Option<usize>> = use_signal(|| None);

    use_context_provider(|| api);
    use_context_provider(|| login_token);
    use_context_provider(|| task_state);
    use_context_provider(|| project_state);
    use_context_provider(|| active_view);
    use_context_provider(|| selected_task_id);
    use_context_provider(|| command_state);

    // Load initial data when token becomes available
    let _data_loader = use_effect(move || {
        let tok = login_token.0.read().clone();
        println!("[DATA] Effect fired. token present: {}", tok.is_some());
        if tok.is_some() {
            let api_clone = api.read().clone();
            println!("[DATA] Loading initial data...");
            let mut ts = task_state;
            let mut ps = project_state;
            spawn(async move {
                ts.write().loading.set(true);

                let (inbox, today, upcoming, someday, logbook, projects, areas) = tokio::join!(
                    api_clone.list_inbox(),
                    api_clone.list_today(),
                    api_clone.list_upcoming(),
                    api_clone.list_someday(),
                    api_clone.list_logbook(),
                    api_clone.list_projects(),
                    api_clone.list_areas(),
                );

                if let Ok(tasks) = inbox {
                    ts.write().inbox.set(tasks);
                }
                if let Ok(tasks) = today {
                    ts.write().today.set(tasks);
                }
                if let Ok(tasks) = upcoming {
                    ts.write().upcoming.set(tasks);
                }
                if let Ok(tasks) = someday {
                    ts.write().someday.set(tasks);
                }
                if let Ok(tasks) = logbook {
                    ts.write().logbook.set(tasks);
                }
                if let Ok(p) = projects {
                    ps.write().projects.set(p);
                }
                if let Ok(a) = areas {
                    ps.write().areas.set(a);
                }

                println!("[DATA] Initial data loaded");
                ts.write().loading.set(false);
            });
        }
    });

    // SSE real-time updates — runs for the lifetime of the authenticated session
    let _sse_coroutine = use_coroutine(move |_rx: UnboundedReceiver<()>| {
        async move {
            let mut last_id: Option<String> = None;
            loop {
                // Wait until we have a token.
                let (base, tok) = {
                    let api_ref = api.read();
                    let t = api_ref.token().map(|s| s.to_string());
                    (api_ref.base_url().to_string(), t)
                };

                let Some(tok) = tok else {
                    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                    continue;
                };

                match sse::connect_sse(&base, &tok, last_id.as_deref()).await {
                    Ok(mut stream) => {
                        while let Some(event) = stream.next().await {
                            match event {
                                Ok(evt) => {
                                    if evt.id.is_some() {
                                        last_id = evt.id.clone();
                                    }
                                    handle_sse_event(
                                        &evt,
                                        api,
                                        task_state,
                                        project_state,
                                        active_view,
                                    )
                                    .await;
                                }
                                Err(_) => break, // Stream error, will reconnect
                            }
                        }
                    }
                    Err(e) => {
                        eprintln!("[sse] connection failed: {e}");
                    }
                }

                // Reconnect delay
                tokio::time::sleep(std::time::Duration::from_secs(2)).await;
            }
        }
    });

    rsx! {
        document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
        if login_token.0.read().is_none() {
            views::login::LoginView {}
        } else {
            div {
                class: "app-frame",
                tabindex: 0,
                onkeydown: move |evt: Event<KeyboardData>| {
                    handle_global_keydown(
                        evt,
                        command_state,
                        active_view,
                        selected_task_id,
                        selected_list_index,
                        task_state,
                        api,
                        project_state,
                    );
                },
                Sidebar {}
                div { class: "app-main",
                    Toolbar {}
                    div { class: "app-content",
                        match *active_view.read() {
                            ActiveView::Today => rsx! { views::today::TodayView {} },
                            ActiveView::Inbox => rsx! { views::inbox::InboxView {} },
                            ActiveView::Upcoming => rsx! { views::upcoming::UpcomingView {} },
                            ActiveView::Someday => rsx! { views::someday::SomedayView {} },
                            ActiveView::Logbook => rsx! { views::logbook::LogbookView {} },
                            ActiveView::Project(ref id) => rsx! {
                                views::project::ProjectView { project_id: id.clone() }
                            },
                        }
                    }
                }
                if selected_task_id.read().is_some() {
                    components::task_detail::TaskDetail {}
                }
                if *command_state.read().open.read() {
                    CommandPalette {}
                }
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Global keyboard handler
// ---------------------------------------------------------------------------

fn get_current_task_list(
    task_state: Signal<TaskState>,
    project_state: Signal<ProjectState>,
    active_view: Signal<ActiveView>,
) -> Vec<crate::api::types::Task> {
    let view = active_view.read().clone();
    match view {
        ActiveView::Inbox => task_state.read().inbox.read().clone(),
        ActiveView::Today => task_state.read().today.read().clone(),
        ActiveView::Upcoming => task_state.read().upcoming.read().clone(),
        ActiveView::Someday => task_state.read().someday.read().clone(),
        ActiveView::Logbook => task_state.read().logbook.read().clone(),
        ActiveView::Project(ref id) => project_state
            .read()
            .project_tasks
            .read()
            .get(id)
            .cloned()
            .unwrap_or_default(),
    }
}

fn handle_global_keydown(
    evt: Event<KeyboardData>,
    mut command_state: Signal<CommandState>,
    mut active_view: Signal<ActiveView>,
    mut selected_task_id: Signal<Option<String>>,
    mut selected_list_index: Signal<Option<usize>>,
    task_state: Signal<TaskState>,
    api: Signal<ApiClient>,
    project_state: Signal<ProjectState>,
) {
    let key = evt.key();
    let modifiers = evt.modifiers();
    let meta = modifiers.contains(Modifiers::META);
    let shift = modifiers.contains(Modifiers::SHIFT);
    let palette_open = *command_state.read().open.read();

    // When command palette is open, let it handle its own keys
    if palette_open {
        // Only handle Escape globally to close the palette
        if key == Key::Escape {
            evt.prevent_default();
            command_state.write().open.set(false);
            command_state.write().query.set(String::new());
            command_state.write().selected_index.set(0);
        }
        return;
    }

    // ⌘K — toggle command palette
    if meta && key == Key::Character("k".into()) {
        evt.prevent_default();
        let is_open = *command_state.read().open.read();
        command_state.write().open.set(!is_open);
        if is_open {
            command_state.write().query.set(String::new());
            command_state.write().selected_index.set(0);
        }
        return;
    }

    // ⌘N — create new task
    if meta && !shift && key == Key::Character("n".into()) {
        evt.prevent_default();
        let api_clone = api.read().clone();
        let mut ts = task_state;
        spawn(async move {
            if let Ok(task) = api_clone.create_task("New task").await {
                let mut inbox = ts.read().inbox.read().clone();
                inbox.push(task);
                ts.write().inbox.set(inbox);
            }
        });
        return;
    }

    // ⌘1-5 — navigation
    if meta && !shift {
        let nav = match &key {
            Key::Character(c) if c == "1" => Some(ActiveView::Inbox),
            Key::Character(c) if c == "2" => Some(ActiveView::Today),
            Key::Character(c) if c == "3" => Some(ActiveView::Upcoming),
            Key::Character(c) if c == "4" => Some(ActiveView::Someday),
            Key::Character(c) if c == "5" => Some(ActiveView::Logbook),
            _ => None,
        };
        if let Some(view) = nav {
            evt.prevent_default();
            active_view.set(view);
            selected_list_index.set(None);
            selected_task_id.set(None);
            return;
        }
    }

    // Escape — close detail panel or deselect task
    if key == Key::Escape {
        evt.prevent_default();
        if selected_task_id.read().is_some() {
            selected_task_id.set(None);
        }
        selected_list_index.set(None);
        return;
    }

    // Task-level shortcuts (when a task is selected)
    let current_task_id = selected_task_id.read().clone();
    if let Some(tid) = current_task_id {
        // ⌘⇧C — complete task
        if meta && shift && key == Key::Character("c".into()) {
            evt.prevent_default();
            let api_clone = api.read().clone();
            let task_id = tid.clone();
            spawn(async move {
                let _ = api_clone.complete_task(&task_id).await;
            });
            return;
        }

        // ⌘T — schedule for today
        if meta && !shift && key == Key::Character("t".into()) {
            evt.prevent_default();
            let api_clone = api.read().clone();
            let task_id = tid.clone();
            spawn(async move {
                let _ = api_clone.update_task_schedule(&task_id, "today").await;
            });
            return;
        }

        // Backspace / Delete — delete task
        if key == Key::Backspace || key == Key::Delete {
            evt.prevent_default();
            let api_clone = api.read().clone();
            let task_id = tid.clone();
            selected_task_id.set(None);
            selected_list_index.set(None);
            spawn(async move {
                let _ = api_clone.delete_task(&task_id).await;
            });
            return;
        }

        // Space — toggle completion
        if key == Key::Character(" ".into()) {
            evt.prevent_default();
            let api_clone = api.read().clone();
            let task_id = tid.clone();
            spawn(async move {
                let _ = api_clone.complete_task(&task_id).await;
            });
            return;
        }
    }

    // Arrow key navigation in task list
    let tasks = get_current_task_list(task_state, project_state, active_view);

    if !tasks.is_empty() {
        match key {
            Key::ArrowDown => {
                evt.prevent_default();
                let current = selected_list_index.read().unwrap_or(0);
                let next = if selected_list_index.read().is_none() {
                    0
                } else if current + 1 >= tasks.len() {
                    tasks.len() - 1
                } else {
                    current + 1
                };
                selected_list_index.set(Some(next));
                if let Some(task) = tasks.get(next) {
                    selected_task_id.set(Some(task.id.clone()));
                }
            }
            Key::ArrowUp => {
                evt.prevent_default();
                let current = selected_list_index.read().unwrap_or(0);
                let next = if selected_list_index.read().is_none() {
                    0
                } else if current == 0 {
                    0
                } else {
                    current - 1
                };
                selected_list_index.set(Some(next));
                if let Some(task) = tasks.get(next) {
                    selected_task_id.set(Some(task.id.clone()));
                }
            }
            Key::Enter => {
                // Open detail panel — selected_task_id is already set by arrow navigation
                // No additional action needed; the detail panel shows when selected_task_id is Some
            }
            _ => {}
        }
    }
}

// ---------------------------------------------------------------------------
// SSE event handlers
// ---------------------------------------------------------------------------

async fn handle_sse_event(
    evt: &SseParsedEvent,
    api: Signal<ApiClient>,
    task_state: Signal<TaskState>,
    mut project_state: Signal<ProjectState>,
    active_view: Signal<ActiveView>,
) {
    match evt.event_type.as_str() {
        // Task events — refetch the active view's tasks
        "task.created"
        | "task.completed"
        | "task.cancelled"
        | "task.deleted"
        | "task.scheduled_today"
        | "task.deferred"
        | "task.moved_to_inbox"
        | "task.moved_to_project"
        | "task.removed_from_project" => {
            refresh_active_view(api, task_state, project_state, active_view).await;
        }

        // Project events — refetch the project list
        "project.created" | "project.deleted" | "project.completed" => {
            let client = api.read().clone();
            if let Ok(projects) = client.list_projects().await {
                project_state.write().projects.set(projects);
            }
        }

        // Activity events are informational; a task detail refresh could go here
        // if the detail panel tracked the viewed task id. For now, skip.
        _ => {}
    }
}

async fn refresh_active_view(
    api: Signal<ApiClient>,
    mut task_state: Signal<TaskState>,
    mut project_state: Signal<ProjectState>,
    active_view: Signal<ActiveView>,
) {
    let client = api.read().clone();
    let view = active_view.read().clone();

    match view {
        ActiveView::Inbox => {
            if let Ok(tasks) = client.list_inbox().await {
                task_state.write().inbox.set(tasks);
            }
        }
        ActiveView::Today => {
            if let Ok(tasks) = client.list_today().await {
                task_state.write().today.set(tasks);
            }
        }
        ActiveView::Upcoming => {
            if let Ok(tasks) = client.list_upcoming().await {
                task_state.write().upcoming.set(tasks);
            }
        }
        ActiveView::Someday => {
            if let Ok(tasks) = client.list_someday().await {
                task_state.write().someday.set(tasks);
            }
        }
        ActiveView::Logbook => {
            if let Ok(tasks) = client.list_logbook().await {
                task_state.write().logbook.set(tasks);
            }
        }
        ActiveView::Project(ref id) => {
            if let Ok(tasks) = client.list_tasks_by_project(id).await {
                project_state
                    .write()
                    .project_tasks
                    .write()
                    .insert(id.clone(), tasks);
            }
        }
    }
}
