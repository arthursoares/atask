use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::checkbox::Checkbox;
use crate::components::task_meta::TaskMeta;

#[derive(Clone, PartialEq, Props)]
pub struct TaskItemProps {
    task: Task,
    #[props(default = false)]
    selected: bool,
    #[props(default = false)]
    today_view: bool,
    #[props(default = true)]
    show_project: bool,
    #[props(default = false)]
    draggable: bool,
    #[props(default = false)]
    drag_over: bool,
    on_select: EventHandler<String>,
    on_complete: EventHandler<String>,
    #[props(default)]
    on_drag_start: EventHandler<String>,
    #[props(default)]
    on_drop_target: EventHandler<String>,
}

#[component]
pub fn TaskItem(props: TaskItemProps) -> Element {
    let is_completed = props.task.is_completed();
    let task_id = props.task.id.clone();
    let task_id_for_complete = task_id.clone();
    let task_id_for_drag = task_id.clone();
    let task_id_for_drop = task_id.clone();

    let mut item_classes = String::from("task-item");
    if props.selected {
        item_classes.push_str(" selected");
    }
    if props.drag_over {
        item_classes.push_str(" drag-over");
    }

    let title_class = if is_completed {
        "task-title completed"
    } else {
        "task-title"
    };

    let is_draggable = props.draggable;

    rsx! {
        div {
            class: item_classes,
            draggable: if is_draggable { "true" } else { "false" },
            onclick: move |_| props.on_select.call(task_id.clone()),
            ondragstart: move |_evt| {
                props.on_drag_start.call(task_id_for_drag.clone());
            },
            ondragover: move |evt| {
                evt.prevent_default();
            },
            ondrop: move |evt| {
                evt.prevent_default();
                props.on_drop_target.call(task_id_for_drop.clone());
            },
            Checkbox {
                checked: is_completed,
                today: props.today_view && !is_completed,
                on_toggle: move |_| props.on_complete.call(task_id_for_complete.clone()),
            }
            span { class: title_class, "{props.task.title}" }
            TaskMeta {
                task: props.task.clone(),
                show_project: props.show_project,
            }
        }
    }
}
