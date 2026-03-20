use dioxus::prelude::*;
use crate::api::types::Task;
use crate::components::checkbox::Checkbox;
use crate::components::task_meta::TaskMeta;

#[derive(Clone, PartialEq, Props)]
pub struct TaskItemProps {
    task: Task,
    selected: bool,
    today_view: bool,
    #[props(default = true)]
    show_project: bool,
    on_select: EventHandler<String>,
    on_complete: EventHandler<String>,
}

#[component]
pub fn TaskItem(props: TaskItemProps) -> Element {
    let class = {
        let mut c = "task-item".to_string();
        if props.selected {
            c.push_str(" selected");
        }
        c
    };

    let title_class = if props.task.is_completed() {
        "task-title completed"
    } else {
        "task-title"
    };

    let task_id = props.task.id.clone();
    let task_id_complete = props.task.id.clone();
    let show_today_badge = !props.today_view;

    rsx! {
        div {
            class: "{class}",
            onclick: move |_| props.on_select.call(task_id.clone()),

            Checkbox {
                checked: props.task.is_completed(),
                today: props.task.is_today() && props.today_view,
                on_toggle: move |_| props.on_complete.call(task_id_complete.clone()),
            }

            span { class: "{title_class}", "{props.task.title}" }

            TaskMeta {
                task: props.task.clone(),
                show_project: props.show_project && show_today_badge,
            }
        }
    }
}
