use dioxus::prelude::*;

#[derive(Debug, Clone, PartialEq)]
pub enum ActiveView {
    Inbox,
    Today,
    Upcoming,
    Someday,
    Logbook,
    Project(String),
}

impl Default for ActiveView {
    fn default() -> Self {
        Self::Today
    }
}

/// Newtype wrapper for selected task ID signal.
/// Shared via context between App (reads for detail panel) and views (writes on click).
/// Using a newtype ensures Dioxus reactivity works across parent/child boundaries.
#[derive(Clone, Copy)]
pub struct SelectedTask(pub Signal<Option<String>>);
