use dioxus::prelude::*;

use crate::state::navigation::ActiveView;
use crate::state::projects::ProjectState;

/// Signal that, when set to true, shows the new-section input in the project view.
#[derive(Clone, Copy)]
pub struct AddSectionTrigger(pub Signal<bool>);

#[component]
pub fn Toolbar() -> Element {
    let active_view: Signal<ActiveView> = use_context();
    let project_state: Signal<ProjectState> = use_context();
    let mut add_section_trigger: AddSectionTrigger = use_context();

    let is_project = matches!(&*active_view.read(), ActiveView::Project(_));

    let (icon, title, subtitle) = match &*active_view.read() {
        ActiveView::Today => {
            let now = chrono::Local::now();
            let date_str = now.format("%A, %b %-d").to_string();
            let icon = rsx! {
                svg {
                    class: "toolbar-icon",
                    view_box: "0 0 16 16",
                    fill: "#F5A623",
                    polygon { points: "8 1 10.2 5.4 15 6.2 11.5 9.5 12.4 14.3 8 12 3.6 14.3 4.5 9.5 1 6.2 5.8 5.4" }
                }
            };
            (icon, "Today".to_string(), Some(date_str))
        }
        ActiveView::Inbox => {
            let icon = rsx! {
                svg {
                    class: "toolbar-icon",
                    view_box: "0 0 16 16",
                    fill: "none",
                    stroke: "currentColor",
                    stroke_width: "1.4",
                    path { d: "M2 10l2.5-2.5h7L14 10v3.5a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V10z" }
                    path { d: "M2 10h3.5a1 1 0 0 1 1 1v0a1 1 0 0 0 1 1h1a1 1 0 0 0 1-1v0a1 1 0 0 1 1-1H14" }
                    line { x1: "5.5", y1: "3", x2: "10.5", y2: "3" }
                    line { x1: "4", y1: "5.5", x2: "12", y2: "5.5" }
                }
            };
            (icon, "Inbox".to_string(), None)
        }
        ActiveView::Upcoming => {
            let icon = rsx! {
                svg {
                    class: "toolbar-icon",
                    view_box: "0 0 16 16",
                    fill: "none",
                    stroke: "currentColor",
                    stroke_width: "1.4",
                    rect { x: "2", y: "3", width: "12", height: "11", rx: "1.5" }
                    line { x1: "5", y1: "1.5", x2: "5", y2: "4.5" }
                    line { x1: "11", y1: "1.5", x2: "11", y2: "4.5" }
                    line { x1: "2", y1: "7", x2: "14", y2: "7" }
                }
            };
            (icon, "Upcoming".to_string(), None)
        }
        ActiveView::Someday => {
            let icon = rsx! {
                svg {
                    class: "toolbar-icon",
                    view_box: "0 0 16 16",
                    fill: "none",
                    stroke: "currentColor",
                    stroke_width: "1.4",
                    circle { cx: "8", cy: "8", r: "6" }
                    polyline { points: "8 4 8 8 11 10" }
                }
            };
            (icon, "Someday".to_string(), None)
        }
        ActiveView::Logbook => {
            let icon = rsx! {
                svg {
                    class: "toolbar-icon",
                    view_box: "0 0 16 16",
                    fill: "none",
                    stroke: "currentColor",
                    stroke_width: "1.4",
                    rect { x: "3", y: "2", width: "10", height: "12", rx: "1.5" }
                    path { d: "M3 5h10" }
                    path { d: "M6 2v3" }
                    path { d: "M10 2v3" }
                    polyline { points: "5.5 9 7 10.5 10.5 7.5" }
                }
            };
            (icon, "Logbook".to_string(), None)
        }
        ActiveView::Project(id) => {
            let projects = project_state.read().projects.read().clone();
            let project_name = projects
                .iter()
                .find(|p| p.id == *id)
                .map(|p| p.title.clone())
                .unwrap_or_else(|| "Project".to_string());
            let icon = rsx! {
                svg {
                    class: "toolbar-icon",
                    view_box: "0 0 16 16",
                    circle { cx: "8", cy: "8", r: "5", fill: "var(--accent)" }
                }
            };
            (icon, project_name, None)
        }
    };

    // For project view, compute progress from project_tasks
    let (completed, total) = if let ActiveView::Project(ref id) = *active_view.read() {
        let tasks = project_state
            .read()
            .project_tasks
            .read()
            .get(id)
            .cloned()
            .unwrap_or_default();
        let total = tasks.len() as u32;
        let completed = tasks.iter().filter(|t| t.is_completed()).count() as u32;
        (completed, total)
    } else {
        (0, 0)
    };

    let progress_pct = if total > 0 {
        format!("{}%", (completed as f64 / total as f64 * 100.0) as u32)
    } else {
        "0%".to_string()
    };

    rsx! {
        div { class: "toolbar-wrapper",
            div { class: "app-toolbar",
                div { class: "app-toolbar-left",
                    {icon}
                    span { class: "app-view-title", "{title}" }
                    if let Some(sub) = subtitle {
                        span { class: "toolbar-subtitle", "{sub}" }
                    }
                    if is_project {
                        span { class: "toolbar-progress-label", "{completed} / {total}" }
                    }
                }
                div { class: "app-toolbar-right",
                    if is_project {
                        button {
                            class: "btn btn-ghost btn-sm",
                            onclick: move |_| {
                                add_section_trigger.0.set(true);
                            },
                            "+ Add Section"
                        }
                    }
                    button { class: "toolbar-btn",
                        svg {
                            view_box: "0 0 16 16",
                            fill: "none",
                            stroke: "currentColor",
                            stroke_width: "1.4",
                            circle { cx: "7", cy: "7", r: "4.5" }
                            line { x1: "10.2", y1: "10.2", x2: "14", y2: "14" }
                        }
                    }
                    button { class: "toolbar-btn",
                        svg {
                            view_box: "0 0 16 16",
                            fill: "none",
                            stroke: "currentColor",
                            stroke_width: "2",
                            line { x1: "8", y1: "3", x2: "8", y2: "13" }
                            line { x1: "3", y1: "8", x2: "13", y2: "8" }
                        }
                    }
                }
            }
            if is_project {
                div { class: "toolbar-progress-bar",
                    div { class: "toolbar-progress-fill", width: progress_pct }
                }
            }
        }
    }
}
