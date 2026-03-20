use dioxus::prelude::*;

use crate::state::projects::ProjectState;

#[derive(Clone, PartialEq, Props)]
pub struct TagPickerProps {
    task_id: String,
    current_tags: Vec<String>,
    on_add: EventHandler<String>,
    on_remove: EventHandler<String>,
}

#[component]
pub fn TagPicker(props: TagPickerProps) -> Element {
    let project_state: Signal<ProjectState> = use_context();

    rsx! {
        div { class: "picker-dropdown",
            {
                let tags = project_state.read().tags.read().clone();

                rsx! {
                    for tag in tags {
                        {
                            let tag_id = tag.id.clone();
                            let is_active = props.current_tags.contains(&tag_id);
                            let class = if is_active { "picker-item active" } else { "picker-item" };
                            let on_add = props.on_add.clone();
                            let on_remove = props.on_remove.clone();
                            let click_id = tag_id.clone();
                            rsx! {
                                div {
                                    key: "{tag_id}",
                                    class,
                                    onclick: move |_| {
                                        if is_active {
                                            on_remove.call(click_id.clone());
                                        } else {
                                            on_add.call(click_id.clone());
                                        }
                                    },
                                    span { class: "tag tag-default", "{tag.title}" }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
