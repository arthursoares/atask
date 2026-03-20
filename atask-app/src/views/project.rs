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
        project_id: Some("p1".into()),
        section_id: None,
        area_id: None,
        location_id: None,
        recurrence_rule: None,
        tags: None,
        deleted: false,
        deleted_at: None,
    }
}

struct SectionData {
    id: String,
    title: String,
    tasks: Vec<Task>,
}

fn sample_project_data() -> (Vec<Task>, Vec<SectionData>) {
    let sectionless = vec![
        Task {
            id: "p1-t1".into(),
            title: "Define success criteria for v0 launch".into(),
            deadline: Some("2026-03-28".into()),
            index: 0,
            ..default_task()
        },
        Task {
            id: "p1-t2".into(),
            title: "Write CLAUDE.md for the project repo".into(),
            today_index: Some(0),
            index: 1,
            ..default_task()
        },
    ];

    let sections = vec![
        SectionData {
            id: "s1".into(),
            title: "Domain & API".into(),
            tasks: vec![
                Task {
                    id: "p1-t3".into(),
                    title: "Draft domain model entity relationships".into(),
                    status: 1,
                    section_id: Some("s1".into()),
                    index: 0,
                    ..default_task()
                },
                Task {
                    id: "p1-t4".into(),
                    title: "Review API endpoint naming conventions".into(),
                    status: 1,
                    section_id: Some("s1".into()),
                    index: 1,
                    ..default_task()
                },
                Task {
                    id: "p1-t5".into(),
                    title: "Write domain event catalog documentation".into(),
                    section_id: Some("s1".into()),
                    index: 2,
                    ..default_task()
                },
                Task {
                    id: "p1-t6".into(),
                    title: "Set up SQLite migration scaffold with goose".into(),
                    section_id: Some("s1".into()),
                    index: 3,
                    ..default_task()
                },
            ],
        },
        SectionData {
            id: "s2".into(),
            title: "Client".into(),
            tasks: vec![
                Task {
                    id: "p1-t7".into(),
                    title: "Design component library for macOS client".into(),
                    today_index: Some(1),
                    section_id: Some("s2".into()),
                    index: 0,
                    ..default_task()
                },
                Task {
                    id: "p1-t8".into(),
                    title: "Research Swift vs Dioxus for native client".into(),
                    section_id: Some("s2".into()),
                    index: 1,
                    ..default_task()
                },
                Task {
                    id: "p1-t9".into(),
                    title: "Scaffold Dioxus project with routing".into(),
                    section_id: Some("s2".into()),
                    index: 2,
                    ..default_task()
                },
                Task {
                    id: "p1-t10".into(),
                    title: "Implement SSE event subscription in Rust".into(),
                    section_id: Some("s2".into()),
                    index: 3,
                    ..default_task()
                },
            ],
        },
        SectionData {
            id: "s3".into(),
            title: "Infrastructure".into(),
            tasks: vec![
                Task {
                    id: "p1-t11".into(),
                    title: "Write Dockerfile with multi-stage build".into(),
                    section_id: Some("s3".into()),
                    index: 0,
                    ..default_task()
                },
                Task {
                    id: "p1-t12".into(),
                    title: "Set up docker-compose for local dev".into(),
                    section_id: Some("s3".into()),
                    index: 1,
                    ..default_task()
                },
            ],
        },
    ];

    (sectionless, sections)
}

#[derive(Clone, PartialEq, Props)]
pub struct ProjectViewProps {
    project_id: String,
}

#[component]
pub fn ProjectView(props: ProjectViewProps) -> Element {
    let _project_id = &props.project_id;
    let (initial_sectionless, initial_sections) = sample_project_data();

    let mut sectionless_tasks = use_signal(|| initial_sectionless);
    let section_data: Signal<Vec<(String, String, Signal<Vec<Task>>)>> = use_signal(|| {
        initial_sections
            .into_iter()
            .map(|s| (s.id, s.title, Signal::new(s.tasks)))
            .collect()
    });

    let mut collapsed_sections: Signal<Vec<String>> = use_signal(|| Vec::new());

    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone().unwrap_or_default();

    rsx! {
        div { class: "task-list",
            // Sectionless tasks at top
            for task in sectionless_tasks.read().iter().cloned() {
                {
                    let task_id = task.id.clone();
                    let task_id_complete = task.id.clone();
                    let is_selected = task.id == selected_id;
                    let is_today = task.today_index.is_some();
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
                                let mut t = sectionless_tasks.write();
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
                    // placeholder
                },
            }

            // Sections
            for (section_id, section_title, section_tasks) in section_data.read().iter().cloned() {
                {
                    let sid = section_id.clone();
                    let sid_toggle = section_id.clone();
                    let is_collapsed = collapsed_sections.read().contains(&sid);
                    let task_count = section_tasks.read().len();

                    rsx! {
                        SectionHeader {
                            key: "{sid}",
                            title: section_title,
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
                            for task in section_tasks.read().iter().cloned() {
                                {
                                    let task_id = task.id.clone();
                                    let task_id_complete = task.id.clone();
                                    let is_selected = task.id == selected_id;
                                    let is_today = task.today_index.is_some();
                                    let mut section_tasks_signal = section_tasks;
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
                                                let mut t = section_tasks_signal.write();
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
                                    // placeholder
                                },
                            }
                        }
                    }
                }
            }
        }
    }
}
