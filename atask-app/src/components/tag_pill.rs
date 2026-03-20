use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct TagPillProps {
    label: String,
    variant: String,
}

#[component]
pub fn TagPill(props: TagPillProps) -> Element {
    let variant_class = match props.variant.as_str() {
        "today" => "tag tag-today",
        "deadline" => "tag tag-deadline",
        "overdue" => "tag tag-overdue",
        "agent" => "tag tag-agent",
        "success" => "tag tag-success",
        _ => "tag tag-default",
    };

    rsx! {
        span { class: "{variant_class}", "{props.label}" }
    }
}
