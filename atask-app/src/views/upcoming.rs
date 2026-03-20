use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::Task;
use crate::components::section_header::SectionHeader;
use crate::components::task_item::TaskItem;
use crate::state::tasks::TaskState;

/// Group tasks by start_date for the upcoming view.
struct DateGroup {
    label: String,
    tasks: Vec<Task>,
}

fn group_by_date(tasks: &[Task]) -> Vec<DateGroup> {
    use std::collections::BTreeMap;
    let mut map: BTreeMap<String, Vec<Task>> = BTreeMap::new();
    for task in tasks {
        let key = task.start_date.clone().unwrap_or_else(|| "No Date".to_string());
        map.entry(key).or_default().push(task.clone());
    }
    map.into_iter()
        .map(|(label, tasks)| DateGroup { label, tasks })
        .collect()
}

#[component]
pub fn UpcomingView() -> Element {
    let api: Signal<ApiClient> = use_context();
    let mut task_state: Signal<TaskState> = use_context();
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let tasks: Vec<Task> = task_state.read().upcoming.read().clone();
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
                    p { "Nothing scheduled ahead." }
                }
            }
        };
    }

    let groups = group_by_date(&tasks);

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
                                    // Optimistic: remove from view immediately
                                    {
                                        let mut upcoming = task_state.write().upcoming;
                                        upcoming.write().retain(|t| t.id != task_id_complete);
                                    }
                                    let api_clone = api.read().clone();
                                    let id = task_id_complete.clone();
                                    spawn(async move {
                                        if let Err(e) = api_clone.complete_task(&id).await {
                                            eprintln!("Failed to complete task: {e}");
                                        }
                                        // Always refetch to stay in sync
                                        if let Ok(tasks) = api_clone.list_upcoming().await {
                                            task_state.write().upcoming.set(tasks);
                                        }
                                    });
                                },
                            }
                        }
                    }
                }
            }
        }
    }
}
