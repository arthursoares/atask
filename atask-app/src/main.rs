use dioxus::prelude::*;

mod api;
mod components;
mod state;
mod views;

use api::client::ApiClient;
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
