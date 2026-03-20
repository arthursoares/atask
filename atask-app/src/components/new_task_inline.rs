use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct NewTaskInlineProps {
    on_create: EventHandler<String>,
}

#[component]
pub fn NewTaskInline(props: NewTaskInlineProps) -> Element {
    let mut editing = use_signal(|| false);
    let mut input_value = use_signal(|| String::new());

    rsx! {
        if *editing.read() {
            div { class: "new-task-inline",
                input {
                    class: "input",
                    r#type: "text",
                    placeholder: "Task title",
                    autofocus: true,
                    value: "{input_value.read()}",
                    oninput: move |evt: Event<FormData>| {
                        input_value.set(evt.value().to_string());
                    },
                    onkeydown: move |evt: Event<KeyboardData>| {
                        if evt.key() == Key::Enter {
                            let title = input_value.read().trim().to_string();
                            if !title.is_empty() {
                                props.on_create.call(title);
                            }
                            editing.set(false);
                            input_value.set(String::new());
                        } else if evt.key() == Key::Escape {
                            editing.set(false);
                            input_value.set(String::new());
                        }
                    },
                    onmounted: move |evt: Event<MountedData>| {
                        let _ = evt.data().set_focus(true);
                    },
                }
            }
        } else {
            div {
                class: "new-task-inline",
                onclick: move |_| editing.set(true),

                span { class: "new-task-plus", "+" }
                span { "New Task" }
            }
        }
    }
}
