use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::state::navigation::ActiveView;
use crate::state::tasks::TaskState;
use crate::state::projects::ProjectState;

#[component]
pub fn Sidebar() -> Element {
    let active_view: Signal<ActiveView> = use_context();
    let task_state: Signal<TaskState> = use_context();
    let mut project_state: Signal<ProjectState> = use_context();
    let api: Signal<ApiClient> = use_context();

    let mut adding_project: Signal<bool> = use_signal(|| false);
    let mut new_project_title: Signal<String> = use_signal(|| String::new());
    let mut adding_area: Signal<bool> = use_signal(|| false);
    let mut new_area_title: Signal<String> = use_signal(|| String::new());

    // Signal reads happen inside rsx! for reactivity — see below.

    rsx! {
        div { class: "sidebar",
            // Traffic lights / drag region
            div { class: "sidebar-drag-region" }

            // Read counts inside rsx! so Dioxus tracks signal dependencies
            {
                let inbox_count = task_state.read().inbox.read().len() as u32;
                let today_count = task_state.read().today.read().len() as u32;
                let upcoming_count = task_state.read().upcoming.read().len() as u32;
                let someday_count = task_state.read().someday.read().len() as u32;

                rsx! {
                    // Navigation items
                    div { class: "sidebar-group",
                        NavItem {
                            label: "Inbox",
                            view: ActiveView::Inbox,
                            active_view,
                            badge: inbox_count,
                            icon: NavIcon::Inbox,
                        }
                        NavItem {
                            label: "Today",
                            view: ActiveView::Today,
                            active_view,
                            badge: today_count,
                            icon: NavIcon::Today,
                        }
                        NavItem {
                            label: "Upcoming",
                            view: ActiveView::Upcoming,
                            active_view,
                            badge: upcoming_count,
                            icon: NavIcon::Upcoming,
                        }
                        NavItem {
                            label: "Someday",
                            view: ActiveView::Someday,
                            active_view,
                            badge: someday_count,
                            icon: NavIcon::Someday,
                        }
                        NavItem {
                            label: "Logbook",
                            view: ActiveView::Logbook,
                            active_view,
                            badge: 0,
                            icon: NavIcon::Logbook,
                        }
                    }
                }
            }

            div { class: "sidebar-separator" }

            // Projects — read signal inside rsx!
            {
                let projects = project_state.read().projects.read().clone();
                let active_projects: Vec<_> = projects.into_iter().filter(|p| p.status == 0).collect();
                rsx! {
                    div { class: "sidebar-group",
                        div { class: "sidebar-group-label", "Projects" }
                        for project in active_projects {
                            {
                                let pid = project.id.clone();
                                let pname = project.title.clone();
                                rsx! {
                                    ProjectItem {
                                        key: "{pid}",
                                        id: pid,
                                        name: pname,
                                        active_view,
                                    }
                                }
                            }
                        }
                        if *adding_project.read() {
                            input {
                                class: "sidebar-inline-input",
                                placeholder: "Project name",
                                value: "{new_project_title}",
                                autofocus: true,
                                oninput: move |e: Event<FormData>| new_project_title.set(e.value()),
                                onkeydown: move |e: Event<KeyboardData>| {
                                    if e.key() == Key::Enter {
                                        let title = new_project_title.read().clone();
                                        if !title.is_empty() {
                                            let api_clone = api.read().clone();
                                            spawn(async move {
                                                if let Ok(p) = api_clone.create_project(&title).await {
                                                    let mut projects = project_state.read().projects.read().clone();
                                                    projects.push(p);
                                                    project_state.write().projects.set(projects);
                                                }
                                            });
                                        }
                                        adding_project.set(false);
                                        new_project_title.set(String::new());
                                    } else if e.key() == Key::Escape {
                                        adding_project.set(false);
                                        new_project_title.set(String::new());
                                    }
                                },
                            }
                        } else {
                            div {
                                class: "sidebar-add-btn",
                                onclick: move |_| adding_project.set(true),
                                "+ Project"
                            }
                        }
                    }
                }
            }

            // Areas — read signal inside rsx!
            {
                let areas = project_state.read().areas.read().clone();
                let active_areas: Vec<_> = areas.into_iter().filter(|a| !a.archived).collect();
                if !active_areas.is_empty() {
                    rsx! {
                        div { class: "sidebar-separator" }

                        div { class: "sidebar-group",
                            div { class: "sidebar-group-label", "Areas" }
                            for area in active_areas {
                                {
                                    let aname = area.title.clone();
                                    rsx! {
                                        AreaItem {
                                            name: aname,
                                            active: false,
                                        }
                                    }
                                }
                            }
                            if *adding_area.read() {
                                input {
                                    class: "sidebar-inline-input",
                                    placeholder: "Area name",
                                    value: "{new_area_title}",
                                    autofocus: true,
                                    oninput: move |e: Event<FormData>| new_area_title.set(e.value()),
                                    onkeydown: move |e: Event<KeyboardData>| {
                                        if e.key() == Key::Enter {
                                            let title = new_area_title.read().clone();
                                            if !title.is_empty() {
                                                let api_clone = api.read().clone();
                                                spawn(async move {
                                                    if let Ok(a) = api_clone.create_area(&title).await {
                                                        let mut areas = project_state.read().areas.read().clone();
                                                        areas.push(a);
                                                        project_state.write().areas.set(areas);
                                                    }
                                                });
                                            }
                                            adding_area.set(false);
                                            new_area_title.set(String::new());
                                        } else if e.key() == Key::Escape {
                                            adding_area.set(false);
                                            new_area_title.set(String::new());
                                        }
                                    },
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
                } else {
                    rsx! {
                        div { class: "sidebar-separator" }

                        div { class: "sidebar-group",
                            div { class: "sidebar-group-label", "Areas" }
                            if *adding_area.read() {
                                input {
                                    class: "sidebar-inline-input",
                                    placeholder: "Area name",
                                    value: "{new_area_title}",
                                    autofocus: true,
                                    oninput: move |e: Event<FormData>| new_area_title.set(e.value()),
                                    onkeydown: move |e: Event<KeyboardData>| {
                                        if e.key() == Key::Enter {
                                            let title = new_area_title.read().clone();
                                            if !title.is_empty() {
                                                let api_clone = api.read().clone();
                                                spawn(async move {
                                                    if let Ok(a) = api_clone.create_area(&title).await {
                                                        let mut areas = project_state.read().areas.read().clone();
                                                        areas.push(a);
                                                        project_state.write().areas.set(areas);
                                                    }
                                                });
                                            }
                                            adding_area.set(false);
                                            new_area_title.set(String::new());
                                        } else if e.key() == Key::Escape {
                                            adding_area.set(false);
                                            new_area_title.set(String::new());
                                        }
                                    },
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
}

#[derive(Debug, Clone, PartialEq)]
enum NavIcon {
    Inbox,
    Today,
    Upcoming,
    Someday,
    Logbook,
}

#[derive(Clone, PartialEq, Props)]
struct NavItemProps {
    label: &'static str,
    view: ActiveView,
    active_view: Signal<ActiveView>,
    badge: u32,
    icon: NavIcon,
}

#[component]
fn NavItem(props: NavItemProps) -> Element {
    let mut active_view = props.active_view;
    let is_active = *active_view.read() == props.view;
    let class = if is_active {
        "sidebar-item active"
    } else {
        "sidebar-item"
    };
    let view = props.view.clone();

    rsx! {
        div {
            class,
            onclick: move |_| active_view.set(view.clone()),
            span { class: "sidebar-icon",
                match props.icon {
                    NavIcon::Inbox => rsx! {
                        svg {
                            view_box: "0 0 16 16",
                            polyline { points: "1 9 4.5 9 6 11 10 11 11.5 9 15 9" }
                            path { d: "M3.04 4.28 1 9v4a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V9l-2.04-4.72A1 1 0 0 0 12.04 3.5H3.96a1 1 0 0 0-.92.78z" }
                        }
                    },
                    NavIcon::Today => rsx! {
                        svg {
                            view_box: "0 0 16 16",
                            style: "fill: var(--today-star); stroke: var(--today-star);",
                            polygon { points: "8 1.5 9.9 5.8 14.5 6.2 11 9.4 12 14 8 11.5 4 14 5 9.4 1.5 6.2 6.1 5.8" }
                        }
                    },
                    NavIcon::Upcoming => rsx! {
                        svg {
                            view_box: "0 0 16 16",
                            rect { x: "2", y: "3", width: "12", height: "11", rx: "1" }
                            line { x1: "2", y1: "6.5", x2: "14", y2: "6.5" }
                            line { x1: "5", y1: "1.5", x2: "5", y2: "4" }
                            line { x1: "11", y1: "1.5", x2: "11", y2: "4" }
                        }
                    },
                    NavIcon::Someday => rsx! {
                        svg {
                            view_box: "0 0 16 16",
                            style: "stroke: var(--someday-tint);",
                            circle { cx: "8", cy: "8", r: "6" }
                            polyline { points: "8 4.5 8 8 10.5 10" }
                        }
                    },
                    NavIcon::Logbook => rsx! {
                        svg {
                            view_box: "0 0 16 16",
                            path { d: "M3.5 2h9a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z" }
                            line { x1: "5.5", y1: "5.5", x2: "10.5", y2: "5.5" }
                            line { x1: "5.5", y1: "8", x2: "10.5", y2: "8" }
                            line { x1: "5.5", y1: "10.5", x2: "8.5", y2: "10.5" }
                        }
                    },
                }
            }
            span { "{props.label}" }
            if props.badge > 0 {
                span { class: "sidebar-badge", "{props.badge}" }
            }
        }
    }
}

#[derive(Clone, PartialEq, Props)]
struct ProjectItemProps {
    id: String,
    name: String,
    active_view: Signal<ActiveView>,
}

#[component]
fn ProjectItem(props: ProjectItemProps) -> Element {
    let mut active_view = props.active_view;
    let is_active = *active_view.read() == ActiveView::Project(props.id.clone());
    let class = if is_active {
        "sidebar-item active"
    } else {
        "sidebar-item"
    };
    let id = props.id.clone();

    rsx! {
        div {
            class,
            onclick: move |_| active_view.set(ActiveView::Project(id.clone())),
            span { class: "sidebar-project-dot" }
            span { "{props.name}" }
        }
    }
}

#[derive(Clone, PartialEq, Props)]
struct AreaItemProps {
    name: String,
    active: bool,
}

#[component]
fn AreaItem(props: AreaItemProps) -> Element {
    rsx! {
        div { class: "sidebar-item",
            span { class: "sidebar-icon",
                svg {
                    view_box: "0 0 16 16",
                    path { d: "M1.5 3.5a1 1 0 0 1 1-1h4l1.5 1.5h5.5a1 1 0 0 1 1 1v7a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1z" }
                }
            }
            span { "{props.name}" }
        }
    }
}
