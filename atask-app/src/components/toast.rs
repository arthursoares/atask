use dioxus::prelude::*;

#[component]
pub fn Toast() -> Element {
    let toast: Signal<Option<String>> = use_context();

    rsx! {
        if let Some(ref msg) = *toast.read() {
            div { class: "toast", "{msg}" }
        }
    }
}
