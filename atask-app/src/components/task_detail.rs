use std::collections::HashMap;
use dioxus::prelude::*;

use crate::api::types::{Task, ChecklistItem};
use crate::state::app::*;
use super::checklist_item::ChecklistItemComponent;
use super::schedule_picker::SchedulePicker;
use super::project_picker::ProjectPicker;
use super::date_picker::DatePicker;
use super::tag_picker::TagPicker;

fn find_task(
    id: &str,
    inbox: &[Task],
    today: &[Task],
    upcoming: &[Task],
    someday: &[Task],
    logbook: &[Task],
    project_tasks: &HashMap<String, Vec<Task>>,
) -> Option<Task> {
    for list in [inbox, today, upcoming, someday, logbook] {
        if let Some(t) = list.iter().find(|t| t.id == id) {
            return Some(t.clone());
        }
    }
    for tasks in project_tasks.values() {
        if let Some(t) = tasks.iter().find(|t| t.id == id) {
            return Some(t.clone());
        }
    }
    None
}

#[component]
pub fn TaskDetail() -> Element {
    let mut selected: SelectedTaskSignal = use_context();
    let api: ApiSignal = use_context();
    let inbox: InboxTasks = use_context();
    let today: TodayTasks = use_context();
    let upcoming: UpcomingTasks = use_context();
    let someday: SomedayTasks = use_context();
    let logbook: LogbookTasks = use_context();
    let project_tasks: ProjectTasks = use_context();
    let projects: ProjectList = use_context();
    let tags: TagList = use_context();

    let mut last_loaded_id: Signal<Option<String>> = use_signal(|| None);
    let mut title_draft: Signal<String> = use_signal(|| String::new());
    let mut notes_draft: Signal<String> = use_signal(|| String::new());
    let mut schedule_draft: Signal<i64> = use_signal(|| 0);
    let mut checklist: Signal<Vec<ChecklistItem>> = use_signal(|| Vec::new());
    let mut checklist_input: Signal<String> = use_signal(|| String::new());
    let mut show_project_picker: Signal<bool> = use_signal(|| false);
    let mut show_tag_picker: Signal<bool> = use_signal(|| false);

    // Fetch checklist when selected task changes
    use_effect(move || {
        let id = selected.0.read().clone();
        if let Some(ref tid) = id {
            if *last_loaded_id.read() != id {
                let api_clone = api.0.read().clone();
                let tid = tid.clone();
                spawn(async move {
                    match api_clone.list_checklist(&tid).await {
                        Ok(items) => checklist.set(items),
                        Err(_) => checklist.set(Vec::new()),
                    }
                });
                last_loaded_id.set(id);
                // Close pickers on task switch
                show_project_picker.set(false);
                show_tag_picker.set(false);
            }
        } else {
            checklist.set(Vec::new());
            last_loaded_id.set(None);
        }
    });

    rsx! {
        {
            let selected_id = selected.0.read().clone();
            match selected_id {
                None => rsx! {},
                Some(task_id) => {
                    let task = find_task(
                        &task_id,
                        &inbox.0.read(),
                        &today.0.read(),
                        &upcoming.0.read(),
                        &someday.0.read(),
                        &logbook.0.read(),
                        &project_tasks.0.read(),
                    );

                    match task {
                        None => rsx! {
                            div { class: "detail-panel",
                                p { "Task not found." }
                            }
                        },
                        Some(task) => {
                            // Init drafts on task switch
                            if *last_loaded_id.read() != Some(task_id.clone()) {
                                title_draft.set(task.title.clone());
                                notes_draft.set(task.notes.clone());
                                schedule_draft.set(task.schedule);
                            }

                            let task_schedule = *schedule_draft.read();
                            let project_name = task.project_id.as_ref().and_then(|pid| {
                                projects.0.read().iter().find(|p| p.id == *pid).map(|p| p.title.clone())
                            });

                            let task_tags: Vec<String> = task.tags.clone().unwrap_or_default();
                            let tag_pills: Vec<(String, String)> = task_tags.iter().filter_map(|tid| {
                                tags.0.read().iter().find(|t| t.id == *tid).map(|t| (t.id.clone(), t.title.clone()))
                            }).collect();

                            rsx! {
                                div { class: "detail-panel",
                                    // Close button
                                    div { class: "detail-close",
                                        onclick: move |_| selected.0.set(None),
                                        "\u{2715}"
                                    }

                                    // Title (ghost input)
                                    div { class: "detail-header",
                                        input {
                                            class: "input input-ghost detail-title-input",
                                            value: "{title_draft}",
                                            oninput: move |e: Event<FormData>| title_draft.set(e.value()),
                                            onkeydown: {
                                                let tid = task_id.clone();
                                                move |e: Event<KeyboardData>| {
                                                    if e.key() == Key::Enter {
                                                        e.prevent_default();
                                                        let title = title_draft.read().clone();
                                                        let api_clone = api.0.read().clone();
                                                        let tid = tid.clone();
                                                        println!("[DETAIL] Saving title: {title}");
                                                        spawn(async move {
                                                            match api_clone.update_task_title(&tid, &title).await {
                                                                Ok(_) => println!("[DETAIL] Title saved"),
                                                                Err(e) => println!("[DETAIL] Title save error: {e}"),
                                                            }
                                                        });
                                                    }
                                                }
                                            },
                                            onblur: {
                                                let tid = task_id.clone();
                                                move |_| {
                                                    let title = title_draft.read().clone();
                                                    let api_clone = api.0.read().clone();
                                                    let tid = tid.clone();
                                                    spawn(async move {
                                                        let _ = api_clone.update_task_title(&tid, &title).await;
                                                    });
                                                }
                                            },
                                        }
                                    }

                                    // Fields
                                    div { class: "detail-fields",
                                        // PROJECT
                                        div { class: "detail-field detail-field-picker",
                                            div { class: "detail-field-label", "PROJECT" }
                                            div {
                                                class: "detail-field-value",
                                                onclick: move |_| {
                                                    let current = *show_project_picker.read();
                                                    show_project_picker.set(!current);
                                                    show_tag_picker.set(false);
                                                },
                                                if let Some(ref name) = project_name {
                                                    "{name}"
                                                } else {
                                                    "None"
                                                }
                                            }
                                            if *show_project_picker.read() {
                                                ProjectPicker {
                                                    current_project_id: task.project_id.clone(),
                                                    on_select: {
                                                        let tid = task_id.clone();
                                                        move |project_id: Option<String>| {
                                                            show_project_picker.set(false);
                                                            let api_clone = api.0.read().clone();
                                                            let tid = tid.clone();
                                                            let pid = project_id.clone();
                                                            spawn(async move {
                                                                let _ = api_clone.move_task_to_project(&tid, pid.as_deref()).await;
                                                            });
                                                        }
                                                    },
                                                }
                                            }
                                        }

                                        // SCHEDULE
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "SCHEDULE" }
                                            SchedulePicker {
                                                current: task_schedule,
                                                on_change: {
                                                    let tid = task_id.clone();
                                                    move |schedule: String| {
                                                        println!("[DETAIL] Changing schedule to: {schedule}");
                                                        let new_val = match schedule.as_str() {
                                                            "inbox" => 0,
                                                            "anytime" => 1,
                                                            "someday" => 2,
                                                            _ => 0,
                                                        };
                                                        schedule_draft.set(new_val);
                                                        let api_clone = api.0.read().clone();
                                                        let tid = tid.clone();
                                                        let sched = schedule.clone();
                                                        spawn(async move {
                                                            match api_clone.update_task_schedule(&tid, &sched).await {
                                                                Ok(_) => println!("[DETAIL] Schedule saved"),
                                                                Err(e) => println!("[DETAIL] Schedule save error: {e}"),
                                                            }
                                                        });
                                                    }
                                                },
                                            }
                                        }

                                        // START DATE
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "START DATE" }
                                            DatePicker {
                                                value: task.start_date.clone(),
                                                on_change: {
                                                    let tid = task_id.clone();
                                                    move |date: Option<String>| {
                                                        let api_clone = api.0.read().clone();
                                                        let tid = tid.clone();
                                                        let d = date.clone();
                                                        spawn(async move {
                                                            let _ = api_clone.set_task_start_date(&tid, d.as_deref()).await;
                                                        });
                                                    }
                                                },
                                            }
                                        }

                                        // DEADLINE
                                        div { class: "detail-field",
                                            div { class: "detail-field-label", "DEADLINE" }
                                            DatePicker {
                                                value: task.deadline.clone(),
                                                on_change: {
                                                    let tid = task_id.clone();
                                                    move |date: Option<String>| {
                                                        let api_clone = api.0.read().clone();
                                                        let tid = tid.clone();
                                                        let d = date.clone();
                                                        spawn(async move {
                                                            let _ = api_clone.set_task_deadline(&tid, d.as_deref()).await;
                                                        });
                                                    }
                                                },
                                            }
                                        }

                                        // TAGS
                                        div { class: "detail-field detail-field-picker",
                                            div { class: "detail-field-label", "TAGS" }
                                            div { class: "detail-tags-row",
                                                for (tag_id, tag_title) in tag_pills.iter() {
                                                    span {
                                                        class: "tag tag-default",
                                                        key: "{tag_id}",
                                                        "{tag_title}"
                                                    }
                                                }
                                                span {
                                                    class: "detail-tag-add-btn",
                                                    onclick: move |_| {
                                                        let current = *show_tag_picker.read();
                                                        show_tag_picker.set(!current);
                                                        show_project_picker.set(false);
                                                    },
                                                    "+ Add"
                                                }
                                            }
                                            if *show_tag_picker.read() {
                                                TagPicker {
                                                    current_tags: task_tags.clone(),
                                                    on_add: {
                                                        let tid = task_id.clone();
                                                        move |tag_id: String| {
                                                            let api_clone = api.0.read().clone();
                                                            let tid = tid.clone();
                                                            let tag_id = tag_id.clone();
                                                            spawn(async move {
                                                                let _ = api_clone.add_task_tag(&tid, &tag_id).await;
                                                            });
                                                        }
                                                    },
                                                    on_remove: {
                                                        let tid = task_id.clone();
                                                        move |tag_id: String| {
                                                            let api_clone = api.0.read().clone();
                                                            let tid = tid.clone();
                                                            let tag_id = tag_id.clone();
                                                            spawn(async move {
                                                                let _ = api_clone.remove_task_tag(&tid, &tag_id).await;
                                                            });
                                                        }
                                                    },
                                                }
                                            }
                                        }
                                    }

                                    // NOTES
                                    div { class: "detail-section",
                                        div { class: "detail-section-title", "NOTES" }
                                        textarea {
                                            class: "detail-notes-input",
                                            value: "{notes_draft}",
                                            placeholder: "Add notes...",
                                            oninput: move |e: Event<FormData>| notes_draft.set(e.value()),
                                            onkeydown: {
                                                let tid = task_id.clone();
                                                move |e: Event<KeyboardData>| {
                                                    if e.modifiers().meta() && e.key() == Key::Enter {
                                                        e.prevent_default();
                                                        let notes = notes_draft.read().clone();
                                                        let api_clone = api.0.read().clone();
                                                        let tid = tid.clone();
                                                        println!("[DETAIL] Cmd+Enter: saving notes");
                                                        spawn(async move {
                                                            match api_clone.update_task_notes(&tid, &notes).await {
                                                                Ok(_) => println!("[DETAIL] Notes saved"),
                                                                Err(e) => println!("[DETAIL] Notes save error: {e}"),
                                                            }
                                                        });
                                                    }
                                                }
                                            },
                                            onblur: {
                                                let tid = task_id.clone();
                                                move |_| {
                                                    let notes = notes_draft.read().clone();
                                                    let api_clone = api.0.read().clone();
                                                    let tid = tid.clone();
                                                    println!("[DETAIL] Blur: saving notes");
                                                    spawn(async move {
                                                        match api_clone.update_task_notes(&tid, &notes).await {
                                                            Ok(_) => println!("[DETAIL] Notes saved via blur"),
                                                            Err(e) => println!("[DETAIL] Notes save error: {e}"),
                                                        }
                                                    });
                                                }
                                            },
                                        }
                                    }

                                    // CHECKLIST
                                    div { class: "detail-section",
                                        div { class: "detail-section-title", "CHECKLIST" }
                                        for item in checklist.read().iter() {
                                            {
                                                let item_id = item.id.clone();
                                                let is_checked = item.is_completed();
                                                rsx! {
                                                    ChecklistItemComponent {
                                                        title: item.title.clone(),
                                                        checked: is_checked,
                                                        on_toggle: {
                                                            let tid = task_id.clone();
                                                            let item_id = item_id.clone();
                                                            move |_| {
                                                                let api_clone = api.0.read().clone();
                                                                let tid = tid.clone();
                                                                let iid = item_id.clone();
                                                                let was_checked = is_checked;
                                                                spawn(async move {
                                                                    if was_checked {
                                                                        let _ = api_clone.uncomplete_checklist_item(&tid, &iid).await;
                                                                    } else {
                                                                        let _ = api_clone.complete_checklist_item(&tid, &iid).await;
                                                                    }
                                                                    if let Ok(items) = api_clone.list_checklist(&tid).await {
                                                                        checklist.set(items);
                                                                    }
                                                                });
                                                            }
                                                        },
                                                    }
                                                }
                                            }
                                        }
                                        // Add checklist item
                                        input {
                                            class: "checklist-add-input",
                                            placeholder: "Add item...",
                                            value: "{checklist_input}",
                                            oninput: move |e: Event<FormData>| checklist_input.set(e.value()),
                                            onkeydown: {
                                                let tid = task_id.clone();
                                                move |e: Event<KeyboardData>| {
                                                    if e.key() == Key::Enter {
                                                        let title = checklist_input.read().clone();
                                                        if !title.is_empty() {
                                                            checklist_input.set(String::new());
                                                            let api_clone = api.0.read().clone();
                                                            let tid = tid.clone();
                                                            spawn(async move {
                                                                let _ = api_clone.add_checklist_item(&tid, &title).await;
                                                                if let Ok(items) = api_clone.list_checklist(&tid).await {
                                                                    checklist.set(items);
                                                                }
                                                            });
                                                        }
                                                    }
                                                }
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
}
