use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct ActivityEntryProps {
    author: String,
    #[props(default = false)]
    is_agent: bool,
    timestamp: String,
    content: String,
}

#[component]
pub fn ActivityEntry(props: ActivityEntryProps) -> Element {
    let avatar_class = if props.is_agent {
        "activity-avatar agent"
    } else {
        "activity-avatar human"
    };

    let author_class = if props.is_agent {
        "activity-author agent"
    } else {
        "activity-author"
    };

    let initial = props
        .author
        .chars()
        .next()
        .unwrap_or('?')
        .to_uppercase()
        .to_string();

    let avatar_content = if props.is_agent {
        "\u{2726}".to_string()
    } else {
        initial
    };

    rsx! {
        div { class: "activity-entry",
            div { class: avatar_class, "{avatar_content}" }
            div { class: "activity-body",
                div { class: "activity-header",
                    span { class: author_class, "{props.author}" }
                    span { class: "activity-time", "{props.timestamp}" }
                }
                div { class: "activity-content", "{props.content}" }
            }
        }
    }
}
