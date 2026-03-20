use dioxus::prelude::*;

use crate::api::types::Task;
use crate::components::tag_pill::TagPill;

#[derive(Clone, PartialEq, Props)]
pub struct TaskMetaProps {
    task: Task,
    #[props(default = true)]
    show_project: bool,
}

/// Format a deadline date string (e.g. "2026-03-25") into a short display like "Due Mar 25".
fn format_deadline(deadline: &str) -> String {
    // Attempt to parse YYYY-MM-DD; fall back to raw string.
    if deadline.len() >= 10 {
        let month = match &deadline[5..7] {
            "01" => "Jan",
            "02" => "Feb",
            "03" => "Mar",
            "04" => "Apr",
            "05" => "May",
            "06" => "Jun",
            "07" => "Jul",
            "08" => "Aug",
            "09" => "Sep",
            "10" => "Oct",
            "11" => "Nov",
            "12" => "Dec",
            _ => return format!("Due {deadline}"),
        };
        let day = deadline[8..10].trim_start_matches('0');
        format!("Due {month} {day}")
    } else {
        format!("Due {deadline}")
    }
}

#[component]
pub fn TaskMeta(props: TaskMetaProps) -> Element {
    let mut items: Vec<Element> = Vec::new();

    // Project pill
    if props.show_project {
        if let Some(ref _project_id) = props.task.project_id {
            // Hardcoded lookup for now
            let project_name = "Project";
            items.push(rsx! {
                span { class: "task-project-pill", "{project_name}" }
            });
        }
    }

    // Deadline
    if let Some(ref deadline) = props.task.deadline {
        let label = format_deadline(deadline);
        items.push(rsx! {
            TagPill { label, variant: "deadline" }
        });
    }

    // Today badge
    if props.task.is_today() {
        items.push(rsx! {
            TagPill { label: "★ Today", variant: "today" }
        });
    }

    // Limit to 3 items
    items.truncate(3);

    if items.is_empty() {
        return rsx! {};
    }

    rsx! {
        div { class: "task-meta",
            for (i, item) in items.into_iter().enumerate() {
                if i > 0 {
                    span { class: "task-meta-separator", "·" }
                }
                {item}
            }
        }
    }
}
