use std::collections::BTreeMap;
use dioxus::prelude::*;
use crate::state::app::{UpcomingTasks, ApiSignal, SelectedTaskSignal};
use crate::state::date_fmt::format_section_date;
use crate::components::task_item::TaskItem;
use crate::components::section_header::SectionHeader;

#[component]
pub fn UpcomingView() -> Element {
    let api: ApiSignal = use_context();
    let mut upcoming: UpcomingTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();
    let mut collapsed: Signal<std::collections::HashSet<String>> = use_signal(|| std::collections::HashSet::new());

    rsx! {
        div { class: "view-content",
            {
                let tasks = upcoming.0.read().clone();
                if tasks.is_empty() {
                    rsx! {
                        div { class: "empty-state",
                            p { class: "empty-state-text", "Nothing scheduled ahead." }
                        }
                    }
                } else {
                    // Group tasks by start_date
                    let mut groups: BTreeMap<String, Vec<crate::api::types::Task>> = BTreeMap::new();
                    for task in tasks.iter() {
                        let key = task.start_date.clone().unwrap_or_default();
                        groups.entry(key).or_default().push(task.clone());
                    }

                    let collapsed_set = collapsed.read().clone();

                    rsx! {
                        for (date_key, group_tasks) in groups.iter() {
                            {
                                let date_key = date_key.clone();
                                let label = format_section_date(&date_key);
                                let count = group_tasks.len();
                                let is_collapsed = collapsed_set.contains(&date_key);

                                rsx! {
                                    SectionHeader {
                                        title: label,
                                        count: count,
                                        collapsed: is_collapsed,
                                        on_toggle: {
                                            let dk = date_key.clone();
                                            move |_| {
                                                let mut set = collapsed.read().clone();
                                                if set.contains(&dk) {
                                                    set.remove(&dk);
                                                } else {
                                                    set.insert(dk.clone());
                                                }
                                                collapsed.set(set);
                                            }
                                        },
                                    }

                                    if !is_collapsed {
                                        div { class: "task-list",
                                            for task in group_tasks.iter() {
                                                {
                                                    let task_id = task.id.clone();
                                                    let is_selected = *selected.0.read() == Some(task_id.clone());
                                                    rsx! {
                                                        TaskItem {
                                                            key: "{task_id}",
                                                            task: task.clone(),
                                                            selected: is_selected,
                                                            today_view: false,
                                                            show_project: true,
                                                            on_select: move |id: String| {
                                                                selected.0.set(Some(id));
                                                            },
                                                            on_complete: {
                                                                let task_id = task_id.clone();
                                                                move |_id: String| {
                                                                    let mut tasks = upcoming.0.read().clone();
                                                                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                                                                        t.status = 1;
                                                                    }
                                                                    upcoming.0.set(tasks);

                                                                    let api_clone = api.0.read().clone();
                                                                    let tid = task_id.clone();
                                                                    spawn(async move {
                                                                        let _ = api_clone.complete_task(&tid).await;
                                                                        if let Ok(fresh) = api_clone.list_upcoming().await {
                                                                            upcoming.0.set(fresh);
                                                                        }
                                                                    });
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
            }
        }
    }
}
