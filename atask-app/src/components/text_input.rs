use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct TextInputProps {
    value: String,
    #[props(default = String::new())]
    placeholder: String,
    #[props(default = false)]
    ghost: bool,
    on_change: EventHandler<String>,
    on_submit: Option<EventHandler<String>>,
}

#[component]
pub fn TextInput(props: TextInputProps) -> Element {
    let class = if props.ghost { "input input-ghost" } else { "input" };

    rsx! {
        input {
            class,
            r#type: "text",
            value: "{props.value}",
            placeholder: "{props.placeholder}",
            oninput: move |e: Event<FormData>| {
                props.on_change.call(e.value());
            },
            onkeydown: move |e: Event<KeyboardData>| {
                if e.key() == Key::Enter {
                    if let Some(ref on_submit) = props.on_submit {
                        on_submit.call(props.value.clone());
                    }
                }
            },
        }
    }
}
