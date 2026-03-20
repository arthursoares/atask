use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::state::command::{
    all_commands, command_to_view, filter_commands, CommandCategory, CommandState,
};
use crate::state::navigation::ActiveView;
use crate::state::tasks::TaskState;

#[component]
pub fn CommandPalette() -> Element {
    let mut cmd_state: Signal<CommandState> = use_context();
    let mut active_view: Signal<ActiveView> = use_context();
    let selected_task_id: Signal<Option<String>> = use_context();
    let api: Signal<ApiClient> = use_context();
    let task_state: Signal<TaskState> = use_context();

    let has_selected = selected_task_id.read().is_some();
    let commands = all_commands(has_selected);
    let query_val = cmd_state.read().query.read().clone();
    let filtered = filter_commands(&commands, &query_val);

    // Clamp selected_index
    let max_idx = if filtered.is_empty() {
        0
    } else {
        filtered.len() - 1
    };
    let sel_idx = {
        let idx = *cmd_state.read().selected_index.read();
        if idx > max_idx { 0 } else { idx }
    };

    // Group filtered commands by category, preserving order
    let groups = group_by_category(&filtered);

    rsx! {
        div {
            class: "command-backdrop",
            onclick: move |_| {
                cmd_state.write().open.set(false);
                cmd_state.write().query.set(String::new());
                cmd_state.write().selected_index.set(0);
            },
        }
        div {
            class: "command-palette",
            onkeydown: move |evt: Event<KeyboardData>| {
                let key = evt.key();
                match key {
                    Key::Escape => {
                        evt.prevent_default();
                        cmd_state.write().open.set(false);
                        cmd_state.write().query.set(String::new());
                        cmd_state.write().selected_index.set(0);
                    }
                    Key::ArrowDown => {
                        evt.prevent_default();
                        let current = *cmd_state.read().selected_index.read();
                        let next = if current >= max_idx { 0 } else { current + 1 };
                        cmd_state.write().selected_index.set(next);
                    }
                    Key::ArrowUp => {
                        evt.prevent_default();
                        let current = *cmd_state.read().selected_index.read();
                        let next = if current == 0 { max_idx } else { current - 1 };
                        cmd_state.write().selected_index.set(next);
                    }
                    Key::Enter => {
                        evt.prevent_default();
                        if let Some(cmd) = filtered.get(sel_idx) {
                            let cmd_id = cmd.id.to_string();
                            cmd_state.write().open.set(false);
                            cmd_state.write().query.set(String::new());
                            cmd_state.write().selected_index.set(0);
                            execute_command(
                                &cmd_id,
                                &mut active_view,
                                &selected_task_id,
                                &api,
                                &task_state,
                            );
                        }
                    }
                    _ => {}
                }
            },
            // Input area
            div { class: "command-input-wrap",
                div { class: "command-input-icon",
                    svg {
                        width: "16",
                        height: "16",
                        view_box: "0 0 24 24",
                        fill: "none",
                        stroke: "currentColor",
                        stroke_width: "2",
                        stroke_linecap: "round",
                        stroke_linejoin: "round",
                        circle { cx: "11", cy: "11", r: "8" }
                        line { x1: "21", y1: "21", x2: "16.65", y2: "16.65" }
                    }
                }
                input {
                    class: "command-input",
                    placeholder: "Type a command or search...",
                    value: "{query_val}",
                    autofocus: true,
                    oninput: move |e: Event<FormData>| {
                        cmd_state.write().query.set(e.value());
                        cmd_state.write().selected_index.set(0);
                    },
                }
                span { class: "command-shortcut-hint", "ESC" }
            }
            // Results
            div { class: "command-results",
                if filtered.is_empty() {
                    div { class: "command-group-label", "No results" }
                } else {
                    {groups.into_iter().map(|(category, items)| {
                        rsx! {
                            div { class: "command-group",
                                div { class: "command-group-label", "{category.label()}" }
                                {items.into_iter().map(|(global_idx, cmd)| {
                                    let is_active = global_idx == sel_idx;
                                    let active_class = if is_active { "command-item active" } else { "command-item" };
                                    let cmd_id = cmd.id.to_string();
                                    rsx! {
                                        div {
                                            class: "{active_class}",
                                            onclick: move |_| {
                                                let id = cmd_id.clone();
                                                cmd_state.write().open.set(false);
                                                cmd_state.write().query.set(String::new());
                                                cmd_state.write().selected_index.set(0);
                                                execute_command(
                                                    &id,
                                                    &mut active_view,
                                                    &selected_task_id,
                                                    &api,
                                                    &task_state,
                                                );
                                            },
                                            onmouseenter: move |_| {
                                                cmd_state.write().selected_index.set(global_idx);
                                            },
                                            div { class: "command-item-icon",
                                                {command_icon(cmd.id)}
                                            }
                                            span { class: "command-item-label", "{cmd.label}" }
                                            if let Some(shortcut) = cmd.shortcut {
                                                span { class: "command-item-shortcut", "{shortcut}" }
                                            }
                                        }
                                    }
                                })}
                            }
                        }
                    })}
                }
            }
        }
    }
}

