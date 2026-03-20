use dioxus::prelude::*;
use crate::api::types::{Project, Section, Area, Tag, Task};
use std::collections::HashMap;

#[derive(Clone, Default)]
pub struct ProjectState {
    pub projects: Signal<Vec<Project>>,
    pub areas: Signal<Vec<Area>>,
    pub tags: Signal<Vec<Tag>>,
    pub sections: Signal<HashMap<String, Vec<Section>>>,
    pub project_tasks: Signal<HashMap<String, Vec<Task>>>,
}
