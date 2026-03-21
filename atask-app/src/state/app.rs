use std::collections::HashMap;
use dioxus::prelude::*;
use crate::api::client::ApiClient;
use crate::api::types::{Task, Project, Area, Tag, Section};

// ── Navigation ──

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

// ── Newtype Signal Wrappers ──
// Each is Clone + Copy so they can be passed through context and closures.
// The inner Signal is the reactive primitive.

#[derive(Clone, Copy)]
pub struct TokenSignal(pub Signal<Option<String>>);

#[derive(Clone, Copy)]
pub struct ApiSignal(pub Signal<ApiClient>);

#[derive(Clone, Copy)]
pub struct ViewSignal(pub Signal<ActiveView>);

#[derive(Clone, Copy)]
pub struct SelectedTaskSignal(pub Signal<Option<String>>);

#[derive(Clone, Copy)]
pub struct InboxTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct TodayTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct UpcomingTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct SomedayTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct LogbookTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct ProjectList(pub Signal<Vec<Project>>);

#[derive(Clone, Copy)]
pub struct AreaList(pub Signal<Vec<Area>>);

#[derive(Clone, Copy)]
pub struct TagList(pub Signal<Vec<Tag>>);

#[derive(Clone, Copy)]
pub struct LoadingSignal(pub Signal<bool>);

#[derive(Clone, Copy)]
pub struct ProjectTasks(pub Signal<HashMap<String, Vec<Task>>>);

#[derive(Clone, Copy)]
pub struct ProjectSections(pub Signal<HashMap<String, Vec<Section>>>);

// ── Command Palette ──

#[derive(Clone, Copy)]
pub struct CommandOpen(pub Signal<bool>);

#[derive(Clone, Copy)]
pub struct CommandQuery(pub Signal<String>);

#[derive(Clone, Copy)]
pub struct CommandIndex(pub Signal<usize>);
