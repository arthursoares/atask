use dioxus::prelude::*;
use crate::state::app::*;

#[component]
pub fn Sidebar() -> Element {
    let mut active_view: ViewSignal = use_context();
    let inbox: InboxTasks = use_context();
    let today: TodayTasks = use_context();
    let mut projects: ProjectList = use_context();
    let mut areas: AreaList = use_context();
    let api: ApiSignal = use_context();

    let mut adding_project = use_signal(|| false);
    let mut project_input = use_signal(|| String::new());
    let mut adding_area = use_signal(|| false);
    let mut area_input = use_signal(|| String::new());

    rsx! {
        div { class: "sidebar",
            // Drag region (space for native traffic lights)
            div { class: "sidebar-toolbar" }

            // Nav items
            div { class: "sidebar-group",
                // Inbox
                div {
                    class: if matches!(*active_view.0.read(), ActiveView::Inbox) { "sidebar-item active" } else { "sidebar-item" },
                    onclick: move |_| active_view.0.set(ActiveView::Inbox),
                    span { class: "sidebar-icon",
                        svg {
                            view_box: "0 0 24 24",
                            xmlns: "http://www.w3.org/2000/svg",
                            path { d: "M22 12H16L14 15H10L8 12H2" }
                            path { d: "M5.45 5.11L2 12V18C2 18.5304 2.21071 19.0391 2.58579 19.4142C2.96086 19.7893 3.46957 20 4 20H20C20.5304 20 21.0391 19.7893 21.4142 19.4142C21.7893 19.0391 22 18.5304 22 18V12L18.55 5.11C18.3844 4.77679 18.1292 4.49637 17.813 4.30028C17.4967 4.10419 17.1321 4.0002 16.76 4H7.24C6.86792 4.0002 6.50326 4.10419 6.18704 4.30028C5.87083 4.49637 5.61558 4.77679 5.45 5.11Z" }
                        }
                    }
                    span { "Inbox" }
                    {
                        let count = inbox.0.read().len();
                        if count > 0 {
                            rsx! { span { class: "sidebar-badge", "{count}" } }
                        } else {
                            rsx! {}
                        }
                    }
                }

                // Today
                div {
                    class: if matches!(*active_view.0.read(), ActiveView::Today) { "sidebar-item active" } else { "sidebar-item" },
                    onclick: move |_| active_view.0.set(ActiveView::Today),
                    span { class: "sidebar-icon",
                        svg {
                            view_box: "0 0 24 24",
                            xmlns: "http://www.w3.org/2000/svg",
                            polygon {
                                points: "12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2",
                                fill: "var(--today-star)",
                                stroke: "var(--today-star)",
                            }
                        }
                    }
                    span { "Today" }
                    {
                        let count = today.0.read().len();
                        if count > 0 {
                            rsx! { span { class: "sidebar-badge", "{count}" } }
                        } else {
                            rsx! {}
                        }
                    }
                }

                // Upcoming
                div {
                    class: if matches!(*active_view.0.read(), ActiveView::Upcoming) { "sidebar-item active" } else { "sidebar-item" },
                    onclick: move |_| active_view.0.set(ActiveView::Upcoming),
                    span { class: "sidebar-icon",
                        svg {
                            view_box: "0 0 24 24",
                            xmlns: "http://www.w3.org/2000/svg",
                            rect { x: "3", y: "4", width: "18", height: "18", rx: "2", ry: "2" }
                            line { x1: "16", y1: "2", x2: "16", y2: "6" }
                            line { x1: "8", y1: "2", x2: "8", y2: "6" }
                            line { x1: "3", y1: "10", x2: "21", y2: "10" }
                        }
                    }
                    span { "Upcoming" }
                }

                // Someday
                div {
                    class: if matches!(*active_view.0.read(), ActiveView::Someday) { "sidebar-item active" } else { "sidebar-item" },
                    onclick: move |_| active_view.0.set(ActiveView::Someday),
                    span { class: "sidebar-icon",
                        svg {
                            view_box: "0 0 24 24",
                            xmlns: "http://www.w3.org/2000/svg",
                            stroke: "var(--someday-tint)",
                            circle { cx: "12", cy: "12", r: "10" }
                            path { d: "M12 6V12L16 14" }
                        }
                    }
                    span { "Someday" }
                }

                // Logbook
                div {
                    class: if matches!(*active_view.0.read(), ActiveView::Logbook) { "sidebar-item active" } else { "sidebar-item" },
                    onclick: move |_| active_view.0.set(ActiveView::Logbook),
                    span { class: "sidebar-icon",
                        svg {
                            view_box: "0 0 24 24",
                            xmlns: "http://www.w3.org/2000/svg",
                            path { d: "M21 8V21H3V8" }
                            path { d: "M1 3H23V8H1Z" }
                            path { d: "M10 12H14" }
                        }
                    }
                    span { "Logbook" }
                }
            }

            div { class: "sidebar-separator" }

            // Areas with nested projects
            {
                let area_list = areas.0.read().clone();
                let project_list = projects.0.read().clone();

                // Projects grouped by area
                let orphan_projects: Vec<_> = project_list.iter()
                    .filter(|p| p.area_id.is_none() && p.completed_at.is_none())
                    .collect();

                rsx! {
                    // Each area with its projects
                    for area in area_list.iter() {
                        {
                            let area_id = area.id.clone();
                            let area_title = area.title.clone();
                            let area_projects: Vec<_> = project_list.iter()
                                .filter(|p| p.area_id.as_deref() == Some(&area_id) && p.completed_at.is_none())
                                .collect();

                            rsx! {
                                div { class: "sidebar-group",
                                    div { class: "sidebar-group-label", "{area_title}" }
                                    for project in area_projects.iter() {
                                        {
                                            let project_id = project.id.clone();
                                            let project_id_match = project.id.clone();
                                            let project_color = if project.color.is_empty() {
                                                "var(--accent)".to_string()
                                            } else {
                                                project.color.clone()
                                            };
                                            rsx! {
                                                div {
                                                    class: if matches!(&*active_view.0.read(), ActiveView::Project(id) if id == &project_id_match) { "sidebar-item active" } else { "sidebar-item" },
                                                    onclick: move |_| active_view.0.set(ActiveView::Project(project_id.clone())),
                                                    span {
                                                        class: "sidebar-project-dot",
                                                        style: "background: {project_color};",
                                                    }
                                                    span { "{project.title}" }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }

                    // Orphan projects (no area)
                    if !orphan_projects.is_empty() {
                        div { class: "sidebar-group",
                            for project in orphan_projects.iter() {
                                {
                                    let project_id = project.id.clone();
                                    let project_id_match = project.id.clone();
                                    let project_color = if project.color.is_empty() {
                                        "var(--accent)".to_string()
                                    } else {
                                        project.color.clone()
                                    };
                                    rsx! {
                                        div {
                                            class: if matches!(&*active_view.0.read(), ActiveView::Project(id) if id == &project_id_match) { "sidebar-item active" } else { "sidebar-item" },
                                            onclick: move |_| active_view.0.set(ActiveView::Project(project_id.clone())),
                                            span {
                                                class: "sidebar-project-dot",
                                                style: "background: {project_color};",
                                            }
                                            span { "{project.title}" }
                                        }
                                    }
                                }
                            }
                        }
                    }

                    // Add project / area buttons
                    div { class: "sidebar-group",
                        if *adding_project.read() {
                            div { class: "sidebar-item",
                                input {
                                    class: "sidebar-inline-input",
                                    r#type: "text",
                                    placeholder: "Project name",
                                    autofocus: true,
                                    value: "{project_input.read()}",
                                    oninput: move |evt: Event<FormData>| project_input.set(evt.value().to_string()),
                                    onkeydown: move |evt: Event<KeyboardData>| {
                                        if evt.key() == Key::Enter {
                                            let title = project_input.read().trim().to_string();
                                            if !title.is_empty() {
                                                let api_clone = api.0.read().clone();
                                                spawn(async move {
                                                    if let Ok(project) = api_clone.create_project(&title).await {
                                                        let mut list = projects.0.read().clone();
                                                        list.push(project);
                                                        projects.0.set(list);
                                                    }
                                                });
                                            }
                                            adding_project.set(false);
                                            project_input.set(String::new());
                                        } else if evt.key() == Key::Escape {
                                            adding_project.set(false);
                                            project_input.set(String::new());
                                        }
                                    },
                                }
                            }
                        } else {
                            div {
                                class: "sidebar-add-btn",
                                onclick: move |_| adding_project.set(true),
                                "+ Project"
                            }
                        }

                        if *adding_area.read() {
                            div { class: "sidebar-item",
                                input {
                                    class: "sidebar-inline-input",
                                    r#type: "text",
                                    placeholder: "Area name",
                                    autofocus: true,
                                    value: "{area_input.read()}",
                                    oninput: move |evt: Event<FormData>| area_input.set(evt.value().to_string()),
                                    onkeydown: move |evt: Event<KeyboardData>| {
                                        if evt.key() == Key::Enter {
                                            let title = area_input.read().trim().to_string();
                                            if !title.is_empty() {
                                                let api_clone = api.0.read().clone();
                                                spawn(async move {
                                                    if let Ok(area) = api_clone.create_area(&title).await {
                                                        let mut list = areas.0.read().clone();
                                                        list.push(area);
                                                        areas.0.set(list);
                                                    }
                                                });
                                            }
                                            adding_area.set(false);
                                            area_input.set(String::new());
                                        } else if evt.key() == Key::Escape {
                                            adding_area.set(false);
                                            area_input.set(String::new());
                                        }
                                    },
                                }
                            }
                        } else {
                            div {
                                class: "sidebar-add-btn",
                                onclick: move |_| adding_area.set(true),
                                "+ Area"
                            }
                        }
                    }
                }
            }
        }
    }
}
