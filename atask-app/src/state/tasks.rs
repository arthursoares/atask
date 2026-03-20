use dioxus::prelude::*;
use crate::api::types::Task;

#[derive(Clone, Default)]
pub struct TaskState {
    pub inbox: Signal<Vec<Task>>,
    pub today: Signal<Vec<Task>>,
    pub upcoming: Signal<Vec<Task>>,
    pub someday: Signal<Vec<Task>>,
    pub logbook: Signal<Vec<Task>>,
    pub loading: Signal<bool>,
}
