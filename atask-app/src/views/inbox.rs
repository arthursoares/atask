use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::project_picker::ProjectPicker;
use crate::components::task_item::TaskItem;
use crate::state::tasks::TaskState;

#[component]
pub fn InboxView() -> Element {
    let api: Signal<ApiClient> = use_context();
    let mut task_state: Signal<TaskState> = use_context();
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let tasks: Vec<Task> = task_state.read().inbox.read().clone();
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
                    p { class: "empty-state-text empty-state-success", "Inbox Zero \u{2713}" }
                }
                NewTaskInline {
                    on_create: move |title: String| {
                        let api_clone = api.read().clone();
                        spawn(async move {
                            match api_clone.create_task(&title).await {
                                Ok(task) => {
                                    task_state.write().inbox.write().push(task);
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

    let mut picker_open_for: Signal<Option<String>> = use_signal(|| None);

    rsx! {
        div { class: "task-list",
            for task in tasks {
                {
                    let task_id = task.id.clone();
                    let task_id_complete = task.id.clone();
                    let task_id_today = task.id.clone();
                    let task_id_someday = task.id.clone();
                    let task_id_project = task.id.clone();
                    let task_id_picker = task.id.clone();
                    let is_selected = task.id == selected_id;
                    rsx! {
                        div {
                            key: "{task_id}",
                            class: "inbox-task-row",
                            TaskItem {
                                task: task,
                                selected: is_selected,
                                today_view: false,
                                on_select: move |id: String| {
                                    selected_task_id.set(Some(id));
                                },
                                on_complete: move |_id: String| {
                                    // Optimistic: remove from view immediately
                                    {
                                        let mut inbox = task_state.write().inbox;
                                        inbox.write().retain(|t| t.id != task_id_complete);
                                    }
                                    let api_clone = api.read().clone();
                                    let id = task_id_complete.clone();
                                    spawn(async move {
                                        if let Err(e) = api_clone.complete_task(&id).await {
                                            eprintln!("Failed to complete task: {e}");
                                        }
                                        if let Ok(tasks) = api_clone.list_inbox().await {
                                            task_state.write().inbox.set(tasks);
                                        }
                                    });
                                },
                            }
                            div { class: "inbox-actions",
                                button {
                                    class: "inbox-action-btn",
                                    title: "Schedule for Today",
                                    onclick: move |_| {
                                        // Optimistic: remove from inbox
                                        {
                                            let mut inbox = task_state.write().inbox;
                                            inbox.write().retain(|t| t.id != task_id_today);
                                        }
                                        let api_clone = api.read().clone();
                                        let id = task_id_today.clone();
                                        spawn(async move {
                                            if let Err(e) = api_clone.update_task_schedule(&id, "anytime").await {
                                                eprintln!("Failed to schedule task for today: {e}");
                                            }
                                            if let Ok(tasks) = api_clone.list_inbox().await {
                                                task_state.write().inbox.set(tasks);
                                            }
                                        });
                                    },
                                    "\u{2605}"
                                }
                                button {
                                    class: "inbox-action-btn",
                                    title: "Defer to Someday",
                                    onclick: move |_| {
                                        // Optimistic: remove from inbox
                                        {
                                            let mut inbox = task_state.write().inbox;
                                            inbox.write().retain(|t| t.id != task_id_someday);
                                        }
                                        let api_clone = api.read().clone();
                                        let id = task_id_someday.clone();
                                        spawn(async move {
                                            if let Err(e) = api_clone.update_task_schedule(&id, "someday").await {
                                                eprintln!("Failed to defer task to someday: {e}");
                                            }
                                            if let Ok(tasks) = api_clone.list_inbox().await {
                                                task_state.write().inbox.set(tasks);
                                            }
                                        });
                                    },
                                    "\u{1F4A4}"
                                }
                                button {
                                    class: "inbox-action-btn",
                                    title: "Move to Project",
                                    onclick: move |_| {
                                        let current = picker_open_for.read().clone();
                                        if current.as_deref() == Some(&*task_id_project) {
                                            picker_open_for.set(None);
                                        } else {
                                            picker_open_for.set(Some(task_id_project.clone()));
                                        }
                                    },
                                    "\u{1F4C1}"
                                }
                            }
                            if picker_open_for.read().as_deref() == Some(&*task_id_picker) {
                                ProjectPicker {
                                    current_project_id: None,
                                    on_select: move |project_id: Option<String>| {
                                        picker_open_for.set(None);
                                        if let Some(pid) = project_id {
                                            // Optimistic: remove from inbox
                                            let tid = task_id_picker.clone();
                                            {
                                                let mut inbox = task_state.write().inbox;
                                                inbox.write().retain(|t| t.id != tid);
                                            }
                                            let api_clone = api.read().clone();
                                            spawn(async move {
                                                if let Err(e) = api_clone.move_task_to_project(&tid, Some(&pid)).await {
                                                    eprintln!("Failed to move task to project: {e}");
                                                }
                                                if let Ok(tasks) = api_clone.list_inbox().await {
                                                    task_state.write().inbox.set(tasks);
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

            NewTaskInline {
                on_create: move |title: String| {
                    let api_clone = api.read().clone();
                    spawn(async move {
                        match api_clone.create_task(&title).await {
                            Ok(task) => {
                                task_state.write().inbox.write().push(task);
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
