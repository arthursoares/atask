use dioxus::prelude::*;
use crate::state::app::{InboxTasks, ApiSignal, SelectedTaskSignal};
use crate::components::task_item::TaskItem;
use crate::components::new_task_inline::NewTaskInline;

#[component]
pub fn InboxView() -> Element {
    let api: ApiSignal = use_context();
    let mut inbox: InboxTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();

    rsx! {
        div { class: "view-content",
            // Read signal INSIDE rsx!
            {
                let tasks = inbox.0.read().clone();
                if tasks.is_empty() {
                    rsx! {
                        div { class: "empty-state",
                            p { class: "empty-state-text empty-state-success", "Inbox Zero \u{2713}" }
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
                                            today_view: false,
                                            show_project: true,
                                            on_select: move |id: String| {
                                                selected.0.set(Some(id));
                                            },
                                            on_complete: {
                                                let task_id = task_id.clone();
                                                move |_id: String| {
                                                    // Optimistic: mark as completed locally
                                                    let mut tasks = inbox.0.read().clone();
                                                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                                                        t.status = 1; // completed
                                                    }
                                                    inbox.0.set(tasks);

                                                    // Fire API — no refetch, task stays with strikethrough
                                                    let api_clone = api.0.read().clone();
                                                    let tid = task_id.clone();
                                                    spawn(async move {
                                                        let _ = api_clone.complete_task(&tid).await;
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
                        // Create task — lands in inbox by default, no schedule change needed
                        let _ = api_clone.create_task(&title).await;
                        // Refetch inbox list
                        if let Ok(fresh) = api_clone.list_inbox().await {
                            inbox.0.set(fresh);
                        }
                    });
                },
            }
        }
    }
}
