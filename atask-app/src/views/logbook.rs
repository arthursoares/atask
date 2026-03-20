use std::collections::BTreeMap;
use dioxus::prelude::*;
use crate::state::app::{LogbookTasks, ApiSignal, SelectedTaskSignal};
use crate::state::date_fmt::format_section_date;
use crate::api::types::Task;
use crate::components::checkbox::Checkbox;
use crate::components::section_header::SectionHeader;
use crate::components::task_meta::TaskMeta;

/// Extract the date portion (YYYY-MM-DD) from a datetime string.
/// Handles both "2026-03-20T15:04:05Z" and "2026-03-20" formats.
fn extract_date(datetime: &str) -> String {
    datetime.split('T').next().unwrap_or(datetime).to_string()
}

#[component]
pub fn LogbookView() -> Element {
    let api: ApiSignal = use_context();
    let mut logbook: LogbookTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();
    let mut collapsed: Signal<std::collections::HashSet<String>> = use_signal(|| std::collections::HashSet::new());

    rsx! {
        div { class: "view-content",
            {
                let tasks = logbook.0.read().clone();
                if tasks.is_empty() {
                    rsx! {
                        div { class: "empty-state",
                            p { class: "empty-state-text", "Nothing completed yet. Get started!" }
                        }
                    }
                } else {
                    // Group tasks by completed_at date, descending
                    let mut groups: BTreeMap<String, Vec<Task>> = BTreeMap::new();
                    for task in tasks.iter() {
                        let key = task.completed_at.as_deref()
                            .map(extract_date)
                            .unwrap_or_default();
                        groups.entry(key).or_default().push(task.clone());
                    }

                    // Reverse to get newest first
                    let groups_desc: Vec<(String, Vec<Task>)> = groups.into_iter().rev().collect();
                    let collapsed_set = collapsed.read().clone();

                    rsx! {
                        for (date_key, group_tasks) in groups_desc.iter() {
                            {
                                let date_key = date_key.clone();
                                let label = format_section_date(&date_key);
                                let count = group_tasks.len();
                                let is_collapsed = collapsed_set.contains(&date_key);

                                rsx! {
                                    SectionHeader {
                                        title: label,
                                        count: count,
                                        collapsed: is_collapsed,
                                        on_toggle: {
                                            let dk = date_key.clone();
                                            move |_| {
                                                let mut set = collapsed.read().clone();
                                                if set.contains(&dk) {
                                                    set.remove(&dk);
                                                } else {
                                                    set.insert(dk.clone());
                                                }
                                                collapsed.set(set);
                                            }
                                        },
                                    }

                                    if !is_collapsed {
                                        div { class: "task-list",
                                            for task in group_tasks.iter() {
                                                {
                                                    let task_id = task.id.clone();
                                                    let is_selected = *selected.0.read() == Some(task_id.clone());
                                                    let is_completed = task.is_completed();
                                                    let is_cancelled = task.is_cancelled();

                                                    let item_class = {
                                                        let mut c = "task-item".to_string();
                                                        if is_selected {
                                                            c.push_str(" selected");
                                                        }
                                                        c
                                                    };

                                                    let title_class = if is_cancelled {
                                                        "task-title completed tertiary"
                                                    } else {
                                                        "task-title completed"
                                                    };

                                                    rsx! {
                                                        div {
                                                            class: "{item_class}",
                                                            onclick: {
                                                                let tid = task_id.clone();
                                                                move |_| selected.0.set(Some(tid.clone()))
                                                            },

                                                            if is_cancelled {
                                                                // Cancelled: show X instead of checkbox
                                                                div { class: "checkbox cancelled-icon",
                                                                    span { "\u{2715}" }
                                                                }
                                                            } else if is_completed {
                                                                // Completed: checked checkbox, click to reopen
                                                                Checkbox {
                                                                    checked: true,
                                                                    today: false,
                                                                    on_toggle: {
                                                                        let task_id = task_id.clone();
                                                                        move |_| {
                                                                            // Optimistic: remove from logbook
                                                                            let tasks = logbook.0.read().clone();
                                                                            let filtered: Vec<Task> = tasks.into_iter()
                                                                                .filter(|t| t.id != task_id)
                                                                                .collect();
                                                                            logbook.0.set(filtered);

                                                                            let api_clone = api.0.read().clone();
                                                                            let tid = task_id.clone();
                                                                            spawn(async move {
                                                                                let _ = api_clone.reopen_task(&tid).await;
                                                                                if let Ok(fresh) = api_clone.list_logbook().await {
                                                                                    logbook.0.set(fresh);
                                                                                }
                                                                            });
                                                                        }
                                                                    },
                                                                }
                                                            }

                                                            span { class: "{title_class}", "{task.title}" }

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
                        }
                    }
                }
            }
        }
    }
}
