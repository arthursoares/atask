use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct TagPillProps {
    label: String,
    #[props(default = "default".to_string())]
    variant: String,
}

#[component]
pub fn TagPill(props: TagPillProps) -> Element {
    let variant_class = match props.variant.as_str() {
        "today" => "tag tag-today",
        "deadline" => "tag tag-deadline",
        "agent" => "tag tag-agent",
        "success" => "tag tag-success",
        "accent" => "tag tag-accent",
        _ => "tag tag-default",
    };

    rsx! {
        span { class: variant_class, "{props.label}" }
    }
}
