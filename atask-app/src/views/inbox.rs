use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::task_item::TaskItem;

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

#[component]
pub fn InboxView() -> Element {
    let mut tasks = use_signal(|| {
        vec![
            Task {
                id: "inbox-1".into(),
                title: "Review Pascal's feedback on metering spec".into(),
                notes: "Added 2 hours ago".into(),
                schedule: 0,
                index: 0,
                ..default_task()
            },
            Task {
                id: "inbox-2".into(),
                title: "Look into NATS JetStream for persistent event delivery".into(),
                notes: "Added yesterday".into(),
                schedule: 0,
                index: 1,
                ..default_task()
            },
            Task {
                id: "inbox-3".into(),
                title: "Investigate Home Assistant energy monitoring dashboard".into(),
                notes: "Added 2 days ago".into(),
                schedule: 0,
                index: 2,
                ..default_task()
            },
        ]
    });

    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let task_list: Vec<Task> = tasks.read().clone();

    if task_list.is_empty() {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { style: "color: var(--success);", "Inbox Zero \u{2713}" }
                }
            }
        };
    }

    rsx! {
        div { class: "task-list",
            for task in task_list {
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
                                let mut t = tasks.write();
                                if let Some(task) = t.iter_mut().find(|t| t.id == task_id_complete) {
                                    task.status = if task.status == 0 { 1 } else { 0 };
                                }
                            },
                        }
                    }
                }
            }

            NewTaskInline {
                on_create: move |_title: String| {
                    // placeholder — will create tasks via API later
                },
            }
        }
    }
}
