use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::section_header::SectionHeader;
use crate::components::task_item::TaskItem;

fn default_task() -> Task {
    Task {
        id: String::new(),
        title: String::new(),
        notes: String::new(),
        status: 0,
        schedule: 0,
        start_date: None,
        deadline: None,
        completed_at: None,
        created_at: String::new(),
        updated_at: String::new(),
        index: 0,
        today_index: None,
        project_id: None,
        section_id: None,
        area_id: None,
        location_id: None,
        recurrence_rule: None,
        tags: None,
        deleted: false,
        deleted_at: None,
    }
}

struct DateGroup {
    label: String,
    tasks: Vec<Task>,
}

#[component]
pub fn UpcomingView() -> Element {
    let mut tasks = use_signal(|| {
        vec![
            DateGroup {
                label: "Tomorrow \u{2014} Fri, Mar 21".into(),
                tasks: vec![
                    Task {
                        id: "up-1".into(),
                        title: "Review pull request for event sourcing refactor".into(),
                        schedule: 1,
                        start_date: Some("2026-03-21".into()),
                        index: 0,
                        ..default_task()
                    },
                ],
            },
            DateGroup {
                label: "Saturday, Mar 22".into(),
                tasks: vec![
                    Task {
                        id: "up-2".into(),
                        title: "Prepare demo environment for client meeting".into(),
                        schedule: 1,
                        start_date: Some("2026-03-22".into()),
                        index: 1,
                        ..default_task()
                    },
                ],
            },
            DateGroup {
                label: "Next Week \u{2014} Mon, Mar 24".into(),
                tasks: vec![
                    Task {
                        id: "up-3".into(),
                        title: "Write integration tests for SSE streaming".into(),
                        schedule: 1,
                        start_date: Some("2026-03-24".into()),
                        index: 2,
                        ..default_task()
                    },
                    Task {
                        id: "up-4".into(),
                        title: "Draft RFC for agent authentication flow".into(),
                        schedule: 1,
                        start_date: Some("2026-03-24".into()),
                        index: 3,
                        ..default_task()
                    },
                ],
            },
        ]
    });

    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let groups: Vec<DateGroup> = tasks.read().iter().map(|g| DateGroup {
        label: g.label.clone(),
        tasks: g.tasks.clone(),
    }).collect();

    let all_empty = groups.iter().all(|g| g.tasks.is_empty());

    if all_empty {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { "Nothing scheduled ahead." }
                }
            }
        };
    }

    rsx! {
        div { class: "task-list",
            for group in groups {
                SectionHeader {
                    title: group.label.clone(),
                    count: group.tasks.len(),
                    collapsed: false,
                    on_toggle: move |_| {},
                }
                for task in group.tasks {
                    {
                        let task_id = task.id.clone();
                        let task_id_complete = task.id.clone();
                        let is_selected = task.id == selected_id;
                        rsx! {
                            TaskItem {
                                key: "{task_id}",
                                task: task,
                                selected: is_selected,
                                today_view: false,
                                on_select: move |id: String| {
                                    selected_task_id.set(Some(id));
                                },
                                on_complete: move |_id: String| {
                                    let mut all_groups = tasks.write();
                                    for g in all_groups.iter_mut() {
                                        if let Some(t) = g.tasks.iter_mut().find(|t| t.id == task_id_complete) {
                                            t.status = if t.status == 0 { 1 } else { 0 };
                                            break;
                                        }
                                    }
                                },
                            }
                        }
                    }
                }
            }
        }
    }
}
