use dioxus::prelude::*;

use crate::state::projects::ProjectState;

#[derive(Clone, PartialEq, Props)]
pub struct ProjectPickerProps {
    current_project_id: Option<String>,
    on_select: EventHandler<Option<String>>,
}

#[component]
pub fn ProjectPicker(props: ProjectPickerProps) -> Element {
    let project_state: Signal<ProjectState> = use_context();
    let mut search_query = use_signal(|| String::new());

    rsx! {
        div { class: "picker-dropdown",
            {
                let projects = project_state.read().projects.read().clone();
                let active_projects: Vec<_> = projects.into_iter().filter(|p| p.status == 0).collect();
                let show_search = active_projects.len() > 5;
                let query = search_query.read().to_lowercase();
                let filtered: Vec<_> = if query.is_empty() {
                    active_projects
                } else {
                    active_projects.into_iter().filter(|p| p.title.to_lowercase().contains(&query)).collect()
                };

                rsx! {
                    if show_search {
                        div { class: "picker-search",
                            input {
                                r#type: "text",
                                placeholder: "Filter projects...",
                                value: "{search_query}",
                                oninput: move |evt: Event<FormData>| {
                                    search_query.set(evt.value());
                                },
                            }
                        }
                    }

                    {
                        let none_class = if props.current_project_id.is_none() {
                            "picker-item active"
                        } else {
                            "picker-item"
                        };
                        let on_select_none = props.on_select.clone();
                        rsx! {
                            div {
                                class: none_class,
                                onclick: move |_| on_select_none.call(None),
                                span { class: "picker-item-label", "None" }
                            }
                        }
                    }

                    for project in filtered {
                        {
                            let pid = project.id.clone();
                            let is_active = props.current_project_id.as_ref() == Some(&pid);
                            let class = if is_active { "picker-item active" } else { "picker-item" };
                            let on_select = props.on_select.clone();
                            let select_id = pid.clone();
                            rsx! {
                                div {
                                    key: "{pid}",
                                    class,
                                    onclick: move |_| on_select.call(Some(select_id.clone())),
                                    span { class: "sidebar-project-dot" }
                                    span { "{project.title}" }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
