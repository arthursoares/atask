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
    let mut title_draft: Signal<String> = use_signal(|| String::new());
    let mut notes_draft: Signal<String> = use_signal(|| String::new());
    let mut checklist_input: Signal<String> = use_signal(|| String::new());

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

                            let task_schedule = task.schedule;

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

                            // Sync drafts with task data on task switch.
                            // Use peek() to avoid subscribing (which would loop).
                            {
                                let t = task.title.clone();
                                let n = task.notes.clone();
                                if title_draft.peek().is_empty() || title_draft.peek().as_str() != t {
                                    title_draft.set(t);
                                }
                                if notes_draft.peek().as_str() != n {
                                    notes_draft.set(n);
                                }
                            }

                            let title_value = title_draft.read().clone();
                            let notes_value = notes_draft.read().clone();

                            // Read signals inside rsx! for reactivity
                            let checklist_items: Vec<ChecklistItemData> = checklist.read().clone();
                            let activity_items: Vec<Activity> = activity.read().clone();
                            let checklist_input_value = checklist_input.read().clone();

                            let task_id_title_key = task.id.clone();
                            let task_id_title_blur = task.id.clone();
                            let task_id_notes = task.id.clone();
                            let task_id_schedule = task.id.clone();
                            let task_id_schedule2 = task.id.clone();
                            let task_id_schedule3 = task.id.clone();
                            let task_id_checklist = task.id.clone();

                            rsx! {
                                div { class: "detail-panel",
                                    div { class: "detail-header",
                                        div { class: "detail-close",
                                            onclick: move |_| selected_task_id.set(None),
                                            "\u{2715}"
                                        }
                                        // Editable title
                                        input {
                                            class: "input input-ghost detail-title-input",
                                            value: "{title_value}",
                                            oninput: move |e: Event<FormData>| {
                                                title_draft.set(e.value());
                                            },
                                            onkeydown: move |e: Event<KeyboardData>| {
                                                if e.key() == Key::Enter {
                                                    let val = title_draft.read().trim().to_string();
                                                    if !val.is_empty() {
                                                        let api_clone = api.read().clone();
                                                        let tid = task_id_title_key.clone();
                                                        spawn(async move {
                                                            if let Err(e) = api_clone.update_task_title(&tid, &val).await {
                                                                eprintln!("Failed to update title: {e}");
                                                            }
                                                        });
                                                    }
                                                }
                                            },
                                            onblur: move |_| {
                                                let val = title_draft.read().trim().to_string();
                                                if !val.is_empty() {
                                                    let api_clone = api.read().clone();
                                                    let tid = task_id_title_blur.clone();
                                                    spawn(async move {
                                                        if let Err(e) = api_clone.update_task_title(&tid, &val).await {
                                                            eprintln!("Failed to update title: {e}");
                                                        }
                                                    });
                                                }
                                            },
                                        }
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
                                        // SCHEDULE — interactive picker
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "SCHEDULE" }
                                            div { class: "schedule-picker",
                                                button {
                                                    class: if task_schedule == 0 { "schedule-option active" } else { "schedule-option" },
                                                    onclick: {
                                                        let tid = task_id_schedule.clone();
                                                        move |_| {
                                                            let api_clone = api.read().clone();
                                                            let tid = tid.clone();
                                                            spawn(async move {
                                                                if let Err(e) = api_clone.update_task_schedule(&tid, "inbox").await {
                                                                    eprintln!("Failed to update schedule: {e}");
                                                                }
                                                            });
                                                        }
                                                    },
                                                    "Inbox"
                                                }
                                                button {
                                                    class: if task_schedule == 1 { "schedule-option active" } else { "schedule-option" },
                                                    onclick: {
                                                        let tid = task_id_schedule2.clone();
                                                        move |_| {
                                                            let api_clone = api.read().clone();
                                                            let tid = tid.clone();
                                                            spawn(async move {
                                                                if let Err(e) = api_clone.update_task_schedule(&tid, "anytime").await {
                                                                    eprintln!("Failed to update schedule: {e}");
                                                                }
                                                            });
                                                        }
                                                    },
                                                    "Today"
                                                }
                                                button {
                                                    class: if task_schedule == 2 { "schedule-option active" } else { "schedule-option" },
                                                    onclick: {
                                                        let tid = task_id_schedule3.clone();
                                                        move |_| {
                                                            let api_clone = api.read().clone();
                                                            let tid = tid.clone();
                                                            spawn(async move {
                                                                if let Err(e) = api_clone.update_task_schedule(&tid, "someday").await {
                                                                    eprintln!("Failed to update schedule: {e}");
                                                                }
                                                            });
                                                        }
                                                    },
                                                    "Someday"
                                                }
                                            }
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
                                        // NOTES — editable textarea
                                        div { class: "detail-section",
                                            div { class: "detail-section-title", "NOTES" }
                                            textarea {
                                                class: "detail-notes-input",
                                                placeholder: "Add notes...",
                                                value: "{notes_value}",
                                                oninput: move |e: Event<FormData>| {
                                                    notes_draft.set(e.value());
                                                },
                                                onblur: move |_| {
                                                    let val = notes_draft.read().clone();
                                                    let api_clone = api.read().clone();
                                                    let tid = task_id_notes.clone();
                                                    spawn(async move {
                                                        if let Err(e) = api_clone.update_task_notes(&tid, &val).await {
                                                            eprintln!("Failed to update notes: {e}");
                                                        }
                                                    });
                                                },
                                            }
                                        }
                                        // CHECKLIST
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
                                            // Add checklist item input
                                            input {
                                                class: "input checklist-add-input",
                                                placeholder: "Add item...",
                                                value: "{checklist_input_value}",
                                                oninput: move |e: Event<FormData>| {
                                                    checklist_input.set(e.value());
                                                },
                                                onkeydown: move |e: Event<KeyboardData>| {
                                                    if e.key() == Key::Enter {
                                                        let val = checklist_input.read().trim().to_string();
                                                        if !val.is_empty() {
                                                            checklist_input.set(String::new());
                                                            let api_clone = api.read().clone();
                                                            let tid = task_id_checklist.clone();
                                                            spawn(async move {
                                                                match api_clone.add_checklist_item(&tid, &val).await {
                                                                    Ok(_) => {
                                                                        // Refresh checklist
                                                                        match api_clone.list_checklist(&tid).await {
                                                                            Ok(items) => checklist.set(items),
                                                                            Err(_) => {}
                                                                        }
                                                                    }
                                                                    Err(e) => {
                                                                        eprintln!("Failed to add checklist item: {e}");
                                                                    }
                                                                }
                                                            });
                                                        }
                                                    }
                                                },
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
