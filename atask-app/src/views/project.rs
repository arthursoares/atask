use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::section_header::SectionHeader;
use crate::components::task_item::TaskItem;
use crate::state::projects::ProjectState;

#[derive(Clone, PartialEq, Props)]
pub struct ProjectViewProps {
    project_id: String,
}

#[component]
pub fn ProjectView(props: ProjectViewProps) -> Element {
    let project_id = props.project_id.clone();
    let api: Signal<ApiClient> = use_context();
    let mut project_state: Signal<ProjectState> = use_context();
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    let mut loading = use_signal(|| false);
    let mut collapsed_sections: Signal<Vec<String>> = use_signal(|| Vec::new());

    // Fetch project data when project_id changes
    let pid = project_id.clone();
    let _loader = use_effect(move || {
        let pid = pid.clone();
        let api_clone = api.read().clone();
        let mut ps = project_state;
        loading.set(true);
        spawn(async move {
            let (tasks_result, sections_result) = tokio::join!(
                api_clone.list_tasks_by_project(&pid),
                api_clone.list_sections(&pid),
            );
            if let Ok(tasks) = tasks_result {
                let mut pt = ps.write().project_tasks;
                pt.write().insert(pid.clone(), tasks);
            }
            if let Ok(sections) = sections_result {
                let mut sec = ps.write().sections;
                sec.write().insert(pid.clone(), sections);
            }
            loading.set(false);
        });
    });

    let is_loading = *loading.read();
    let all_tasks: Vec<Task> = project_state
        .read()
        .project_tasks
        .read()
        .get(&project_id)
        .cloned()
        .unwrap_or_default();

    let sections = project_state
        .read()
        .sections
        .read()
        .get(&project_id)
        .cloned()
        .unwrap_or_default();

    if is_loading && all_tasks.is_empty() {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { class: "empty-state-text", "Loading..." }
                }
            }
        };
    }

    // Split tasks: sectionless vs by section
    let sectionless: Vec<Task> = all_tasks
        .iter()
        .filter(|t| t.section_id.is_none())
        .cloned()
        .collect();

    rsx! {
        div { class: "task-list",
            // Sectionless tasks
            for task in sectionless {
                {
                    let task_id = task.id.clone();
                    let task_id_complete = task.id.clone();
                    let is_selected = task.id == selected_id;
                    let is_today = task.today_index.is_some();
                    let pid = project_id.clone();
                    rsx! {
                        TaskItem {
                            key: "{task_id}",
                            task: task,
                            selected: is_selected,
                            today_view: is_today,
                            show_project: false,
                            on_select: move |id: String| {
                                selected_task_id.set(Some(id));
                            },
                            on_complete: move |_id: String| {
                                {
                                    let mut pt = project_state.write().project_tasks;
                                    let mut map = pt.write();
                                    if let Some(tasks) = map.get_mut(&pid) {
                                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id_complete) {
                                            t.status = if t.status == 0 { 1 } else { 0 };
                                        }
                                    }
                                }
                                let api_clone = api.read().clone();
                                let id = task_id_complete.clone();
                                let pid = pid.clone();
                                spawn(async move {
                                    if let Err(e) = api_clone.complete_task(&id).await {
                                        eprintln!("Failed to complete task: {e}");
                                        if let Ok(tasks) = api_clone.list_tasks_by_project(&pid).await {
                                            let mut pt = project_state.write().project_tasks;
                                            pt.write().insert(pid, tasks);
                                        }
                                    }
                                });
                            },
                        }
                    }
                }
            }

            NewTaskInline {
                on_create: {
                    let pid = project_id.clone();
                    move |title: String| {
                        let api_clone = api.read().clone();
                        let pid = pid.clone();
                        spawn(async move {
                            match api_clone.create_task(&title).await {
                                Ok(task) => {
                                    let mut pt = project_state.write().project_tasks;
                                    let mut map = pt.write();
                                    map.entry(pid).or_default().push(task);
                                }
                                Err(e) => {
                                    eprintln!("Failed to create task: {e}");
                                }
                            }
                        });
                    }
                },
            }

            // Sections
            for section in sections {
                {
                    let sid = section.id.clone();
                    let sid_toggle = section.id.clone();
                    let is_collapsed = collapsed_sections.read().contains(&sid);

                    let section_tasks: Vec<Task> = all_tasks
                        .iter()
                        .filter(|t| t.section_id.as_deref() == Some(&sid))
                        .cloned()
                        .collect();
                    let task_count = section_tasks.len();

                    rsx! {
                        SectionHeader {
                            key: "{sid}",
                            title: section.title.clone(),
                            count: task_count,
                            collapsed: is_collapsed,
                            on_toggle: move |_| {
                                let mut collapsed = collapsed_sections.write();
                                if let Some(pos) = collapsed.iter().position(|id| *id == sid_toggle) {
                                    collapsed.remove(pos);
                                } else {
                                    collapsed.push(sid_toggle.clone());
                                }
                            },
                        }

                        if !is_collapsed {
                            for task in section_tasks {
                                {
                                    let task_id = task.id.clone();
                                    let task_id_complete = task.id.clone();
                                    let is_selected = task.id == selected_id;
                                    let is_today = task.today_index.is_some();
                                    let pid = project_id.clone();
                                    rsx! {
                                        TaskItem {
                                            key: "{task_id}",
                                            task: task,
                                            selected: is_selected,
                                            today_view: is_today,
                                            show_project: false,
                                            on_select: move |id: String| {
                                                selected_task_id.set(Some(id));
                                            },
                                            on_complete: move |_id: String| {
                                                {
                                                    let mut pt = project_state.write().project_tasks;
                                                    let mut map = pt.write();
                                                    if let Some(tasks) = map.get_mut(&pid) {
                                                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id_complete) {
                                                            t.status = if t.status == 0 { 1 } else { 0 };
                                                        }
                                                    }
                                                }
                                                let api_clone = api.read().clone();
                                                let id = task_id_complete.clone();
                                                let pid = pid.clone();
                                                spawn(async move {
                                                    if let Err(e) = api_clone.complete_task(&id).await {
                                                        eprintln!("Failed to complete task: {e}");
                                                        if let Ok(tasks) = api_clone.list_tasks_by_project(&pid).await {
                                                            let mut pt = project_state.write().project_tasks;
                                                            pt.write().insert(pid, tasks);
                                                        }
                                                    }
                                                });
                                            },
                                        }
                                    }
                                }
                            }

                            NewTaskInline {
                                on_create: {
                                    let pid = project_id.clone();
                                    move |title: String| {
                                        let api_clone = api.read().clone();
                                        let pid = pid.clone();
                                        spawn(async move {
                                            match api_clone.create_task(&title).await {
                                                Ok(task) => {
                                                    let mut pt = project_state.write().project_tasks;
                                                    let mut map = pt.write();
                                                    map.entry(pid).or_default().push(task);
                                                }
                                                Err(e) => {
                                                    eprintln!("Failed to create task: {e}");
                                                }
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
