use dioxus::prelude::*;
use crate::state::app::*;

#[derive(Clone)]
struct Cmd {
    id: &'static str,
    label: &'static str,
    shortcut: &'static str,
    category: &'static str,
    requires_task: bool,
}

const COMMANDS: &[Cmd] = &[
    Cmd { id: "nav-inbox", label: "Go to Inbox", shortcut: "\u{2318}1", category: "Navigation", requires_task: false },
    Cmd { id: "nav-today", label: "Go to Today", shortcut: "\u{2318}2", category: "Navigation", requires_task: false },
    Cmd { id: "nav-upcoming", label: "Go to Upcoming", shortcut: "\u{2318}3", category: "Navigation", requires_task: false },
    Cmd { id: "nav-someday", label: "Go to Someday", shortcut: "\u{2318}4", category: "Navigation", requires_task: false },
    Cmd { id: "nav-logbook", label: "Go to Logbook", shortcut: "\u{2318}5", category: "Navigation", requires_task: false },
    Cmd { id: "new-task", label: "New Task", shortcut: "\u{2318}N", category: "Creation", requires_task: false },
    Cmd { id: "complete", label: "Complete Task", shortcut: "\u{2318}\u{21E7}C", category: "Actions", requires_task: true },
    Cmd { id: "schedule-today", label: "Schedule for Today", shortcut: "\u{2318}T", category: "Actions", requires_task: true },
    Cmd { id: "defer-someday", label: "Defer to Someday", shortcut: "", category: "Actions", requires_task: true },
    Cmd { id: "move-inbox", label: "Move to Inbox", shortcut: "", category: "Actions", requires_task: true },
    Cmd { id: "delete", label: "Delete Task", shortcut: "\u{232B}", category: "Actions", requires_task: true },
];

fn filtered_commands(query: &str, has_task: bool) -> Vec<&'static Cmd> {
    let q = query.to_lowercase();
    COMMANDS
        .iter()
        .filter(|cmd| {
            if cmd.requires_task && !has_task {
                return false;
            }
            if q.is_empty() {
                return true;
            }
            cmd.label.to_lowercase().contains(&q)
        })
        .collect()
}

fn execute_command(
    id: &str,
    mut active_view: ViewSignal,
    selected_task: SelectedTaskSignal,
    api: ApiSignal,
    mut open: CommandOpen,
    mut query: CommandQuery,
    mut index: CommandIndex,
) {
    match id {
        "nav-inbox" => active_view.0.set(ActiveView::Inbox),
        "nav-today" => active_view.0.set(ActiveView::Today),
        "nav-upcoming" => active_view.0.set(ActiveView::Upcoming),
        "nav-someday" => active_view.0.set(ActiveView::Someday),
        "nav-logbook" => active_view.0.set(ActiveView::Logbook),
        "new-task" => {
            let api_clone = api.0.read().clone();
            spawn(async move {
                let _ = api_clone.create_task("New task").await;
            });
        }
        "complete" => {
            if let Some(ref tid) = *selected_task.0.read() {
                let api_clone = api.0.read().clone();
                let tid = tid.clone();
                spawn(async move { let _ = api_clone.complete_task(&tid).await; });
            }
        }
        "schedule-today" => {
            if let Some(ref tid) = *selected_task.0.read() {
                let api_clone = api.0.read().clone();
                let tid = tid.clone();
                spawn(async move { let _ = api_clone.update_task_schedule(&tid, "anytime").await; });
            }
        }
        "defer-someday" => {
            if let Some(ref tid) = *selected_task.0.read() {
                let api_clone = api.0.read().clone();
                let tid = tid.clone();
                spawn(async move { let _ = api_clone.update_task_schedule(&tid, "someday").await; });
            }
        }
        "move-inbox" => {
            if let Some(ref tid) = *selected_task.0.read() {
                let api_clone = api.0.read().clone();
                let tid = tid.clone();
                spawn(async move { let _ = api_clone.update_task_schedule(&tid, "inbox").await; });
            }
        }
        "delete" => {
            if let Some(ref tid) = *selected_task.0.read() {
                let api_clone = api.0.read().clone();
                let tid = tid.clone();
                spawn(async move { let _ = api_clone.delete_task(&tid).await; });
            }
        }
        _ => {}
    }
    open.0.set(false);
    query.0.set(String::new());
    index.0.set(0);
}

