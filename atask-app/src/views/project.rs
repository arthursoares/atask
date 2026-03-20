use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::Task;
use crate::components::new_task_inline::NewTaskInline;
use crate::components::section_header::SectionHeader;
use crate::components::task_item::TaskItem;
use crate::components::toolbar::AddSectionTrigger;
use crate::state::navigation::SelectedTask;
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
    let mut selected_task: SelectedTask = use_context();
    let selected_id = selected_task.0.read().clone().unwrap_or_default();

    let mut add_section_trigger: AddSectionTrigger = use_context();
    let mut show_section_input = use_signal(|| false);
    let mut section_input_value = use_signal(|| String::new());

    let mut loading = use_signal(|| false);
    let mut collapsed_sections: Signal<Vec<String>> = use_signal(|| Vec::new());
    let mut dragging_id: Signal<Option<String>> = use_signal(|| None);
    let mut drag_over_id: Signal<Option<String>> = use_signal(|| None);

    // Watch for add-section trigger from toolbar
    let _section_trigger = use_effect(move || {
        if *add_section_trigger.0.read() {
            show_section_input.set(true);
            add_section_trigger.0.set(false);
        }
    });

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

    if !is_loading && all_tasks.is_empty() && sections.is_empty() {
        return rsx! {
            div { class: "task-list",
                div { class: "empty-state",
                    p { class: "empty-state-text", "No tasks in this project yet." }
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
                    let is_drag_over = drag_over_id.read().as_deref() == Some(&task.id);
                    rsx! {
                        TaskItem {
                            key: "{task_id}",
                            task: task,
                            selected: is_selected,
                            today_view: is_today,
                            show_project: false,
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
                                let pid = pid.clone();
                                move |_target_id: String| {
                                    drag_over_id.set(None);
                                    let dragged = dragging_id.read().clone();
                                    dragging_id.set(None);
                                    if let Some(dragged) = dragged {
                                        if dragged != task_id_drop {
                                            let mut tasks = project_state
                                                .read()
                                                .project_tasks
                                                .read()
                                                .get(&pid)
                                                .cloned()
                                                .unwrap_or_default();
                                            if let (Some(from), Some(to)) = (
                                                tasks.iter().position(|t| t.id == dragged),
                                                tasks.iter().position(|t| t.id == task_id_drop),
                                            ) {
                                                let item = tasks.remove(from);
                                                tasks.insert(to, item);
                                                project_state
                                                    .write()
                                                    .project_tasks
                                                    .write()
                                                    .insert(pid.clone(), tasks);

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
                                    let mut pt = project_state.write().project_tasks;
                                    let mut map = pt.write();
                                    if let Some(tasks) = map.get_mut(&pid) {
                                        tasks.retain(|t| t.id != task_id_complete);
                                    }
                                }
                                let api_clone = api.read().clone();
                                let id = task_id_complete.clone();
                                let pid = pid.clone();
                                spawn(async move {
                                    if let Err(e) = api_clone.complete_task(&id).await {
                                        eprintln!("Failed to complete task: {e}");
                                    }
                                    // Always refetch to stay in sync
                                    if let Ok(tasks) = api_clone.list_tasks_by_project(&pid).await {
                                        let mut pt = project_state.write().project_tasks;
                                        pt.write().insert(pid, tasks);
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
                                                selected_task.0.set(Some(id));
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

            // New section inline input
            if *show_section_input.read() {
                div { class: "new-section-input-row",
                    input {
                        class: "input",
                        placeholder: "Section title...",
                        value: "{section_input_value.read()}",
                        autofocus: true,
                        oninput: move |evt: Event<FormData>| {
                            section_input_value.set(evt.value().clone());
                        },
                        onkeydown: {
                            let pid = project_id.clone();
                            move |evt: Event<KeyboardData>| {
                                if evt.key() == Key::Enter {
                                    let title = section_input_value.read().clone();
                                    if !title.trim().is_empty() {
                                        let api_clone = api.read().clone();
                                        let pid = pid.clone();
                                        spawn(async move {
                                            match api_clone.create_section(&pid, &title).await {
                                                Ok(section) => {
                                                    let mut sec = project_state.write().sections;
                                                    sec.write()
                                                        .entry(pid)
                                                        .or_default()
                                                        .push(section);
                                                }
                                                Err(e) => {
                                                    eprintln!("Failed to create section: {e}");
                                                }
                                            }
                                        });
                                        section_input_value.set(String::new());
                                        show_section_input.set(false);
                                    }
                                } else if evt.key() == Key::Escape {
                                    section_input_value.set(String::new());
                                    show_section_input.set(false);
                                }
                            }
                        },
                    }
                }
            }
        }
    }
}
