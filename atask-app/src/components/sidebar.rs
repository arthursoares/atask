use dioxus::prelude::*;

use crate::state::navigation::ActiveView;

#[component]
pub fn Sidebar() -> Element {
    let active_view: Signal<ActiveView> = use_context();

    rsx! {
        div { class: "sidebar",
            // Traffic lights / drag region
            div { class: "sidebar-drag-region" }

            // Navigation items
            div { class: "sidebar-group",
                NavItem {
                    label: "Inbox",
                    view: ActiveView::Inbox,
                    active_view,
                    badge: 3,
                    icon: NavIcon::Inbox,
                }
                NavItem {
                    label: "Today",
                    view: ActiveView::Today,
                    active_view,
                    badge: 5,
                    icon: NavIcon::Today,
                }
                NavItem {
                    label: "Upcoming",
                    view: ActiveView::Upcoming,
                    active_view,
                    badge: 0,
                    icon: NavIcon::Upcoming,
                }
                NavItem {
                    label: "Someday",
                    view: ActiveView::Someday,
                    active_view,
                    badge: 0,
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

            div { class: "sidebar-separator" }

            // Projects
            div { class: "sidebar-group",
                div { class: "sidebar-group-label", "Projects" }
                ProjectItem {
                    id: "proj-atask-v0",
                    name: "atask v0",
                    color: "#4670a0",
                    badge: 12,
                    active_view,
                }
                ProjectItem {
                    id: "proj-homelab",
                    name: "Homelab",
                    color: "#4a8860",
                    badge: 4,
                    active_view,
                }
                ProjectItem {
                    id: "proj-roon-ext",
                    name: "Roon Ext",
                    color: "#a07846",
                    badge: 7,
                    active_view,
                }
            }

            div { class: "sidebar-separator" }

            // Areas
            div { class: "sidebar-group",
                div { class: "sidebar-group-label", "Areas" }
                AreaItem {
                    name: "Work",
                    active: false,
                }
                AreaItem {
                    name: "Home",
                    active: false,
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
    id: &'static str,
    name: &'static str,
    color: &'static str,
    badge: u32,
    active_view: Signal<ActiveView>,
}

#[component]
fn ProjectItem(props: ProjectItemProps) -> Element {
    let mut active_view = props.active_view;
    let is_active = *active_view.read() == ActiveView::Project(props.id.to_string());
    let class = if is_active {
        "sidebar-item active"
    } else {
        "sidebar-item"
    };
    let id = props.id.to_string();

    rsx! {
        div {
            class,
            onclick: move |_| active_view.set(ActiveView::Project(id.clone())),
            span {
                class: "sidebar-project-dot",
                style: "background: {props.color};",
            }
            span { "{props.name}" }
            if props.badge > 0 {
                span { class: "sidebar-badge", "{props.badge}" }
            }
        }
    }
}

#[derive(Clone, PartialEq, Props)]
struct AreaItemProps {
    name: &'static str,
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
