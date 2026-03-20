use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct NewTaskInlineProps {
    on_create: EventHandler<String>,
}

#[component]
pub fn NewTaskInline(props: NewTaskInlineProps) -> Element {
    let mut editing = use_signal(|| false);
    let mut input_value = use_signal(|| String::new());

    if *editing.read() {
        rsx! {
            div { class: "new-task-inline",
                div { class: "new-task-plus", "+" }
                input {
                    class: "input",
                    placeholder: "Task title…",
                    value: "{input_value}",
                    autofocus: true,
                    oninput: move |e: Event<FormData>| {
                        input_value.set(e.value());
                    },
                    onkeydown: move |e: Event<KeyboardData>| {
                        if e.key() == Key::Enter {
                            let val = input_value.read().trim().to_string();
                            if !val.is_empty() {
                                props.on_create.call(val);
                            }
                            input_value.set(String::new());
                            editing.set(false);
                        } else if e.key() == Key::Escape {
                            input_value.set(String::new());
                            editing.set(false);
                        }
                    },
                }
            }
        }
    } else {
        rsx! {
            div {
                class: "new-task-inline",
                onclick: move |_| editing.set(true),
                div { class: "new-task-plus", "+" }
                span { "New Task" }
            }
        }
    }
}
