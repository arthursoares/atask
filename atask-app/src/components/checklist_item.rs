use dioxus::prelude::*;

#[component]
pub fn ChecklistItemComponent(title: String, checked: bool, on_toggle: EventHandler<()>) -> Element {
    let check_class = if checked {
        "checklist-check checked"
    } else {
        "checklist-check"
    };
    let title_class = if checked {
        "checklist-title completed"
    } else {
        "checklist-title"
    };

    rsx! {
        div { class: "checklist-item",
            div {
                class: "{check_class}",
                onclick: move |_| on_toggle.call(()),
                if checked {
                    svg {
                        view_box: "0 0 10 10",
                        polyline { points: "1.5 5 4 7.5 8.5 2.5" }
                    }
                }
            }
            span { class: "{title_class}", "{title}" }
        }
    }
}
