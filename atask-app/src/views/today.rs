use dioxus::prelude::*;
use crate::state::app::{TodayTasks, ApiSignal, SelectedTaskSignal};
use crate::components::task_item::TaskItem;
use crate::components::new_task_inline::NewTaskInline;

#[component]
pub fn TodayView() -> Element {
    let api: ApiSignal = use_context();
    let mut today: TodayTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();

    rsx! {
        div { class: "view-content",
            // Read signal INSIDE rsx!
            {
                let tasks = today.0.read().clone();
                if tasks.is_empty() {
                    rsx! {
                        div { class: "empty-state",
                            p { class: "empty-state-text", "Your day is clear." }
                        }
                    }
                } else {
                    rsx! {
                        div { class: "task-list",
                            for task in tasks.iter() {
                                {
                                    let task_id = task.id.clone();
                                    let is_selected = *selected.0.read() == Some(task_id.clone());
                                    rsx! {
                                        TaskItem {
                                            key: "{task_id}",
                                            task: task.clone(),
                                            selected: is_selected,
                                            today_view: true,
                                            show_project: true,
                                            on_select: move |id: String| {
                                                selected.0.set(Some(id));
                                            },
                                            on_complete: {
                                                let task_id = task_id.clone();
                                                move |_id: String| {
                                                    // Optimistic: mark as completed locally
                                                    let mut tasks = today.0.read().clone();
                                                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                                                        t.status = 1; // completed
                                                    }
                                                    today.0.set(tasks);

                                                    // API call + refetch
                                                    let api_clone = api.0.read().clone();
                                                    let tid = task_id.clone();
                                                    spawn(async move {
                                                        let _ = api_clone.complete_task(&tid).await;
                                                        // Refetch to get accurate state
                                                        if let Ok(fresh) = api_clone.list_today().await {
                                                            today.0.set(fresh);
                                                        }
                                                    });
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

            // New task creation
            NewTaskInline {
                on_create: move |title: String| {
                    let api_clone = api.0.read().clone();
                    spawn(async move {
                        // Create task (lands in inbox by default)
                        if let Ok(task) = api_clone.create_task(&title).await {
                            // Schedule for today (anytime)
                            let _ = api_clone.update_task_schedule(&task.id, "anytime").await;
                        }
                        // Refetch today list
                        if let Ok(fresh) = api_clone.list_today().await {
                            today.0.set(fresh);
                        }
                    });
                },
            }
        }
    }
}