#[component]
pub fn CommandPalette() -> Element {
    let mut open: CommandOpen = use_context();
    let mut query: CommandQuery = use_context();
    let mut index: CommandIndex = use_context();
    let selected_task: SelectedTaskSignal = use_context();
    let active_view: ViewSignal = use_context();
    let api: ApiSignal = use_context();

    rsx! {
        div {
            class: "command-backdrop",
            onclick: move |_| {
                open.0.set(false);
                query.0.set(String::new());
                index.0.set(0);
            },
            // Stop click propagation from palette to backdrop
            div {
                class: "command-palette",
                onclick: move |evt: Event<MouseData>| {
                    evt.stop_propagation();
                },
                div {
                    class: "command-input-wrap",
                    input {
                        class: "command-input",
                        r#type: "text",
                        placeholder: "Type a command\u{2026}",
                        autofocus: true,
                        value: "{query.0.read()}",
                        oninput: move |evt: Event<FormData>| {
                            query.0.set(evt.value().clone());
                            index.0.set(0);
                        },
                        onkeydown: move |evt: Event<KeyboardData>| {
                            let key = evt.key();
                            match key {
                                Key::ArrowUp => {
                                    evt.prevent_default();
                                    let cur = *index.0.read();
                                    if cur > 0 {
                                        index.0.set(cur - 1);
                                    }
                                }
                                Key::ArrowDown => {
                                    evt.prevent_default();
                                    let q = query.0.read().clone();
                                    let has_task = selected_task.0.read().is_some();
                                    let filtered = filtered_commands(&q, has_task);
                                    let cur = *index.0.read();
                                    if !filtered.is_empty() && cur < filtered.len() - 1 {
                                        index.0.set(cur + 1);
                                    }
                                }
                                Key::Enter => {
                                    evt.prevent_default();
                                    let q = query.0.read().clone();
                                    let has_task = selected_task.0.read().is_some();
                                    let filtered = filtered_commands(&q, has_task);
                                    let cur = *index.0.read();
                                    if let Some(cmd) = filtered.get(cur) {
                                        execute_command(cmd.id, active_view, selected_task, api, open, query, index);
                                    }
                                }
                                Key::Escape => {
                                    evt.prevent_default();
                                    open.0.set(false);
                                    query.0.set(String::new());
                                    index.0.set(0);
                                }
                                _ => {}
                            }
                        },
                    }
                }
                div {
                    class: "command-results",
                    {
                        let q = query.0.read().clone();
                        let has_task = selected_task.0.read().is_some();
                        let filtered = filtered_commands(&q, has_task);
                        let current_index = *index.0.read();

                        // Group by category, preserving order
                        let mut categories: Vec<(&str, Vec<(usize, &Cmd)>)> = Vec::new();
                        let mut global_idx = 0usize;
                        for cmd in &filtered {
                            let cat = cmd.category;
                            if let Some(group) = categories.iter_mut().find(|(c, _)| *c == cat) {
                                group.1.push((global_idx, cmd));
                            } else {
                                categories.push((cat, vec![(global_idx, cmd)]));
                            }
                            global_idx += 1;
                        }

                        rsx! {
                            for (cat, items) in categories {
                                div {
                                    key: "{cat}",
                                    div { class: "command-group-label", "{cat}" }
                                    for (idx, cmd) in items {
                                        {
                                            let is_active = idx == current_index;
                                            let cmd_id = cmd.id;
                                            let class_str = if is_active {
                                                "command-item active"
                                            } else {
                                                "command-item"
                                            };
                                            rsx! {
                                                div {
                                                    key: "{cmd_id}",
                                                    class: "{class_str}",
                                                    onclick: move |_| {
                                                        execute_command(cmd_id, active_view, selected_task, api, open, query, index);
                                                    },
                                                    span { "{cmd.label}" }
                                                    if !cmd.shortcut.is_empty() {
                                                        span { class: "command-shortcut", "{cmd.shortcut}" }
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
