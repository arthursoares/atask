use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct ChecklistItemProps {
    title: String,
    #[props(default = false)]
    checked: bool,
    on_toggle: EventHandler<()>,
}

#[component]
pub fn ChecklistItem(props: ChecklistItemProps) -> Element {
    let check_class = if props.checked {
        "detail-checklist-check done"
    } else {
        "detail-checklist-check"
    };

    rsx! {
        div { class: "detail-checklist-item",
            div {
                class: check_class,
                onclick: move |e| {
                    e.stop_propagation();
                    props.on_toggle.call(());
                },
                if props.checked {
                    svg {
                        view_box: "0 0 12 12",
                        polyline { points: "2.5 6 5 8.5 9.5 3.5" }
                    }
                }
            }
            span { "{props.title}" }
        }
    }
}
