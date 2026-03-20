use dioxus::prelude::*;

mod api;
mod state;
mod components;
mod views;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    rsx! {
        document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
        div { class: "app-frame",
            p { "atask v2 — scaffold works" }
        }
    }
}
