use dioxus::prelude::*;

mod api;
mod components;
mod state;
mod views;

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
            div { class: "sidebar",
                div { class: "sidebar-drag-region" }
                p { class: "sidebar-group-label", "atask" }
            }
            div { class: "app-main",
                div { class: "app-toolbar",
                    span { class: "app-view-title", "Today" }
                }
                div { class: "app-content",
                    p { "Tasks will appear here." }
                }
            }
        }
    }
}
