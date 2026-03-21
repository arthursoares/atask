use std::collections::HashMap;
use dioxus::prelude::*;
use futures_util::StreamExt;

mod api;
mod state;
mod components;
mod views;

use api::client::ApiClient;
use api::sse::connect_sse;
use state::app::*;

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

    // Create ALL signals with newtypes
    let api = ApiSignal(use_signal(|| initial_api));
    let mut token = TokenSignal(use_signal(|| saved.token));
    let active_view = ViewSignal(use_signal(|| ActiveView::Today));
    let selected_task = SelectedTaskSignal(use_signal(|| None));
    let mut inbox = InboxTasks(use_signal(|| Vec::new()));
    let mut today = TodayTasks(use_signal(|| Vec::new()));
    let mut upcoming = UpcomingTasks(use_signal(|| Vec::new()));
    let mut someday = SomedayTasks(use_signal(|| Vec::new()));
    let mut logbook = LogbookTasks(use_signal(|| Vec::new()));
    let mut projects = ProjectList(use_signal(|| Vec::new()));
    let mut areas = AreaList(use_signal(|| Vec::new()));
    let mut tags = TagList(use_signal(|| Vec::new()));
    let mut loading = LoadingSignal(use_signal(|| false));
    let mut project_tasks = ProjectTasks(use_signal(|| HashMap::new()));
    let mut project_sections = ProjectSections(use_signal(|| HashMap::new()));
    let command_open = CommandOpen(use_signal(|| false));
    let command_query = CommandQuery(use_signal(|| String::new()));
    let command_index = CommandIndex(use_signal(|| 0usize));

    // Provide ALL via context
    use_context_provider(|| api);
    use_context_provider(|| token);
    use_context_provider(|| active_view);
    use_context_provider(|| selected_task);
    use_context_provider(|| inbox);
    use_context_provider(|| today);
    use_context_provider(|| upcoming);
    use_context_provider(|| someday);
    use_context_provider(|| logbook);
    use_context_provider(|| projects);
    use_context_provider(|| areas);
    use_context_provider(|| tags);
    use_context_provider(|| loading);
    use_context_provider(|| project_tasks);
    use_context_provider(|| project_sections);
    use_context_provider(|| command_open);
    use_context_provider(|| command_query);
    use_context_provider(|| command_index);

    // Load data when token becomes available
    use_effect(move || {
        // Read INSIDE the effect for tracking
        if token.0.read().is_some() {
            let api_clone = api.0.read().clone();
            spawn(async move {
                loading.0.set(true);
                let (inbox_r, today_r, upcoming_r, someday_r, logbook_r, projects_r, areas_r, tags_r) = tokio::join!(
                    api_clone.list_inbox(),
                    api_clone.list_today(),
                    api_clone.list_upcoming(),
                    api_clone.list_someday(),
                    api_clone.list_logbook(),
                    api_clone.list_projects(),
                    api_clone.list_areas(),
                    api_clone.list_tags(),
                );
                // Check for auth failures — if any request returns 401, clear session
                let is_unauthorized = inbox_r.as_ref().is_err_and(|e| e.is_unauthorized())
                    || today_r.as_ref().is_err_and(|e| e.is_unauthorized())
                    || projects_r.as_ref().is_err_and(|e| e.is_unauthorized());

                if is_unauthorized {
                    println!("[AUTH] Token expired or invalid — clearing session");
                    token.0.set(None);
                    state::credentials::clear();
                    loading.0.set(false);
                    return;
                }

                if let Ok(t) = inbox_r { inbox.0.set(t); }
                if let Ok(t) = today_r { today.0.set(t); }
                if let Ok(t) = upcoming_r { upcoming.0.set(t); }
                if let Ok(t) = someday_r { someday.0.set(t); }
                if let Ok(t) = logbook_r { logbook.0.set(t); }
                if let Ok(p) = projects_r { projects.0.set(p); }
                if let Ok(a) = areas_r { areas.0.set(a); }
                if let Ok(t) = tags_r { tags.0.set(t); }
                loading.0.set(false);
            });
        }
    });

    // SSE coroutine: reconnects on disconnect, refetches on events
    use_coroutine(move |_: UnboundedReceiver<()>| {
        async move {
            loop {
                // Wait until we have a token
                let (base, tok) = loop {
                    let tok_val = token.0.read().clone();
                    if let Some(t) = tok_val {
                        let base = api.0.read().base_url().to_string();
                        break (base, t);
                    }
                    // Poll every 500ms until token is available
                    tokio::time::sleep(std::time::Duration::from_millis(500)).await;
                };

                println!("[SSE] Connecting to {base}/events/stream...");
                match connect_sse(&base, &tok, None).await {
                    Ok(mut stream) => {
                        println!("[SSE] Connected");
                        while let Some(result) = stream.next().await {
                            match result {
                                Ok(evt) => {
                                    println!("[SSE] event: {} data: {}", evt.event_type, &evt.data[..evt.data.len().min(80)]);
                                    let api_clone = api.0.read().clone();
                                    let view = active_view.0.read().clone();

                                    if evt.event_type.starts_with("task.") {
                                        // Refetch active view
                                        match view {
                                            ActiveView::Inbox => {
                                                if let Ok(t) = api_clone.list_inbox().await { inbox.0.set(t); }
                                            }
                                            ActiveView::Today => {
                                                if let Ok(t) = api_clone.list_today().await { today.0.set(t); }
                                            }
                                            ActiveView::Upcoming => {
                                                if let Ok(t) = api_clone.list_upcoming().await { upcoming.0.set(t); }
                                            }
                                            ActiveView::Someday => {
                                                if let Ok(t) = api_clone.list_someday().await { someday.0.set(t); }
                                            }
                                            ActiveView::Logbook => {
                                                if let Ok(t) = api_clone.list_logbook().await { logbook.0.set(t); }
                                            }
                                            ActiveView::Project(ref pid) => {
                                                if let Ok(t) = api_clone.list_tasks_by_project(pid).await {
                                                    let mut map = project_tasks.0.read().clone();
                                                    map.insert(pid.clone(), t);
                                                    project_tasks.0.set(map);
                                                }
                                            }
                                        }
                                    } else if evt.event_type.starts_with("project.") {
                                        if let Ok(p) = api_clone.list_projects().await {
                                            projects.0.set(p);
                                        }
                                    } else if evt.event_type.starts_with("section.") {
                                        // Refetch sections for active project
                                        if let ActiveView::Project(ref pid) = view {
                                            if let Ok(s) = api_clone.list_sections(pid).await {
                                                let mut map = project_sections.0.read().clone();
                                                map.insert(pid.clone(), s);
                                                project_sections.0.set(map);
                                            }
                                        }
                                    }
                                }
                                Err(e) => {
                                    println!("[SSE] Stream error: {e}");
                                    break;
                                }
                            }
                        }
                        println!("[SSE] Stream ended, reconnecting in 2s...");
                    }
                    Err(e) => {
                        println!("[SSE] Connection failed: {e}, retrying in 2s...");
                    }
                }
                tokio::time::sleep(std::time::Duration::from_secs(2)).await;
            }
        }
    });

    // CRITICAL: token read INSIDE rsx!
    rsx! {
        document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
        document::Title { "atask" }
        if token.0.read().is_none() {
            views::login::LoginView {}
        } else {
            div {
                class: "app-frame",
                tabindex: 0,
                onkeydown: move |evt: Event<KeyboardData>| {
                    handle_keydown(evt, command_open, command_query, command_index, active_view, selected_task, api);
                },
                components::sidebar::Sidebar {}
                div { class: "app-main",
                    components::toolbar::Toolbar {}
                    div { class: "app-content",
                        match *active_view.0.read() {
                            ActiveView::Inbox => rsx! { views::inbox::InboxView {} },
                            ActiveView::Today => rsx! { views::today::TodayView {} },
                            ActiveView::Upcoming => rsx! { views::upcoming::UpcomingView {} },
                            ActiveView::Someday => rsx! { views::someday::SomedayView {} },
                            ActiveView::Logbook => rsx! { views::logbook::LogbookView {} },
                            ActiveView::Project(ref id) => rsx! { views::project::ProjectView { project_id: id.clone() } },
                        }
                    }
                }
                if selected_task.0.read().is_some() {
                    components::task_detail::TaskDetail {}
                }
                if *command_open.0.read() {
                    components::command_palette::CommandPalette {}
                }
            }
        }
    }
}

