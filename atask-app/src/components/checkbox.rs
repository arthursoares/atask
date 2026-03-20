use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct CheckboxProps {
    checked: bool,
    #[props(default = false)]
    today: bool,
    on_toggle: EventHandler<()>,
}

#[component]
pub fn Checkbox(props: CheckboxProps) -> Element {
    let mut class = "checkbox".to_string();
    if props.checked {
        class.push_str(" checked");
    }
    if props.today && !props.checked {
        class.push_str(" today");
    }

    rsx! {
        div {
            class,
            onclick: move |e| {
                e.stop_propagation();
                props.on_toggle.call(());
            },
            svg {
                view_box: "0 0 12 12",
                polyline { points: "2.5 6 5 8.5 9.5 3.5" }
            }
        }
    }
}
