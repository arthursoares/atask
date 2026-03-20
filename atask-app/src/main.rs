use dioxus::prelude::*;

mod api;
mod components;
mod state;
mod views;

use components::sidebar::Sidebar;
use components::toolbar::Toolbar;
use state::navigation::ActiveView;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    let active_view = use_signal(|| ActiveView::Today);
    let selected_task_id: Signal<Option<String>> = use_signal(|| None);
    use_context_provider(|| active_view);
    use_context_provider(|| selected_task_id);

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
                        ActiveView::Project(_) => rsx! {
                            div { class: "empty-state",
                                p { "Project view coming soon." }
                            }
                        },
                    }
                }
            }
        }
    }
}
