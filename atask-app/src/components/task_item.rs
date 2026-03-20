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
    on_select: EventHandler<String>,
    on_complete: EventHandler<String>,
}

#[component]
pub fn TaskItem(props: TaskItemProps) -> Element {
    let is_completed = props.task.is_completed();
    let task_id = props.task.id.clone();
    let task_id_for_complete = task_id.clone();

    let item_class = if props.selected {
        "task-item selected"
    } else {
        "task-item"
    };

    let title_class = if is_completed {
        "task-title completed"
    } else {
        "task-title"
    };

    rsx! {
        div {
            class: item_class,
            onclick: move |_| props.on_select.call(task_id.clone()),
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
