use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::checkbox::Checkbox;
use crate::components::section_header::SectionHeader;
use crate::components::task_meta::TaskMeta;
use crate::state::tasks::TaskState;

/// Group logbook tasks by completed_at date.
struct DateGroup {
    label: String,
    tasks: Vec<Task>,
}

fn group_by_completion_date(tasks: &[Task]) -> Vec<DateGroup> {
    use std::collections::BTreeMap;
    let today = chrono::Local::now().format("%Y-%m-%d").to_string();
    let yesterday = (chrono::Local::now() - chrono::Duration::days(1))
        .format("%Y-%m-%d")
        .to_string();

    // Group by date portion of completed_at
    let mut map: BTreeMap<String, Vec<Task>> = BTreeMap::new();
    for task in tasks {
        let date_key = task
            .completed_at
            .as_ref()
            .map(|dt| dt.split('T').next().unwrap_or(dt).to_string())
            .unwrap_or_else(|| "Unknown".to_string());
        map.entry(date_key).or_default().push(task.clone());
    }

    // Convert to groups with friendly labels, reverse order (newest first)
    map.into_iter()
        .rev()
        .map(|(date, tasks)| {
            let label = if date == today {
                "Today".to_string()
            } else if date == yesterday {
                "Yesterday".to_string()
            } else {
                date
            };
            DateGroup { label, tasks }
        })
        .collect()
}

#[component]
pub fn LogbookView() -> Element {
    let task_state: Signal<TaskState> = use_context();
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let tasks: Vec<Task> = task_state.read().logbook.read().clone();
    let is_loading = *task_state.read().loading.read();

    if is_loading && tasks.is_empty() {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { class: "empty-state-text", "Loading..." }
                }
            }
        };
    }

    if tasks.is_empty() {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { class: "empty-state-text", "Nothing completed yet. Get started!" }
                }
            }
        };
    }

    let groups = group_by_completion_date(&tasks);

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
