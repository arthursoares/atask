use dioxus::prelude::*;

use crate::components::activity_entry::ActivityEntry;
use crate::components::checklist_item::ChecklistItem;
use crate::components::tag_pill::TagPill;

/// Hardcoded detail data for the sample task.
struct TaskDetailData {
    title: &'static str,
    project: &'static str,
    schedule: &'static str,
    start_date: &'static str,
    deadline: &'static str,
    tags: Vec<&'static str>,
    notes: &'static str,
    checklist: Vec<(&'static str, bool)>,
    activities: Vec<ActivityData>,
}

struct ActivityData {
    author: &'static str,
    is_agent: bool,
    timestamp: &'static str,
    content: &'static str,
}

fn get_sample_detail() -> TaskDetailData {
    TaskDetailData {
        title: "Design component library for macOS client",
        project: "atask v0",
        schedule: "Today (Anytime)",
        start_date: "Mar 20, 2026",
        deadline: "None",
        tags: vec!["Design", "Client"],
        notes: "Create a comprehensive design library inspired by Things 3's visual language. Define color tokens, typography scale, spacing system, and all core components needed for the native macOS client.",
        checklist: vec![
            ("Color palette", true),
            ("Typography scale", true),
            ("Core components", true),
            ("App mockup", true),
            ("All view screens", false),
            ("Command palette", false),
        ],
        activities: vec![
            ActivityData {
                author: "Claude",
                is_agent: true,
                timestamp: "5 min ago",
                content: "Decomposed into 6 checklist items based on the design spec scope.",
            },
            ActivityData {
                author: "Arthur",
                is_agent: false,
                timestamp: "20 min ago",
                content: "Decided on Dioxus over SwiftUI. Cross-platform wins.",
            },
        ],
    }
}

#[component]
pub fn TaskDetail() -> Element {
    let mut selected_task_id: Signal<Option<String>> = use_context();
    let selected_id = selected_task_id.read().clone();

    let Some(task_id) = selected_id else {
        return rsx! {};
    };

    // Only task "1" has full detail data; others show a minimal view
    let has_detail = task_id == "1";

    if has_detail {
        let detail = get_sample_detail();
        let mut checklist_state = use_signal(|| {
            vec![true, true, true, true, false, false]
        });

        rsx! {
            div { class: "detail-panel",
                div { class: "detail-header",
                    div { class: "detail-close",
                        onclick: move |_| {
                            selected_task_id.set(None);
                        },
                        "\u{2715}"
                    }
                    div { class: "detail-title", "{detail.title}" }
                    div { class: "detail-meta-row",
                        TagPill { label: "\u{2605} Today".to_string(), variant: "today".to_string() }
                        TagPill { label: "Design".to_string(), variant: "default".to_string() }
                    }
                }
                div { class: "detail-body",
                    // PROJECT
                    div { class: "detail-field",
                        div { class: "detail-field-label", "PROJECT" }
                        div { class: "detail-field-value",
                            span { class: "detail-project-dot" }
                            "\u{25cf} {detail.project}"
                        }
                    }
                    // SCHEDULE
                    div { class: "detail-field",
                        div { class: "detail-field-label", "SCHEDULE" }
                        div { class: "detail-field-value", "{detail.schedule}" }
                    }
                    // START DATE
                    div { class: "detail-field",
                        div { class: "detail-field-label", "START DATE" }
                        div { class: "detail-field-value", "{detail.start_date}" }
                    }
                    // DEADLINE
                    div { class: "detail-field",
                        div { class: "detail-field-label", "DEADLINE" }
                        div { class: "detail-field-value", "{detail.deadline}" }
                    }
                    // TAGS
                    div { class: "detail-field",
                        div { class: "detail-field-label", "TAGS" }
                        div { class: "detail-field-value detail-tags-row",
                            for tag in &detail.tags {
                                TagPill { label: tag.to_string(), variant: "default".to_string() }
                            }
                            span { class: "detail-add-tag", "+ Add" }
                        }
                    }
                    // NOTES
                    div { class: "detail-section",
                        div { class: "detail-section-title", "NOTES" }
                        div { class: "detail-section-content", "{detail.notes}" }
                    }
                    // CHECKLIST
                    div { class: "detail-section",
                        div { class: "detail-section-title", "CHECKLIST" }
                        for (i, (title, _checked)) in detail.checklist.iter().enumerate() {
                            {
                                let idx = i;
                                let is_checked = checklist_state.read()[idx];
                                rsx! {
                                    ChecklistItem {
                                        key: "{idx}",
                                        title: title.to_string(),
                                        checked: is_checked,
                                        on_toggle: move |_| {
                                            let mut state = checklist_state.write();
                                            state[idx] = !state[idx];
                                        },
                                    }
                                }
                            }
                        }
                    }
                    // ACTIVITY
                    div { class: "detail-section",
                        div { class: "detail-section-title", "ACTIVITY" }
                        div { class: "activity-stream",
                            for (i, activity) in detail.activities.iter().enumerate() {
                                ActivityEntry {
                                    key: "{i}",
                                    author: activity.author.to_string(),
                                    is_agent: activity.is_agent,
                                    timestamp: activity.timestamp.to_string(),
                                    content: activity.content.to_string(),
                                }
                            }
                        }
                    }
                }
            }
        }
    } else {
        // Minimal detail for tasks without full sample data
        rsx! {
            div { class: "detail-panel",
                div { class: "detail-header",
                    div { class: "detail-close",
                        onclick: move |_| {
                            selected_task_id.set(None);
                        },
                        "\u{2715}"
                    }
                    div { class: "detail-title", "Task {task_id}" }
                }
                div { class: "detail-body",
                    div { class: "empty-state",
                        p { class: "empty-state-text", "Full details will load from the API." }
                    }
                }
            }
        }
    }
}
