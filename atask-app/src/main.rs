use dioxus::prelude::*;
use futures_util::StreamExt;

mod api;
mod components;
mod state;
mod views;

use api::client::ApiClient;
use api::sse::{self, SseParsedEvent};
use components::sidebar::Sidebar;
use components::toolbar::Toolbar;
use state::navigation::ActiveView;
use state::tasks::TaskState;
use state::projects::ProjectState;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    let api_url = std::env::var("ATASK_API_URL")
        .unwrap_or_else(|_| "http://localhost:8080".to_string());

    let api: Signal<ApiClient> = use_signal(|| ApiClient::new(&api_url));
    let token: Signal<Option<String>> = use_signal(|| None);
    let task_state: Signal<TaskState> = use_signal(|| TaskState::default());
    let project_state: Signal<ProjectState> = use_signal(|| ProjectState::default());
    let active_view = use_signal(|| ActiveView::Today);
    let selected_task_id: Signal<Option<String>> = use_signal(|| None);

    use_context_provider(|| api);
    use_context_provider(|| token);
    use_context_provider(|| task_state);
    use_context_provider(|| project_state);
    use_context_provider(|| active_view);
    use_context_provider(|| selected_task_id);

    // Load initial data when token becomes available
    let _data_loader = use_effect(move || {
        let tok = token.read().clone();
        if tok.is_some() {
            let api_clone = api.read().clone();
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

    if token.read().is_none() {
        rsx! {
            document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
            views::login::LoginView {}
        }
    } else {
        rsx! {
            document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
            div { class: "app-frame",
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
            }
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
