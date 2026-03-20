use dioxus::prelude::*;
use crate::state::app::TagList;

#[derive(Props, Clone, PartialEq)]
pub struct TagPickerProps {
    current_tags: Vec<String>,
    on_add: EventHandler<String>,
    on_remove: EventHandler<String>,
}

#[component]
pub fn TagPicker(props: TagPickerProps) -> Element {
    let tags: TagList = use_context();

    rsx! {
        div { class: "picker-dropdown",
            for tag in tags.0.read().iter() {
                {
                    let tag_id = tag.id.clone();
                    let tag_title = tag.title.clone();
                    let is_active = props.current_tags.contains(&tag.id);
                    rsx! {
                        div {
                            class: if is_active { "picker-item active" } else { "picker-item" },
                            onclick: {
                                let on_add = props.on_add.clone();
                                let on_remove = props.on_remove.clone();
                                let tag_id = tag_id.clone();
                                move |_| {
                                    if is_active {
                                        on_remove.call(tag_id.clone());
                                    } else {
                                        on_add.call(tag_id.clone());
                                    }
                                }
                            },
                            "{tag_title}"
                        }
                    }
                }
            }
        }
    }
}