fn handle_keydown(
    evt: Event<KeyboardData>,
    mut command_open: CommandOpen,
    mut command_query: CommandQuery,
    mut command_index: CommandIndex,
    mut active_view: ViewSignal,
    mut selected_task: SelectedTaskSignal,
    api: ApiSignal,
) {
    let key = evt.key();
    let meta = evt.modifiers().meta();
    let shift = evt.modifiers().shift();

    // If palette is open, only intercept Escape (palette handles its own keys)
    if *command_open.0.read() {
        if key == Key::Escape {
            evt.prevent_default();
            command_open.0.set(false);
            command_query.0.set(String::new());
            command_index.0.set(0);
        }
        return;
    }

    // Cmd+K -- open palette
    if meta && key == Key::Character("k".into()) {
        evt.prevent_default();
        command_open.0.set(true);
        return;
    }

    // Cmd+1-5 -- navigation
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
            active_view.0.set(view);
            selected_task.0.set(None);
            return;
        }
    }

    // Cmd+N -- new task
    if meta && !shift && key == Key::Character("n".into()) {
        evt.prevent_default();
        let api_clone = api.0.read().clone();
        spawn(async move {
            let _ = api_clone.create_task("New task").await;
        });
        return;
    }

    // Escape -- close detail panel
    if key == Key::Escape {
        evt.prevent_default();
        if selected_task.0.read().is_some() {
            selected_task.0.set(None);
        }
        return;
    }

    // Task shortcuts (when task selected)
    let task_id = selected_task.0.read().clone();
    if let Some(tid) = task_id {
        // Cmd+Shift+C -- complete task
        if meta && shift && key == Key::Character("c".into()) {
            evt.prevent_default();
            let api_clone = api.0.read().clone();
            spawn(async move { let _ = api_clone.complete_task(&tid).await; });
            return;
        }
        // Cmd+T -- schedule for today
        if meta && !shift && key == Key::Character("t".into()) {
            evt.prevent_default();
            let api_clone = api.0.read().clone();
            let tid = tid.clone();
            spawn(async move { let _ = api_clone.update_task_schedule(&tid, "anytime").await; });
            return;
        }
        // Space -- complete task
        if key == Key::Character(" ".into()) {
            evt.prevent_default();
            let api_clone = api.0.read().clone();
            let tid = tid.clone();
            spawn(async move { let _ = api_clone.complete_task(&tid).await; });
            return;
        }
        // Backspace/Delete -- delete task
        if key == Key::Backspace || key == Key::Delete {
            evt.prevent_default();
            let api_clone = api.0.read().clone();
            selected_task.0.set(None);
            spawn(async move { let _ = api_clone.delete_task(&tid).await; });
            return;
        }
    }
}
