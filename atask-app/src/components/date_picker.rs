use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct DatePickerProps {
    value: Option<String>,
    on_change: EventHandler<Option<String>>,
    label: String,
}

#[component]
pub fn DatePicker(props: DatePickerProps) -> Element {
    let on_change = props.on_change.clone();
    let on_clear = props.on_change.clone();

    rsx! {
        div { class: "detail-field",
            div { class: "detail-field-label", "{props.label}" }
            div { class: "date-picker-row",
                input {
                    class: "input date-picker-input",
                    r#type: "date",
                    value: props.value.clone().unwrap_or_default(),
                    onchange: move |evt: Event<FormData>| {
                        let val = evt.value();
                        if val.is_empty() {
                            on_change.call(None);
                        } else {
                            on_change.call(Some(val));
                        }
                    },
                }
                if props.value.is_some() {
                    span {
                        class: "date-picker-clear",
                        onclick: move |_| on_clear.call(None),
                        "Clear"
                    }
                }
            }
        }
    }
}
