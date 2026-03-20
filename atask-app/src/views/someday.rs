use dioxus::prelude::*;
use crate::state::app::{SomedayTasks, ApiSignal, SelectedTaskSignal};
use crate::components::task_item::TaskItem;
use crate::components::new_task_inline::NewTaskInline;

#[component]
pub fn SomedayView() -> Element {
    let api: ApiSignal = use_context();
    let mut someday: SomedayTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();

    rsx! {
        div { class: "view-content",
            {
                let tasks = someday.0.read().clone();
                if tasks.is_empty() {
                    rsx! {
                        div { class: "empty-state",
                            p { class: "empty-state-text", "No someday tasks. Everything is decided." }
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
                                                    let mut tasks = someday.0.read().clone();
                                                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                                                        t.status = 1;
                                                    }
                                                    someday.0.set(tasks);

                                                    let api_clone = api.0.read().clone();
                                                    let tid = task_id.clone();
                                                    spawn(async move {
                                                        let _ = api_clone.complete_task(&tid).await;
                                                        if let Ok(fresh) = api_clone.list_someday().await {
                                                            someday.0.set(fresh);
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

            NewTaskInline {
                on_create: move |title: String| {
                    let api_clone = api.0.read().clone();
                    spawn(async move {
                        if let Ok(task) = api_clone.create_task(&title).await {
                            let _ = api_clone.update_task_schedule(&task.id, "someday").await;
                        }
                        if let Ok(fresh) = api_clone.list_someday().await {
                            someday.0.set(fresh);
                        }
                    });
                },
            }
        }
    }
}
