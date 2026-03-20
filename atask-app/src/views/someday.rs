use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::task_item::TaskItem;
use crate::state::navigation::SelectedTask;
use crate::state::tasks::TaskState;

#[component]
pub fn SomedayView() -> Element {
    let api: Signal<ApiClient> = use_context();
    let mut task_state: Signal<TaskState> = use_context();
    let mut selected_task: SelectedTask = use_context();
    let selected_id = selected_task.0.read().clone().unwrap_or_default();

    let mut dragging_id: Signal<Option<String>> = use_signal(|| None);
    let mut drag_over_id: Signal<Option<String>> = use_signal(|| None);

    let tasks: Vec<Task> = task_state.read().someday.read().clone();
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
                    p { class: "empty-state-text", "No someday tasks. Everything is decided." }
                }
                NewTaskInline {
                    on_create: move |title: String| {
                        let api_clone = api.read().clone();
                        spawn(async move {
                            match api_clone.create_task(&title).await {
                                Ok(task) => {
                                    let task_id = task.id.clone();
                                    task_state.write().someday.write().push(task);
                                    // New tasks default to inbox; move to someday for this view
                                    if let Err(e) = api_clone.update_task_schedule(&task_id, "someday").await {
                                        eprintln!("Failed to set schedule to someday: {e}");
                                    }
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
                    let is_drag_over = drag_over_id.read().as_deref() == Some(&task.id);
                    rsx! {
                        TaskItem {
                            key: "{task_id}",
                            task: task,
                            selected: is_selected,
                            today_view: false,
                            draggable: true,
                            drag_over: is_drag_over,
                            on_select: move |id: String| {
                                selected_task.0.set(Some(id));
                            },
                            on_drag_start: move |id: String| {
                                dragging_id.set(Some(id));
                            },
                            on_drop_target: {
                                let task_id_drop = task_id.clone();
                                move |_target_id: String| {
                                    drag_over_id.set(None);
                                    let dragged = dragging_id.read().clone();
                                    dragging_id.set(None);
                                    if let Some(dragged) = dragged {
                                        if dragged != task_id_drop {
                                            let mut tasks = task_state.read().someday.read().clone();
                                            if let (Some(from), Some(to)) = (
                                                tasks.iter().position(|t| t.id == dragged),
                                                tasks.iter().position(|t| t.id == task_id_drop),
                                            ) {
                                                let item = tasks.remove(from);
                                                tasks.insert(to, item);
                                                task_state.write().someday.set(tasks);

                                                let api_clone = api.read().clone();
                                                let dragged_id = dragged.clone();
                                                let new_index = to as i32;
                                                spawn(async move {
                                                    if let Err(e) = api_clone.reorder_task(&dragged_id, new_index).await {
                                                        eprintln!("Failed to reorder task: {e}");
                                                    }
                                                });
                                            }
                                        }
                                    }
                                }
                            },
                            on_complete: move |_id: String| {
                                // Optimistic: remove from view immediately
                                {
                                    let mut someday = task_state.write().someday;
                                    someday.write().retain(|t| t.id != task_id_complete);
                                }
                                let api_clone = api.read().clone();
                                let id = task_id_complete.clone();
                                spawn(async move {
                                    if let Err(e) = api_clone.complete_task(&id).await {
                                        eprintln!("Failed to complete task: {e}");
                                    }
                                    // Always refetch to stay in sync
                                    if let Ok(tasks) = api_clone.list_someday().await {
                                        task_state.write().someday.set(tasks);
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
                                let task_id = task.id.clone();
                                task_state.write().someday.write().push(task);
                                // New tasks default to inbox; move to someday for this view
                                if let Err(e) = api_clone.update_task_schedule(&task_id, "someday").await {
                                    eprintln!("Failed to set schedule to someday: {e}");
                                }
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
