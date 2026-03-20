use dioxus::prelude::*;
use crate::api::types::Task;
use crate::state::app::ProjectList;
use crate::state::date_fmt;

#[derive(Clone, PartialEq, Props)]
pub struct TaskMetaProps {
    task: Task,
    show_project: bool,
}

#[component]
pub fn TaskMeta(props: TaskMetaProps) -> Element {
    let projects: ProjectList = use_context();

    rsx! {
        div { class: "task-meta",
            // Project pill
            if props.show_project {
                if let Some(ref project_id) = props.task.project_id {
                    {
                        let project_list = projects.0.read();
                        let project = project_list.iter().find(|p| &p.id == project_id);
                        if let Some(project) = project {
                            let color = if project.color.is_empty() {
                                "var(--accent)".to_string()
                            } else {
                                project.color.clone()
                            };
                            rsx! {
                                span { class: "task-project-pill",
                                    span {
                                        class: "sidebar-project-dot",
                                        style: "background: {color};",
                                    }
                                    "{project.title}"
                                }
                            }
                        } else {
                            rsx! {}
                        }
                    }
                }
            }

            // Deadline pill
            if let Some(ref deadline) = props.task.deadline {
                {
                    let (label, variant) = date_fmt::format_deadline(deadline);
                    let class = match variant {
                        "overdue" => "tag tag-overdue",
                        "today" => "tag tag-deadline",
                        _ => "tag tag-default",
                    };
                    rsx! {
                        span { class: "{class}", "{label}" }
                    }
                }
            }

            // Today badge (only if task is today and not in today view context)
            if props.task.is_today() {
                span { class: "tag tag-today", "\u{2605} Today" }
            }
        }
    }
}
