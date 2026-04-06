use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Task {
    pub id: String,
    pub title: String,
    pub notes: String,
    pub status: i32,
    pub schedule: i32,
    pub start_date: Option<String>,
    pub deadline: Option<String>,
    pub completed_at: Option<String>,
    pub index: i32,
    pub today_index: Option<i32>,
    pub time_slot: Option<String>,
    pub project_id: Option<String>,
    pub section_id: Option<String>,
    pub area_id: Option<String>,
    pub location_id: Option<String>,
    pub created_at: String,
    pub updated_at: String,
    pub sync_status: i32,
    pub repeat_rule: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Project {
    pub id: String,
    pub title: String,
    pub notes: String,
    pub status: i32,
    pub color: String,
    pub area_id: Option<String>,
    pub index: i32,
    pub completed_at: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Area {
    pub id: String,
    pub title: String,
    pub index: i32,
    pub archived: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Section {
    pub id: String,
    pub title: String,
    pub project_id: String,
    pub index: i32,
    pub archived: bool,
    pub collapsed: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Tag {
    pub id: String,
    pub title: String,
    pub index: i32,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct TaskTag {
    pub task_id: String,
    pub tag_id: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct ProjectTag {
    pub project_id: String,
    pub tag_id: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct TaskLink {
    pub task_id: String,
    pub linked_task_id: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct ChecklistItem {
    pub id: String,
    pub title: String,
    pub status: i32,
    pub task_id: String,
    pub index: i32,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Activity {
    pub id: String,
    pub task_id: String,
    pub actor_id: Option<String>,
    pub actor_type: String,
    #[serde(rename = "type")]
    pub activity_type: String,
    pub content: String,
    pub created_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Location {
    pub id: String,
    pub name: String,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<i32>,
    pub address: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AppState {
    pub tasks: Vec<Task>,
    pub projects: Vec<Project>,
    pub areas: Vec<Area>,
    pub sections: Vec<Section>,
    pub tags: Vec<Tag>,
    pub task_tags: Vec<TaskTag>,
    pub task_links: Vec<TaskLink>,
    pub project_tags: Vec<ProjectTag>,
    pub checklist_items: Vec<ChecklistItem>,
    pub activities: Vec<Activity>,
    pub locations: Vec<Location>,
}
