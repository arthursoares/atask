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

    rsx! {
        div { class: "date-picker-row",
            input {
                class: "date-picker-input",
                r#type: "date",
                value: "{current}",
                onchange: {
                    let on_change = props.on_change.clone();
                    move |e: Event<FormData>| {
                        let val = e.value();
                        if val.is_empty() {
                            on_change.call(None);
                        } else {
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
                        move |_| on_change.call(None)
                    },
                    "Clear"
                }
            }
        }
    }
}