fn group_by_category(commands: &[crate::state::command::Command]) -> Vec<(CommandCategory, Vec<(usize, &crate::state::command::Command)>)> {
    let mut groups: Vec<(CommandCategory, Vec<(usize, &crate::state::command::Command)>)> = Vec::new();

    for (i, cmd) in commands.iter().enumerate() {
        if let Some(group) = groups.iter_mut().find(|(cat, _)| *cat == cmd.category) {
            group.1.push((i, cmd));
        } else {
            groups.push((cmd.category.clone(), vec![(i, cmd)]));
        }
    }

    groups
}

fn command_icon(id: &str) -> Element {
    // Simple icons based on command type
    let (d, _stroke_width) = match id {
        "nav.inbox" => ("M21 14H14L12 16L10 14H3V4C3 3.45 3.45 3 4 3H20C20.55 3 21 3.45 21 4V14ZM3 14V19C3 19.55 3.45 20 4 20H20C20.55 20 21 19.55 21 19V14", "1.5"),
        "nav.today" => ("M12 2L15.09 8.26L22 9.27L17 14.14L18.18 21.02L12 17.77L5.82 21.02L7 14.14L2 9.27L8.91 8.26L12 2Z", "1.5"),
        "nav.upcoming" => ("M8 2V5M16 2V5M3.5 9.09H20.5M21 8.5V17C21 20 19.5 22 16 22H8C4.5 22 3 20 3 17V8.5C3 5.5 4.5 3.5 8 3.5H16C19.5 3.5 21 5.5 21 8.5Z", "1.5"),
        "nav.someday" => ("M12 3C7.03 3 3 7.03 3 12H1L3.89 14.89L3.96 15.03L7 12H5C5 8.13 8.13 5 12 5S19 8.13 19 12S15.87 19 12 19C10.07 19 8.32 18.21 7.06 16.94L5.64 18.36C7.27 19.99 9.51 21 12 21C16.97 21 21 16.97 21 12S16.97 3 12 3Z", "1.5"),
        "nav.logbook" => ("M9 5H7C5.9 5 5 5.9 5 7V19C5 20.1 5.9 21 7 21H17C18.1 21 19 20.1 19 19V7C19 5.9 18.1 5 17 5H15M9 5C9 3.9 9.9 3 11 3H13C14.1 3 15 3.9 15 5M9 5C9 6.1 9.9 7 11 7H13C14.1 7 15 6.1 15 5", "1.5"),
        id if id.starts_with("task.") => ("M9 11L12 14L22 4M21 12V19C21 20.1 20.1 21 19 21H5C3.9 21 3 20.1 3 19V5C3 3.9 3.9 3 5 3H16", "1.5"),
        id if id.starts_with("create.") => ("M12 5V19M5 12H19", "2"),
        _ => ("M12 5V19M5 12H19", "2"),
    };

    rsx! {
        svg {
            width: "16",
            height: "16",
            view_box: "0 0 24 24",
            fill: "none",
            stroke: "currentColor",
            stroke_width: "1.5",
            stroke_linecap: "round",
            stroke_linejoin: "round",
            path { d: "{d}" }
        }
    }
}

fn execute_command(
    cmd_id: &str,
    active_view: &mut Signal<ActiveView>,
    selected_task_id: &Signal<Option<String>>,
    api: &Signal<ApiClient>,
    task_state: &Signal<TaskState>,
) {
    // Navigation commands
    if let Some(view) = command_to_view(cmd_id) {
        active_view.set(view);
        return;
    }

    match cmd_id {
        "create.task" | "create.task_inbox" => {
            let api_clone = api.read().clone();
            let mut ts = *task_state;
            spawn(async move {
                if let Ok(task) = api_clone.create_task("New task").await {
                    // Add to inbox
                    let mut inbox = ts.read().inbox.read().clone();
                    inbox.push(task);
                    ts.write().inbox.set(inbox);
                }
            });
        }
        "task.complete" => {
            if let Some(tid) = selected_task_id.read().clone() {
                let api_clone = api.read().clone();
                spawn(async move {
                    let _ = api_clone.complete_task(&tid).await;
                });
            }
        }
        "task.schedule_today" => {
            if let Some(tid) = selected_task_id.read().clone() {
                let api_clone = api.read().clone();
                spawn(async move {
                    let _ = api_clone.update_task_schedule(&tid, "today").await;
                });
            }
        }
        "task.defer_someday" => {
            if let Some(tid) = selected_task_id.read().clone() {
                let api_clone = api.read().clone();
                spawn(async move {
                    let _ = api_clone.update_task_schedule(&tid, "someday").await;
                });
            }
        }
        "task.move_inbox" => {
            if let Some(tid) = selected_task_id.read().clone() {
                let api_clone = api.read().clone();
                spawn(async move {
                    let _ = api_clone.update_task_schedule(&tid, "inbox").await;
                });
            }
        }
        "task.delete" => {
            if let Some(tid) = selected_task_id.read().clone() {
                let api_clone = api.read().clone();
                spawn(async move {
                    let _ = api_clone.delete_task(&tid).await;
                });
            }
        }
        _ => {}
    }
}
