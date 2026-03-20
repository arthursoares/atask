use dioxus::prelude::*;

#[derive(Props, Clone, PartialEq)]
pub struct DatePickerProps {
    value: Option<String>,
    on_change: EventHandler<Option<String>>,
}

#[component]
pub fn DatePicker(props: DatePickerProps) -> Element {
    let current = props.value.clone().unwrap_or_default();
    let has_value = props.value.is_some();
    let mut draft = use_signal(|| current.clone());

    // Only fire on_change when the user has typed a full valid date (YYYY-MM-DD = 10 chars)
    // or explicitly cleared the field
    rsx! {
        div { class: "date-picker-row",
            input {
                class: "date-picker-input",
                r#type: "date",
                value: "{draft}",
                oninput: move |e: Event<FormData>| {
                    draft.set(e.value());
                },
                onchange: {
                    let on_change = props.on_change.clone();
                    move |e: Event<FormData>| {
                        let val = e.value();
                        if val.is_empty() {
                            on_change.call(None);
                        } else if val.len() == 10 {
                            // Only fire when we have a complete YYYY-MM-DD
                            on_change.call(Some(val));
                        }
                    }
                },
            }
            if has_value {
                span {
                    class: "date-picker-clear",
                    onclick: {
                        let on_change = props.on_change.clone();
                        move |_| {
                            draft.set(String::new());
                            on_change.call(None);
                        }
                    },
                    "Clear"
                }
            }
        }
    }
}
