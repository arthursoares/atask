use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::api::types::{ChecklistItem as ChecklistItemData, Task};
use crate::components::checklist_item::ChecklistItem;
use crate::components::tag_pill::TagPill;
use crate::state::tasks::TaskState;
use crate::state::projects::ProjectState;

/// Find a task across all task state signals.
fn find_task_in_state(task_state: &TaskState, project_state: &ProjectState, id: &str) -> Option<Task> {
    // Check each view's tasks
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
    // Check project tasks
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

    let selected_id = selected_task_id.read().clone();
    let Some(task_id) = selected_id else {
        return rsx! {};
    };

    let task = find_task_in_state(&task_state.read(), &project_state.read(), &task_id);

    let Some(task) = task else {
        return rsx! {
            div { class: "detail-panel",
                div { class: "detail-header",
                    div { class: "detail-close",
                        onclick: move |_| selected_task_id.set(None),
                        "\u{2715}"
                    }
                    div { class: "detail-title", "Task not found" }
                }
            }
        };
    };

    // Fetch checklist when task changes
    let mut checklist: Signal<Vec<ChecklistItemData>> = use_signal(|| Vec::new());
    let tid = task_id.clone();
    let _checklist_loader = use_effect(move || {
        let api_clone = api.read().clone();
        let tid = tid.clone();
        spawn(async move {
            match api_clone.list_checklist(&tid).await {
                Ok(items) => checklist.set(items),
                Err(_) => checklist.set(Vec::new()),
            }
        });
    });

    // Look up project name
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

    let checklist_items: Vec<ChecklistItemData> = checklist.read().clone();

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
            }
        }
    }
}
