use dioxus::prelude::*;
use crate::state::app::ProjectList;

#[derive(Props, Clone, PartialEq)]
pub struct ProjectPickerProps {
    current_project_id: Option<String>,
    on_select: EventHandler<Option<String>>,
}

#[component]
pub fn ProjectPicker(props: ProjectPickerProps) -> Element {
    let projects: ProjectList = use_context();

    rsx! {
        div { class: "picker-dropdown",
            // "None" option at top
            div {
                class: if props.current_project_id.is_none() { "picker-item active" } else { "picker-item" },
                onclick: {
                    let on_select = props.on_select.clone();
                    move |_| on_select.call(None)
                },
                "No Project"
            }

            for project in projects.0.read().iter() {
                {
                    let pid = project.id.clone();
                    let is_active = props.current_project_id.as_ref() == Some(&project.id);
                    let color = project.color.clone();
                    let title = project.title.clone();
                    rsx! {
                        div {
                            class: if is_active { "picker-item active" } else { "picker-item" },
                            onclick: {
                                let on_select = props.on_select.clone();
                                let pid = pid.clone();
                                move |_| on_select.call(Some(pid.clone()))
                            },
                            span {
                                class: "sidebar-project-dot",
                                style: "background: {color}",
                            }
                            "{title}"
                        }
                    }
                }
            }
        }
    }
}
