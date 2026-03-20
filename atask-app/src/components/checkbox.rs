use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct CheckboxProps {
    checked: bool,
    today: bool,
    on_toggle: EventHandler<()>,
}

#[component]
pub fn Checkbox(props: CheckboxProps) -> Element {
    let class = {
        let mut c = "checkbox".to_string();
        if props.today {
            c.push_str(" today");
        }
        if props.checked {
            c.push_str(" checked");
        }
        c
    };

    rsx! {
        div {
            class: "{class}",
            onclick: move |evt| {
                evt.stop_propagation();
                props.on_toggle.call(());
            },
            svg {
                view_box: "0 0 12 12",
                xmlns: "http://www.w3.org/2000/svg",
                polyline { points: "2.5 6 5 8.5 9.5 3.5" }
            }
        }
    }
}
