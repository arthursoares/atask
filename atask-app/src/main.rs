use dioxus::prelude::*;

mod api;
mod state;
mod components;
mod views;

use api::client::ApiClient;
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
    let token = TokenSignal(use_signal(|| saved.token));
    let active_view = ViewSignal(use_signal(|| ActiveView::Today));
    let selected_task = SelectedTaskSignal(use_signal(|| None));
    let mut inbox = InboxTasks(use_signal(|| Vec::new()));
    let mut today = TodayTasks(use_signal(|| Vec::new()));
    let upcoming = UpcomingTasks(use_signal(|| Vec::new()));
    let someday = SomedayTasks(use_signal(|| Vec::new()));
    let logbook = LogbookTasks(use_signal(|| Vec::new()));
    let mut projects = ProjectList(use_signal(|| Vec::new()));
    let mut areas = AreaList(use_signal(|| Vec::new()));
    let mut tags = TagList(use_signal(|| Vec::new()));
    let mut loading = LoadingSignal(use_signal(|| false));

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

    // Load data when token becomes available
    use_effect(move || {
        // Read INSIDE the effect for tracking
        if token.0.read().is_some() {
            let api_clone = api.0.read().clone();
            spawn(async move {
                loading.0.set(true);
                let (inbox_r, today_r, projects_r, areas_r, tags_r) = tokio::join!(
                    api_clone.list_inbox(),
                    api_clone.list_today(),
                    api_clone.list_projects(),
                    api_clone.list_areas(),
                    api_clone.list_tags(),
                );
                if let Ok(t) = inbox_r { inbox.0.set(t); }
                if let Ok(t) = today_r { today.0.set(t); }
                if let Ok(p) = projects_r { projects.0.set(p); }
                if let Ok(a) = areas_r { areas.0.set(a); }
                if let Ok(t) = tags_r { tags.0.set(t); }
                loading.0.set(false);
            });
        }
    });

    // CRITICAL: token read INSIDE rsx!
    rsx! {
        document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
        if token.0.read().is_none() {
            views::login::LoginView {}
        } else {
            div { class: "app-frame",
                p { "Logged in! Sidebar + Today view coming in Tasks 4-6." }
            }
        }
    }
}
