use dioxus::prelude::*;
use crate::state::app::*;

#[component]
pub fn Toolbar() -> Element {
    let active_view: ViewSignal = use_context();
    let projects: ProjectList = use_context();

    rsx! {
        div { class: "app-toolbar",
            div { class: "app-toolbar-left",
                {
                    match &*active_view.0.read() {
                        ActiveView::Today => {
                            let now = chrono::Local::now();
                            let date_str = now.format("%A, %b %-d").to_string();
                            rsx! {
                                div { class: "app-view-title",
                                    svg {
                                        view_box: "0 0 24 24",
                                        xmlns: "http://www.w3.org/2000/svg",
                                        width: "20",
                                        height: "20",
                                        polygon {
                                            points: "12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2",
                                            fill: "var(--today-star)",
                                            stroke: "var(--today-star)",
                                            stroke_width: "1.8",
                                        }
                                    }
                                    span { "Today" }
                                }
                                span { class: "toolbar-subtitle", "{date_str}" }
                            }
                        }
                        ActiveView::Inbox => rsx! {
                            div { class: "app-view-title",
                                svg {
                                    view_box: "0 0 24 24",
                                    xmlns: "http://www.w3.org/2000/svg",
                                    width: "20",
                                    height: "20",
                                    stroke: "currentColor",
                                    stroke_width: "1.8",
                                    fill: "none",
                                    path { d: "M22 12H16L14 15H10L8 12H2" }
                                    path { d: "M5.45 5.11L2 12V18C2 18.5304 2.21071 19.0391 2.58579 19.4142C2.96086 19.7893 3.46957 20 4 20H20C20.5304 20 21.0391 19.7893 21.4142 19.4142C21.7893 19.0391 22 18.5304 22 18V12L18.55 5.11C18.3844 4.77679 18.1292 4.49637 17.813 4.30028C17.4967 4.10419 17.1321 4.0002 16.76 4H7.24C6.86792 4.0002 6.50326 4.10419 6.18704 4.30028C5.87083 4.49637 5.61558 4.77679 5.45 5.11Z" }
                                }
                                span { "Inbox" }
                            }
                        },
                        ActiveView::Upcoming => rsx! {
                            div { class: "app-view-title",
                                svg {
                                    view_box: "0 0 24 24",
                                    xmlns: "http://www.w3.org/2000/svg",
                                    width: "20",
                                    height: "20",
                                    stroke: "currentColor",
                                    stroke_width: "1.8",
                                    fill: "none",
                                    rect { x: "3", y: "4", width: "18", height: "18", rx: "2", ry: "2" }
                                    line { x1: "16", y1: "2", x2: "16", y2: "6" }
                                    line { x1: "8", y1: "2", x2: "8", y2: "6" }
                                    line { x1: "3", y1: "10", x2: "21", y2: "10" }
                                }
                                span { "Upcoming" }
                            }
                        },
                        ActiveView::Someday => rsx! {
                            div { class: "app-view-title",
                                svg {
                                    view_box: "0 0 24 24",
                                    xmlns: "http://www.w3.org/2000/svg",
                                    width: "20",
                                    height: "20",
                                    stroke: "var(--someday-tint)",
                                    stroke_width: "1.8",
                                    fill: "none",
                                    circle { cx: "12", cy: "12", r: "10" }
                                    path { d: "M12 6V12L16 14" }
                                }
                                span { "Someday" }
                            }
                        },
                        ActiveView::Logbook => rsx! {
                            div { class: "app-view-title",
                                svg {
                                    view_box: "0 0 24 24",
                                    xmlns: "http://www.w3.org/2000/svg",
                                    width: "20",
                                    height: "20",
                                    stroke: "currentColor",
                                    stroke_width: "1.8",
                                    fill: "none",
                                    path { d: "M21 8V21H3V8" }
                                    path { d: "M1 3H23V8H1Z" }
                                    path { d: "M10 12H14" }
                                }
                                span { "Logbook" }
                            }
                        },
                        ActiveView::Project(id) => {
                            let project_list = projects.0.read();
                            let project = project_list.iter().find(|p| &p.id == id);
                            let title = project.map(|p| p.title.clone()).unwrap_or_else(|| "Project".to_string());
                            let color = project.map(|p| {
                                if p.color.is_empty() { "var(--accent)".to_string() } else { p.color.clone() }
                            }).unwrap_or_else(|| "var(--accent)".to_string());
                            rsx! {
                                div { class: "app-view-title",
                                    span {
                                        class: "sidebar-project-dot",
                                        style: "background: {color};",
                                    }
                                    span { "{title}" }
                                }
                            }
                        }
                    }
                }
            }
            div { class: "app-toolbar-right",
                // Search button (placeholder)
                button { class: "app-toolbar-btn",
                    svg {
                        view_box: "0 0 24 24",
                        xmlns: "http://www.w3.org/2000/svg",
                        circle { cx: "11", cy: "11", r: "8" }
                        line { x1: "21", y1: "21", x2: "16.65", y2: "16.65" }
                    }
                }
                // New task button (placeholder)
                button { class: "app-toolbar-btn",
                    svg {
                        view_box: "0 0 24 24",
                        xmlns: "http://www.w3.org/2000/svg",
                        line { x1: "12", y1: "5", x2: "12", y2: "19" }
                        line { x1: "5", y1: "12", x2: "19", y2: "12" }
                    }
                }
            }
        }
    }
}
