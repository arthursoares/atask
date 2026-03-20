use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct SectionHeaderProps {
    title: String,
    count: usize,
    #[props(default = false)]
    collapsed: bool,
    on_toggle: EventHandler<()>,
}

#[component]
pub fn SectionHeader(props: SectionHeaderProps) -> Element {
    let chevron_class = if props.collapsed {
        "section-header-chevron collapsed"
    } else {
        "section-header-chevron"
    };

    rsx! {
        div {
            class: "section-header",
            onclick: move |_| props.on_toggle.call(()),
            svg {
                class: chevron_class,
                view_box: "0 0 16 16",
                fill: "none",
                stroke: "currentColor",
                stroke_width: "1.8",
                polyline { points: "5 3 11 8 5 13" }
            }
            span { class: "section-header-title", "{props.title}" }
            span { class: "section-header-count", "{props.count}" }
            div { class: "section-header-line" }
        }
    }
}
