use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct ButtonProps {
    label: String,
    #[props(default = "primary".to_string())]
    variant: String,
    on_click: EventHandler<()>,
}

#[component]
pub fn Button(props: ButtonProps) -> Element {
    let class = match props.variant.as_str() {
        "secondary" => "btn btn-secondary",
        "ghost" => "btn btn-ghost",
        "danger" => "btn btn-danger",
        _ => "btn btn-primary",
    };

    rsx! {
        button {
            class,
            onclick: move |_| props.on_click.call(()),
            "{props.label}"
        }
    }
}
