use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::task_item::TaskItem;
use crate::state::tasks::TaskState;

#[component]
pub fn TodayView() -> Element {
    let api: Signal<ApiClient> = use_context();
    let mut task_state: Signal<TaskState> = use_context();
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let tasks: Vec<Task> = task_state.read().today.read().clone();
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
                    p { class: "empty-state-text", "Nothing planned for today." }
                }
                NewTaskInline {
                    on_create: move |title: String| {
                        let api_clone = api.read().clone();
                        spawn(async move {
                            match api_clone.create_task(&title).await {
                                Ok(task) => {
                                    task_state.write().today.write().push(task);
                                }
                                Err(e) => {
                                    eprintln!("Failed to create task: {e}");
                                }
                            }
                        });
                    },
                }
            }
        };
    }

    rsx! {
        div { class: "task-list",
            for task in tasks {
                {
                    let task_id = task.id.clone();
                    let task_id_complete = task.id.clone();
                    let is_selected = task.id == selected_id;
                    rsx! {
                        TaskItem {
                            key: "{task_id}",
                            task: task,
                            selected: is_selected,
                            today_view: true,
                            on_select: move |id: String| {
                                selected_task_id.set(Some(id));
                            },
                            on_complete: move |_id: String| {
                                // Optimistic: mark completed locally
                                {
                                    let mut today = task_state.write().today;
                                    let mut tasks = today.write();
                                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id_complete) {
                                        t.status = if t.status == 0 { 1 } else { 0 };
                                    }
                                }
                                // Fire API call
                                let api_clone = api.read().clone();
                                let id = task_id_complete.clone();
                                spawn(async move {
                                    if let Err(e) = api_clone.complete_task(&id).await {
                                        eprintln!("Failed to complete task: {e}");
                                        // Rollback: re-fetch
                                        if let Ok(tasks) = api_clone.list_today().await {
                                            task_state.write().today.set(tasks);
                                        }
                                    }
                                });
                            },
                        }
                    }
                }
            }

            NewTaskInline {
                on_create: move |title: String| {
                    let api_clone = api.read().clone();
                    spawn(async move {
                        match api_clone.create_task(&title).await {
                            Ok(task) => {
                                task_state.write().today.write().push(task);
                            }
                            Err(e) => {
                                eprintln!("Failed to create task: {e}");
                            }
                        }
                    });
                },
            }
        }
    }
}
