use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Task {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Notes")]
    pub notes: String,
    #[serde(rename = "Status")]
    pub status: i64,
    #[serde(rename = "Schedule")]
    pub schedule: i64,
    #[serde(rename = "StartDate")]
    pub start_date: Option<String>,
    #[serde(rename = "Deadline")]
    pub deadline: Option<String>,
    #[serde(rename = "CompletedAt")]
    pub completed_at: Option<String>,
    #[serde(rename = "CreatedAt")]
    pub created_at: String,
    #[serde(rename = "UpdatedAt")]
    pub updated_at: String,
    #[serde(rename = "Index")]
    pub index: i64,
    #[serde(rename = "TodayIndex")]
    pub today_index: Option<i64>,
    #[serde(rename = "ProjectID")]
    pub project_id: Option<String>,
    #[serde(rename = "SectionID")]
    pub section_id: Option<String>,
    #[serde(rename = "AreaID")]
    pub area_id: Option<String>,
    #[serde(rename = "LocationID")]
    pub location_id: Option<String>,
    #[serde(rename = "RecurrenceRule")]
    pub recurrence_rule: Option<RecurrenceRule>,
    #[serde(rename = "Tags", default)]
    pub tags: Option<Vec<String>>,
    #[serde(rename = "Deleted", default)]
    pub deleted: bool,
    #[serde(rename = "DeletedAt")]
    pub deleted_at: Option<String>,
}

// Go domain: StatusPending=0, StatusCompleted=1, StatusCancelled=2
// Go domain: ScheduleInbox=0, ScheduleAnytime=1, ScheduleSomeday=2
impl Task {
    pub fn is_completed(&self) -> bool {
        self.status == 1
    }

    pub fn is_cancelled(&self) -> bool {
        self.status == 2
    }

    pub fn is_today(&self) -> bool {
        self.today_index.is_some()
    }

    pub fn schedule_name(&self) -> &str {
        match self.schedule {
            0 => "Inbox",
            1 => "Anytime",
            2 => "Someday",
            _ => "Unknown",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RecurrenceRule {
    pub mode: String,
    pub interval: u32,
    pub unit: String,
    pub end: Option<RecurrenceEnd>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RecurrenceEnd {
    pub date: Option<String>,
    pub count: Option<u32>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Project {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Notes")]
    pub notes: Option<String>,
    #[serde(rename = "Status")]
    pub status: i64,
    #[serde(rename = "Schedule", default)]
    pub schedule: i64,
    #[serde(rename = "StartDate")]
    pub start_date: Option<String>,
    #[serde(rename = "Deadline")]
    pub deadline: Option<String>,
    #[serde(rename = "CompletedAt")]
    pub completed_at: Option<String>,
    #[serde(rename = "CreatedAt", default)]
    pub created_at: String,
    #[serde(rename = "UpdatedAt", default)]
    pub updated_at: String,
    #[serde(rename = "Index", default)]
    pub index: i64,
    #[serde(rename = "AreaID")]
    pub area_id: Option<String>,
    #[serde(rename = "Tags", default)]
    pub tags: Option<Vec<String>>,
    #[serde(rename = "Color", default)]
    pub color: String,
    #[serde(rename = "AutoComplete", default)]
    pub auto_complete: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Section {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "ProjectID")]
    pub project_id: String,
    #[serde(rename = "Index")]
    pub index: i64,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Area {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Index")]
    pub index: i64,
    #[serde(rename = "Archived")]
    pub archived: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Tag {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Index")]
    pub index: i64,
    #[serde(rename = "ParentID")]
    pub parent_id: Option<String>,
    #[serde(rename = "Shortcut")]
    pub shortcut: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ChecklistItem {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Status")]
    pub status: i64,
    #[serde(rename = "TaskID")]
    pub task_id: String,
    #[serde(rename = "Index")]
    pub index: i64,
}

// Go domain: ChecklistPending=0, ChecklistCompleted=1
impl ChecklistItem {
    pub fn is_completed(&self) -> bool {
        self.status == 1
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Activity {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "TaskID")]
    pub task_id: String,
    #[serde(rename = "ActorID")]
    pub actor_id: String,
    #[serde(rename = "ActorType")]
    pub actor_type: String,
    #[serde(rename = "Type")]
    pub activity_type: String,
    #[serde(rename = "Content")]
    pub content: String,
    #[serde(rename = "CreatedAt")]
    pub created_at: String,
}

/// SSE event from the server's /events/stream endpoint.
#[derive(Debug, Clone, Deserialize)]
pub struct SseEvent {
    pub entity_type: String,
    pub entity_id: String,
    pub actor_id: String,
    #[serde(flatten)]
    pub extra: serde_json::Value,
}

/// Wrapper for mutation responses: { "event": "task.created", "data": {...} }
#[derive(Debug, Clone, Deserialize)]
pub struct EventEnvelope<T> {
    pub event: String,
    pub data: T,
}
