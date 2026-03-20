use dioxus::prelude::*;
use crate::state::app::{ApiSignal, SelectedTaskSignal, ProjectTasks, ProjectSections};
use crate::components::task_item::TaskItem;
use crate::components::new_task_inline::NewTaskInline;

#[derive(Clone, PartialEq, Props)]
pub struct ProjectViewProps {
    project_id: String,
}

#[component]
pub fn ProjectView(props: ProjectViewProps) -> Element {
    let api: ApiSignal = use_context();
    let mut project_tasks: ProjectTasks = use_context();
    let mut project_sections: ProjectSections = use_context();
    let mut selected: SelectedTaskSignal = use_context();

    let project_id = props.project_id.clone();

    // Fetch tasks and sections on mount / when project_id changes
    use_effect({
        let pid = project_id.clone();
        move || {
            let pid = pid.clone();
            let api_clone = api.0.read().clone();
            spawn(async move {
                let (tasks_r, sections_r) = tokio::join!(
                    api_clone.list_tasks_by_project(&pid),
                    api_clone.list_sections(&pid),
                );
                if let Ok(t) = tasks_r {
                    let mut map = project_tasks.0.read().clone();
                    map.insert(pid.clone(), t);
                    project_tasks.0.set(map);
                }
                if let Ok(s) = sections_r {
                    let mut map = project_sections.0.read().clone();
                    map.insert(pid.clone(), s);
                    project_sections.0.set(map);
                }
            });
        }
    });

    let pid = project_id.clone();

    rsx! {
        div { class: "view-content",
            {
                let tasks_map = project_tasks.0.read().clone();
                let sections_map = project_sections.0.read().clone();
                let tasks = tasks_map.get(&pid).cloned().unwrap_or_default();
                let sections = sections_map.get(&pid).cloned().unwrap_or_default();

                // Sectionless tasks: no section_id
                let sectionless: Vec<_> = tasks.iter().filter(|t| t.section_id.is_none()).cloned().collect();

                // Check if everything is empty
                let has_content = !tasks.is_empty() || !sections.is_empty();

                if !has_content {
                    rsx! {
                        div { class: "empty-state",
                            p { class: "empty-state-text", "No tasks in this project yet." }
                        }
                    }
                } else {
                    rsx! {
                        // Sectionless tasks at top
                        if !sectionless.is_empty() {
                            div { class: "task-list",
                                for task in sectionless.iter() {
                                    {
                                        let task_id = task.id.clone();
                                        let is_selected = *selected.0.read() == Some(task_id.clone());
                                        rsx! {
                                            TaskItem {
                                                key: "{task_id}",
                                                task: task.clone(),
                                                selected: is_selected,
                                                today_view: false,
                                                show_project: false,
                                                on_select: move |id: String| {
                                                    selected.0.set(Some(id));
                                                },
                                                on_complete: {
                                                    let task_id = task_id.clone();
                                                    let pid = pid.clone();
                                                    move |_id: String| {
                                                        let api_clone = api.0.read().clone();
                                                        let tid = task_id.clone();
                                                        let pid = pid.clone();
                                                        spawn(async move {
                                                            let _ = api_clone.complete_task(&tid).await;
                                                            if let Ok(fresh) = api_clone.list_tasks_by_project(&pid).await {
                                                                let mut map = project_tasks.0.read().clone();
                                                                map.insert(pid, fresh);
                                                                project_tasks.0.set(map);
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

                        // Each section
                        for section in sections.iter() {
                            {
                                let section_id = section.id.clone();
                                let section_title = section.title.clone();
                                let section_tasks: Vec<_> = tasks.iter()
                                    .filter(|t| t.section_id.as_deref() == Some(&section_id))
                                    .cloned()
                                    .collect();

                                rsx! {
                                    SectionBlock {
                                        key: "{section_id}",
                                        section_id: section_id,
                                        section_title: section_title,
                                        tasks: section_tasks,
                                        project_id: pid.clone(),
                                    }
                                }
                            }
                        }
                    }
                }
            }

            // New task inline for sectionless tasks
            NewTaskInline {
                on_create: {
                    let pid = project_id.clone();
                    move |title: String| {
                        let api_clone = api.0.read().clone();
                        let pid = pid.clone();
                        spawn(async move {
                            if let Ok(task) = api_clone.create_task(&title).await {
                                let _ = api_clone.move_task_to_project(&task.id, Some(&pid)).await;
                            }
                            if let Ok(fresh) = api_clone.list_tasks_by_project(&pid).await {
                                let mut map = project_tasks.0.read().clone();
                                map.insert(pid, fresh);
                                project_tasks.0.set(map);
                            }
                        });
                    }
                },
            }
        }
    }
}

#[derive(Clone, PartialEq, Props)]
struct SectionBlockProps {
    section_id: String,
    section_title: String,
    tasks: Vec<crate::api::types::Task>,
    project_id: String,
}

#[component]
fn SectionBlock(props: SectionBlockProps) -> Element {
    let api: ApiSignal = use_context();
    let mut project_tasks: ProjectTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();

    let section_id = props.section_id.clone();
    let pid = props.project_id.clone();

    rsx! {
        div { class: "section-block",
            // Section header (static divider)
            div { class: "section-header",
                span { class: "section-title", "{props.section_title}" }
                if !props.tasks.is_empty() {
                    span { class: "section-count", "{props.tasks.len()}" }
                }
                div { class: "section-line" }
            }

            div { class: "task-list",
                for task in props.tasks.iter() {
                    {
                        let task_id = task.id.clone();
                        let is_selected = *selected.0.read() == Some(task_id.clone());
                        rsx! {
                            TaskItem {
                                key: "{task_id}",
                                task: task.clone(),
                                selected: is_selected,
                                today_view: false,
                                show_project: false,
                                on_select: move |id: String| {
                                    selected.0.set(Some(id));
                                },
                                on_complete: {
                                    let task_id = task_id.clone();
                                    let pid = pid.clone();
                                    move |_id: String| {
                                        let api_clone = api.0.read().clone();
                                        let tid = task_id.clone();
                                        let pid = pid.clone();
                                        spawn(async move {
                                            let _ = api_clone.complete_task(&tid).await;
                                            if let Ok(fresh) = api_clone.list_tasks_by_project(&pid).await {
                                                let mut map = project_tasks.0.read().clone();
                                                map.insert(pid, fresh);
                                                project_tasks.0.set(map);
                                            }
                                        });
                                    }
                                },
                            }
                        }
                    }
                }
            }

            // New task inline for this section
            NewTaskInline {
                on_create: {
                    let pid = pid.clone();
                    let section_id = section_id.clone();
                    move |title: String| {
                        let api_clone = api.0.read().clone();
                        let pid = pid.clone();
                        let sid = section_id.clone();
                        spawn(async move {
                            if let Ok(task) = api_clone.create_task(&title).await {
                                let _ = api_clone.move_task_to_project(&task.id, Some(&pid)).await;
                                let _ = api_clone.move_task_to_section(&task.id, Some(&sid)).await;
                            }
                            if let Ok(fresh) = api_clone.list_tasks_by_project(&pid).await {
                                let mut map = project_tasks.0.read().clone();
                                map.insert(pid, fresh);
                                project_tasks.0.set(map);
                            }
                        });
                    }
                },
            }
        }
    }
}
