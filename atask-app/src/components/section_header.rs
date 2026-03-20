use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct SectionHeaderProps {
    title: String,
    count: usize,
}

#[component]
pub fn SectionHeader(props: SectionHeaderProps) -> Element {
    rsx! {
        div { class: "section-header",
            span { class: "section-title", "{props.title}" }
            if props.count > 0 {
                span { class: "section-count", "{props.count}" }
            }
            div { class: "section-line" }
        }
    }
}
