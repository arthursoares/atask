use dioxus::prelude::*;

use crate::state::navigation::ActiveView;

#[derive(Clone)]
pub struct CommandState {
    pub open: Signal<bool>,
    pub query: Signal<String>,
    pub selected_index: Signal<usize>,
}

impl Default for CommandState {
    fn default() -> Self {
        Self {
            open: Signal::new(false),
            query: Signal::new(String::new()),
            selected_index: Signal::new(0),
        }
    }
}

#[derive(Clone, Debug)]
pub struct Command {
    pub id: &'static str,
    pub label: &'static str,
    pub shortcut: Option<&'static str>,
    pub category: CommandCategory,
}

#[derive(Clone, Debug, PartialEq)]
pub enum CommandCategory {
    Navigation,
    TaskAction,
    Creation,
}

impl CommandCategory {
    pub fn label(&self) -> &'static str {
        match self {
            CommandCategory::Navigation => "NAVIGATION",
            CommandCategory::TaskAction => "TASK ACTIONS",
            CommandCategory::Creation => "CREATE",
        }
    }
}

pub fn all_commands(has_selected_task: bool) -> Vec<Command> {
    let mut cmds = vec![
        // Navigation
        Command {
            id: "nav.inbox",
            label: "Go to Inbox",
            shortcut: Some("\u{2318}1"),
            category: CommandCategory::Navigation,
        },
        Command {
            id: "nav.today",
            label: "Go to Today",
            shortcut: Some("\u{2318}2"),
            category: CommandCategory::Navigation,
        },
        Command {
            id: "nav.upcoming",
            label: "Go to Upcoming",
            shortcut: Some("\u{2318}3"),
            category: CommandCategory::Navigation,
        },
        Command {
            id: "nav.someday",
            label: "Go to Someday",
            shortcut: Some("\u{2318}4"),
            category: CommandCategory::Navigation,
        },
        Command {
            id: "nav.logbook",
            label: "Go to Logbook",
            shortcut: Some("\u{2318}5"),
            category: CommandCategory::Navigation,
        },
        // Creation
        Command {
            id: "create.task",
            label: "New Task",
            shortcut: Some("\u{2318}N"),
            category: CommandCategory::Creation,
        },
        Command {
            id: "create.task_inbox",
            label: "New Task in Inbox",
            shortcut: Some("\u{2318}\u{21e7}N"),
            category: CommandCategory::Creation,
        },
    ];

    if has_selected_task {
        cmds.extend(vec![
            Command {
                id: "task.complete",
                label: "Complete Task",
                shortcut: Some("\u{2318}\u{21e7}C"),
                category: CommandCategory::TaskAction,
            },
            Command {
                id: "task.schedule_today",
                label: "Schedule for Today",
                shortcut: Some("\u{2318}T"),
                category: CommandCategory::TaskAction,
            },
            Command {
                id: "task.defer_someday",
                label: "Defer to Someday",
                shortcut: None,
                category: CommandCategory::TaskAction,
            },
            Command {
                id: "task.move_inbox",
                label: "Move to Inbox",
                shortcut: None,
                category: CommandCategory::TaskAction,
            },
            Command {
                id: "task.delete",
                label: "Delete Task",
                shortcut: Some("\u{232b}"),
                category: CommandCategory::TaskAction,
            },
        ]);
    }

    cmds
}

pub fn filter_commands(commands: &[Command], query: &str) -> Vec<Command> {
    if query.is_empty() {
        return commands.to_vec();
    }
    let q = query.to_lowercase();
    commands
        .iter()
        .filter(|c| c.label.to_lowercase().contains(&q))
        .cloned()
        .collect()
}

/// Map a command ID to an ActiveView navigation target.
pub fn command_to_view(id: &str) -> Option<ActiveView> {
    match id {
        "nav.inbox" => Some(ActiveView::Inbox),
        "nav.today" => Some(ActiveView::Today),
        "nav.upcoming" => Some(ActiveView::Upcoming),
        "nav.someday" => Some(ActiveView::Someday),
        "nav.logbook" => Some(ActiveView::Logbook),
        _ => None,
    }
}
