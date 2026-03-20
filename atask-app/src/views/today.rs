use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::section_header::SectionHeader;
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
pub fn TodayView() -> Element {
    let mut tasks = use_signal(|| {
        vec![
            Task {
                id: "1".into(),
                title: "Design component library for macOS client".into(),
                status: 0,
                schedule: 1,
                today_index: Some(0),
                project_id: Some("p1".into()),
                ..default_task()
            },
            Task {
                id: "2".into(),
                title: "Write domain event catalog documentation".into(),
                status: 0,
                schedule: 1,
                today_index: Some(1),
                project_id: Some("p1".into()),
                ..default_task()
            },
            Task {
                id: "3".into(),
                title: "Set up Proxmox backup schedule for NUC".into(),
                status: 0,
                schedule: 1,
                today_index: Some(2),
                project_id: Some("p2".into()),
                ..default_task()
            },
            Task {
                id: "4".into(),
                title: "Update Roon now-playing display with color extraction".into(),
                status: 0,
                schedule: 1,
                today_index: Some(3),
                project_id: Some("p3".into()),
                deadline: Some("2026-03-21".into()),
                ..default_task()
            },
            Task {
                id: "5".into(),
                title: "Research Swift vs Dioxus for native client".into(),
                status: 0,
                schedule: 1,
                today_index: Some(4),
                project_id: Some("p1".into()),
                ..default_task()
            },
        ]
    });

    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let mut evening_collapsed = use_signal(|| false);

    let morning_tasks: Vec<Task> = tasks.read().iter().take(4).cloned().collect();
    let evening_tasks: Vec<Task> = tasks.read().iter().skip(4).cloned().collect();

    rsx! {
        div { class: "task-list",
            for task in morning_tasks {
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
                                let mut t = tasks.write();
                                if let Some(task) = t.iter_mut().find(|t| t.id == task_id_complete) {
                                    task.status = if task.status == 0 { 1 } else { 0 };
                                }
                            },
                        }
                    }
                }
            }

            SectionHeader {
                title: "This Evening".to_string(),
                count: evening_tasks.len(),
                collapsed: *evening_collapsed.read(),
                on_toggle: move |_| {
                    let current = *evening_collapsed.read();
                    evening_collapsed.set(!current);
                },
            }

            if !*evening_collapsed.read() {
                for task in evening_tasks {
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
                                    let mut t = tasks.write();
                                    if let Some(task) = t.iter_mut().find(|t| t.id == task_id_complete) {
                                        task.status = if task.status == 0 { 1 } else { 0 };
                                    }
                                },
                            }
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
