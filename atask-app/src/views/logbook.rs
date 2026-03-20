use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::checkbox::Checkbox;
use crate::components::section_header::SectionHeader;
use crate::components::task_meta::TaskMeta;

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
pub fn LogbookView() -> Element {
    let groups = use_signal(|| {
        vec![
            DateGroup {
                label: "Today".into(),
                tasks: vec![
                    Task {
                        id: "log-1".into(),
                        title: "Draft API specification for v0".into(),
                        status: 1,
                        schedule: 1,
                        completed_at: Some("2026-03-20T10:00:00Z".into()),
                        index: 0,
                        ..default_task()
                    },
                    Task {
                        id: "log-2".into(),
                        title: "Set up CI pipeline with GitHub Actions".into(),
                        status: 1,
                        schedule: 1,
                        completed_at: Some("2026-03-20T14:30:00Z".into()),
                        index: 1,
                        ..default_task()
                    },
                ],
            },
            DateGroup {
                label: "Yesterday".into(),
                tasks: vec![
                    Task {
                        id: "log-3".into(),
                        title: "Research SQLite WAL mode performance".into(),
                        status: 1,
                        schedule: 1,
                        completed_at: Some("2026-03-19T16:00:00Z".into()),
                        index: 2,
                        ..default_task()
                    },
                    Task {
                        id: "log-4".into(),
                        title: "Evaluate PostgreSQL migration".into(),
                        status: 2,
                        schedule: 1,
                        completed_at: Some("2026-03-19T17:00:00Z".into()),
                        index: 3,
                        ..default_task()
                    },
                ],
            },
        ]
    });

    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let all_groups: Vec<DateGroup> = groups.read().iter().map(|g| DateGroup {
        label: g.label.clone(),
        tasks: g.tasks.clone(),
    }).collect();

    let all_empty = all_groups.iter().all(|g| g.tasks.is_empty());

    if all_empty {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { "Nothing completed yet. Get started!" }
                }
            }
        };
    }

    rsx! {
        div { class: "task-list",
            for group in all_groups {
                SectionHeader {
                    title: group.label.clone(),
                    count: group.tasks.len(),
                    collapsed: false,
                    on_toggle: move |_| {},
                }
                for task in group.tasks {
                    {
                        let task_id = task.id.clone();
                        let is_selected = task.id == selected_id;
                        let is_cancelled = task.is_cancelled();

                        let item_class = if is_selected {
                            "task-item selected"
                        } else {
                            "task-item"
                        };

                        let title_class = if is_cancelled {
                            "task-title completed tertiary"
                        } else {
                            "task-title completed"
                        };

                        rsx! {
                            div {
                                class: item_class,
                                onclick: move |_| {
                                    selected_task_id.set(Some(task_id.clone()));
                                },
                                if is_cancelled {
                                    div { class: "checkbox cancelled",
                                        "\u{2715}"
                                    }
                                } else {
                                    Checkbox {
                                        checked: true,
                                        today: false,
                                        on_toggle: move |_| {},
                                    }
                                }
                                span { class: title_class, "{task.title}" }
                                TaskMeta {
                                    task: task.clone(),
                                    show_project: true,
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
