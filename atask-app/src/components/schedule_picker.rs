use dioxus::prelude::*;

#[component]
pub fn SchedulePicker(current: i64, on_change: EventHandler<String>) -> Element {
    let options: [(i64, &str, &str); 3] = [
        (0, "inbox", "Inbox"),
        (1, "anytime", "Today"),
        (2, "someday", "Someday"),
    ];

    rsx! {
        div { class: "schedule-picker",
            for (value, key, label) in options {
                {
                    let cls = if current == value {
                        "schedule-option active"
                    } else {
                        "schedule-option"
                    };
                    let key = key.to_string();
                    rsx! {
                        button {
                            class: "{cls}",
                            onclick: {
                                let key = key.clone();
                                move |_| on_change.call(key.clone())
                            },
                            "{label}"
                        }
                    }
                }
            }
        }
    }
}
