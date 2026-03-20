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

            // Grip handle (visible on hover via CSS)
            div { class: "task-grip",
                svg {
                    view_box: "0 0 16 16",
                    xmlns: "http://www.w3.org/2000/svg",
                    width: "12",
                    height: "12",
                    fill: "currentColor",
                    // 6-dot grip pattern
                    circle { cx: "5", cy: "3", r: "1.2" }
                    circle { cx: "11", cy: "3", r: "1.2" }
                    circle { cx: "5", cy: "8", r: "1.2" }
                    circle { cx: "11", cy: "8", r: "1.2" }
                    circle { cx: "5", cy: "13", r: "1.2" }
                    circle { cx: "11", cy: "13", r: "1.2" }
                }
            }

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
