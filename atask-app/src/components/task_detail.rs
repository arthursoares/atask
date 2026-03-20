use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::{Activity, ChecklistItem as ChecklistItemData, Task};
use crate::components::checklist_item::ChecklistItem;
use crate::components::tag_pill::TagPill;
use crate::state::projects::ProjectState;
use crate::state::tasks::TaskState;

/// Find a task across all task state signals and project tasks.
fn find_task_in_state(
    task_state: &TaskState,
    project_state: &ProjectState,
    id: &str,
) -> Option<Task> {
    for tasks in [
        &task_state.today,
        &task_state.inbox,
        &task_state.upcoming,
        &task_state.someday,
        &task_state.logbook,
    ] {
        if let Some(t) = tasks.read().iter().find(|t| t.id == id) {
            return Some(t.clone());
        }
    }
    for tasks in project_state.project_tasks.read().values() {
        if let Some(t) = tasks.iter().find(|t| t.id == id) {
            return Some(t.clone());
        }
    }
    None
}

#[component]
pub fn TaskDetail() -> Element {
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let task_state: Signal<TaskState> = use_context();
    let project_state: Signal<ProjectState> = use_context();
    let api: Signal<ApiClient> = use_context();

    let mut checklist: Signal<Vec<ChecklistItemData>> = use_signal(|| Vec::new());
    let mut activity: Signal<Vec<Activity>> = use_signal(|| Vec::new());

    // Fetch checklist + activity when selected task changes.
    // We read selected_task_id inside the effect so Dioxus tracks it.
    let _data_loader = use_effect(move || {
        let selected_id = selected_task_id.read().clone();
        let Some(tid) = selected_id else {
            checklist.set(Vec::new());
            activity.set(Vec::new());
            return;
        };
        let api_clone = api.read().clone();
        spawn(async move {
            let (cl_result, act_result) = tokio::join!(
                api_clone.list_checklist(&tid),
                api_clone.list_activity(&tid),
            );
            match cl_result {
                Ok(items) => checklist.set(items),
                Err(_) => checklist.set(Vec::new()),
            }
            match act_result {
                Ok(items) => activity.set(items),
                Err(_) => activity.set(Vec::new()),
            }
        });
    });

    // All signal reads that drive rendering happen inside rsx! below.
    rsx! {
        {
            // Read selected_task_id inside rsx! for reactivity
            let selected_id = selected_task_id.read().clone();
            match selected_id {
                None => rsx! {},
                Some(task_id) => {
                    // Read state signals inside rsx! so component re-renders on changes
                    let task = find_task_in_state(&task_state.read(), &project_state.read(), &task_id);
                    match task {
                        None => rsx! {
                            div { class: "detail-panel",
                                div { class: "detail-header",
                                    div { class: "detail-close",
                                        onclick: move |_| selected_task_id.set(None),
                                        "\u{2715}"
                                    }
                                    div { class: "detail-title", "Task not found" }
                                }
                            }
                        },
                        Some(task) => {
                            let project_name = task.project_id.as_ref().and_then(|pid| {
                                project_state
                                    .read()
                                    .projects
                                    .read()
                                    .iter()
                                    .find(|p| p.id == *pid)
                                    .map(|p| p.title.clone())
                            });

                            let schedule_label = match (task.is_today(), task.schedule_name()) {
                                (true, name) => format!("Today ({name})"),
                                (false, name) => name.to_string(),
                            };

                            let start_date_label = task
                                .start_date
                                .as_deref()
                                .unwrap_or("None")
                                .to_string();

                            let deadline_label = task
                                .deadline
                                .as_deref()
                                .unwrap_or("None")
                                .to_string();

                            let tags = task.tags.clone().unwrap_or_default();

                            // Read signals inside rsx! for reactivity
                            let checklist_items: Vec<ChecklistItemData> = checklist.read().clone();
                            let activity_items: Vec<Activity> = activity.read().clone();

                            rsx! {
                                div { class: "detail-panel",
                                    div { class: "detail-header",
                                        div { class: "detail-close",
                                            onclick: move |_| selected_task_id.set(None),
                                            "\u{2715}"
                                        }
                                        div { class: "detail-title", "{task.title}" }
                                        div { class: "detail-meta-row",
                                            if task.is_today() {
                                                TagPill { label: "\u{2605} Today".to_string(), variant: "today".to_string() }
                                            }
                                            for tag in &tags {
                                                TagPill { label: tag.clone(), variant: "default".to_string() }
                                            }
                                        }
                                    }
                                    div { class: "detail-body",
                                        // PROJECT
                                        if let Some(ref pname) = project_name {
                                            div { class: "detail-field",
                                                div { class: "detail-field-label", "PROJECT" }
                                                div { class: "detail-field-value",
                                                    span { class: "detail-project-dot" }
                                                    "\u{25cf} {pname}"
                                                }
                                            }
                                        }
                                        // SCHEDULE
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "SCHEDULE" }
                                            div { class: "detail-field-value", "{schedule_label}" }
                                        }
                                        // START DATE
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "START DATE" }
                                            div { class: "detail-field-value", "{start_date_label}" }
                                        }
                                        // DEADLINE
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "DEADLINE" }
                                            div { class: "detail-field-value", "{deadline_label}" }
                                        }
                                        // TAGS
                                        if !tags.is_empty() {
                                            div { class: "detail-field",
                                                div { class: "detail-field-label", "TAGS" }
                                                div { class: "detail-field-value detail-tags-row",
                                                    for tag in &tags {
                                                        TagPill { label: tag.clone(), variant: "default".to_string() }
                                                    }
                                                }
                                            }
                                        }
                                        // NOTES
                                        if !task.notes.is_empty() {
                                            div { class: "detail-section",
                                                div { class: "detail-section-title", "NOTES" }
                                                div { class: "detail-section-content", "{task.notes}" }
                                            }
                                        }
                                        // CHECKLIST
                                        if !checklist_items.is_empty() {
                                            div { class: "detail-section",
                                                div { class: "detail-section-title", "CHECKLIST" }
                                                for item in checklist_items {
                                                    {
                                                        let item_id = item.id.clone();
                                                        let item_task_id = item.task_id.clone();
                                                        let is_checked = item.is_completed();
                                                        rsx! {
                                                            ChecklistItem {
                                                                key: "{item_id}",
                                                                title: item.title.clone(),
                                                                checked: is_checked,
                                                                on_toggle: move |_| {
                                                                    let api_clone = api.read().clone();
                                                                    let tid = item_task_id.clone();
                                                                    let iid = item_id.clone();
                                                                    let was_checked = is_checked;
                                                                    spawn(async move {
                                                                        let result = if was_checked {
                                                                            api_clone.uncomplete_checklist_item(&tid, &iid).await
                                                                        } else {
                                                                            api_clone.complete_checklist_item(&tid, &iid).await
                                                                        };
                                                                        if let Err(e) = result {
                                                                            eprintln!("Failed to toggle checklist item: {e}");
                                                                        }
                                                                        // Refresh checklist
                                                                        match api_clone.list_checklist(&tid).await {
                                                                            Ok(items) => checklist.set(items),
                                                                            Err(_) => {}
                                                                        }
                                                                    });
                                                                },
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                        // ACTIVITY
                                        if !activity_items.is_empty() {
                                            div { class: "detail-section",
                                                div { class: "detail-section-title", "ACTIVITY" }
                                                for entry in &activity_items {
                                                    div { class: "detail-activity-item",
                                                        span { class: "detail-activity-type", "{entry.activity_type}" }
                                                        if !entry.content.is_empty() {
                                                            span { class: "detail-activity-content", " \u{2014} {entry.content}" }
                                                        }
                                                        span { class: "detail-activity-date", "{entry.created_at}" }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
