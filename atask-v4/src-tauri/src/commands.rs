use crate::db::Database;
use crate::models::*;
use crate::patch_body::{area_patch_body, project_patch_body, task_patch_body};
use chrono::{Datelike, Duration, NaiveDate, Utc};
use serde::{Deserialize, Serialize};

/// Run `f` inside a SQLite transaction on the locked `conn`. The transaction
/// is committed iff `f` returns `Ok`; any `Err` (or panic) causes a rollback.
///
/// This is the safety net for the write-then-enqueue pattern used by every
/// mutating Tauri command: the local row mutation AND the matching
/// `pendingOps` INSERT must land together, or neither. Without this, a crash
/// between the two writes leaves either a local change that never syncs or a
/// pending op whose local state has already been rolled forward to something
/// else.
pub(crate) fn with_tx<F, T>(conn: &rusqlite::Connection, f: F) -> Result<T, String>
where
    F: FnOnce(&rusqlite::Connection) -> Result<T, String>,
{
    let tx = conn
        .unchecked_transaction()
        .map_err(|e| e.to_string())?;
    let result = f(&tx)?;
    tx.commit().map_err(|e| e.to_string())?;
    Ok(result)
}

fn queue_pending_op(
    conn: &rusqlite::Connection,
    method: &str,
    path: &str,
    body: Option<&str>,
) -> Result<(), String> {
    let enabled: String = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'sync_enabled'",
            [],
            |row| row.get(0),
        )
        .unwrap_or_default();
    if enabled != "true" {
        return Ok(());
    }
    let now = chrono::Utc::now().to_rfc3339();
    conn.execute(
        "INSERT INTO pendingOps (method, path, body, createdAt, synced) VALUES (?1, ?2, ?3, ?4, 0)",
        rusqlite::params![method, path, body, now],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateTaskParams {
    pub title: String,
    pub notes: Option<String>,
    pub schedule: Option<i32>,
    pub start_date: Option<String>,
    pub deadline: Option<String>,
    pub time_slot: Option<String>,
    pub project_id: Option<String>,
    pub section_id: Option<String>,
    pub area_id: Option<String>,
    pub tag_ids: Option<Vec<String>>,
    pub repeat_rule: Option<String>,
}

fn query_task(conn: &rusqlite::Connection, id: &str) -> Result<Task, String> {
    conn.query_row(
        "SELECT id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, locationId, createdAt, updatedAt, syncStatus, repeatRule FROM tasks WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            Ok(Task {
                id: row.get(0)?,
                title: row.get(1)?,
                notes: row.get(2)?,
                status: row.get(3)?,
                schedule: row.get(4)?,
                start_date: row.get(5)?,
                deadline: row.get(6)?,
                completed_at: row.get(7)?,
                index: row.get(8)?,
                today_index: row.get(9)?,
                time_slot: row.get(10)?,
                project_id: row.get(11)?,
                section_id: row.get(12)?,
                area_id: row.get(13)?,
                location_id: row.get(14)?,
                created_at: row.get(15)?,
                updated_at: row.get(16)?,
                sync_status: row.get(17)?,
                repeat_rule: row.get(18)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

fn query_all_tasks(conn: &rusqlite::Connection) -> Result<Vec<Task>, String> {
    let mut stmt = conn
        .prepare("SELECT id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, locationId, createdAt, updatedAt, syncStatus, repeatRule FROM tasks")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(Task {
                id: row.get(0)?,
                title: row.get(1)?,
                notes: row.get(2)?,
                status: row.get(3)?,
                schedule: row.get(4)?,
                start_date: row.get(5)?,
                deadline: row.get(6)?,
                completed_at: row.get(7)?,
                index: row.get(8)?,
                today_index: row.get(9)?,
                time_slot: row.get(10)?,
                project_id: row.get(11)?,
                section_id: row.get(12)?,
                area_id: row.get(13)?,
                location_id: row.get(14)?,
                created_at: row.get(15)?,
                updated_at: row.get(16)?,
                sync_status: row.get(17)?,
                repeat_rule: row.get(18)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_projects(conn: &rusqlite::Connection) -> Result<Vec<Project>, String> {
    let mut stmt = conn
        .prepare("SELECT id, title, notes, status, color, areaId, \"index\", completedAt, createdAt, updatedAt FROM projects")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(Project {
                id: row.get(0)?,
                title: row.get(1)?,
                notes: row.get(2)?,
                status: row.get(3)?,
                color: row.get(4)?,
                area_id: row.get(5)?,
                index: row.get(6)?,
                completed_at: row.get(7)?,
                created_at: row.get(8)?,
                updated_at: row.get(9)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_areas(conn: &rusqlite::Connection) -> Result<Vec<Area>, String> {
    let mut stmt = conn
        .prepare("SELECT id, title, \"index\", archived, createdAt, updatedAt FROM areas")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            let archived: i32 = row.get(3)?;
            Ok(Area {
                id: row.get(0)?,
                title: row.get(1)?,
                index: row.get(2)?,
                archived: archived != 0,
                created_at: row.get(4)?,
                updated_at: row.get(5)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_sections(conn: &rusqlite::Connection) -> Result<Vec<Section>, String> {
    let mut stmt = conn
        .prepare("SELECT id, title, projectId, \"index\", archived, collapsed, createdAt, updatedAt FROM sections")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            let archived: i32 = row.get(4)?;
            let collapsed: i32 = row.get(5)?;
            Ok(Section {
                id: row.get(0)?,
                title: row.get(1)?,
                project_id: row.get(2)?,
                index: row.get(3)?,
                archived: archived != 0,
                collapsed: collapsed != 0,
                created_at: row.get(6)?,
                updated_at: row.get(7)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_tags(conn: &rusqlite::Connection) -> Result<Vec<Tag>, String> {
    let mut stmt = conn
        .prepare("SELECT id, title, \"index\", createdAt, updatedAt FROM tags")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(Tag {
                id: row.get(0)?,
                title: row.get(1)?,
                index: row.get(2)?,
                created_at: row.get(3)?,
                updated_at: row.get(4)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_task_tags(conn: &rusqlite::Connection) -> Result<Vec<TaskTag>, String> {
    let mut stmt = conn
        .prepare("SELECT taskId, tagId FROM taskTags")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(TaskTag {
                task_id: row.get(0)?,
                tag_id: row.get(1)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_task_links(conn: &rusqlite::Connection) -> Result<Vec<TaskLink>, String> {
    let mut stmt = conn
        .prepare("SELECT taskId, linkedTaskId FROM taskLinks")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(TaskLink {
                task_id: row.get(0)?,
                linked_task_id: row.get(1)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_project_tags(conn: &rusqlite::Connection) -> Result<Vec<ProjectTag>, String> {
    let mut stmt = conn
        .prepare("SELECT projectId, tagId FROM projectTags")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(ProjectTag {
                project_id: row.get(0)?,
                tag_id: row.get(1)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_checklist_items(conn: &rusqlite::Connection) -> Result<Vec<ChecklistItem>, String> {
    let mut stmt = conn
        .prepare(
            "SELECT id, title, status, taskId, \"index\", createdAt, updatedAt FROM checklistItems",
        )
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(ChecklistItem {
                id: row.get(0)?,
                title: row.get(1)?,
                status: row.get(2)?,
                task_id: row.get(3)?,
                index: row.get(4)?,
                created_at: row.get(5)?,
                updated_at: row.get(6)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn query_all_activities(conn: &rusqlite::Connection) -> Result<Vec<Activity>, String> {
    let mut stmt = conn
        .prepare("SELECT id, taskId, actorId, actorType, type, content, createdAt FROM activities ORDER BY createdAt ASC")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(Activity {
                id: row.get(0)?,
                task_id: row.get(1)?,
                actor_id: row.get(2)?,
                actor_type: row.get(3)?,
                activity_type: row.get(4)?,
                content: row.get(5)?,
                created_at: row.get(6)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>().map_err(|e| e.to_string())
}

fn query_all_locations(conn: &rusqlite::Connection) -> Result<Vec<Location>, String> {
    let mut stmt = conn
        .prepare("SELECT id, name, latitude, longitude, radius, address, createdAt, updatedAt FROM locations")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([], |row| {
            Ok(Location {
                id: row.get(0)?,
                name: row.get(1)?,
                latitude: row.get(2)?,
                longitude: row.get(3)?,
                radius: row.get(4)?,
                address: row.get(5)?,
                created_at: row.get(6)?,
                updated_at: row.get(7)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn read_location(conn: &rusqlite::Connection, id: &str) -> Result<Location, String> {
    conn.query_row(
        "SELECT id, name, latitude, longitude, radius, address, createdAt, updatedAt FROM locations WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            Ok(Location {
                id: row.get(0)?,
                name: row.get(1)?,
                latitude: row.get(2)?,
                longitude: row.get(3)?,
                radius: row.get(4)?,
                address: row.get(5)?,
                created_at: row.get(6)?,
                updated_at: row.get(7)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

pub(crate) fn load_all_impl(conn: &rusqlite::Connection) -> Result<AppState, String> {
    Ok(AppState {
        tasks: query_all_tasks(conn)?,
        projects: query_all_projects(conn)?,
        areas: query_all_areas(conn)?,
        sections: query_all_sections(conn)?,
        tags: query_all_tags(conn)?,
        task_tags: query_all_task_tags(conn)?,
        task_links: query_all_task_links(conn)?,
        project_tags: query_all_project_tags(conn)?,
        checklist_items: query_all_checklist_items(conn)?,
        activities: query_all_activities(conn)?,
        locations: query_all_locations(conn)?,
    })
}

#[tauri::command]
pub fn load_all(db: tauri::State<'_, Database>) -> Result<AppState, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    load_all_impl(&conn)
}

pub(crate) fn create_task_impl(
    conn: &rusqlite::Connection,
    params: CreateTaskParams,
) -> Result<Task, String> {
    let id = uuid::Uuid::new_v4().to_string();
    let now = chrono::Utc::now().to_rfc3339();

    conn.execute(
        "INSERT INTO tasks (id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, createdAt, updatedAt, syncStatus, repeatRule) VALUES (?1, ?2, ?3, 0, ?4, ?5, ?6, NULL, 0, NULL, ?7, ?8, ?9, ?10, ?11, ?11, 0, ?12)",
        rusqlite::params![
            id,
            params.title,
            params.notes.unwrap_or_default(),
            params.schedule.unwrap_or(0),
            params.start_date,
            params.deadline,
            params.time_slot,
            params.project_id,
            params.section_id,
            params.area_id,
            now,
            params.repeat_rule,
        ],
    )
    .map_err(|e| e.to_string())?;

    if let Some(tag_ids) = &params.tag_ids {
        for tag_id in tag_ids {
            conn.execute(
                "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?1, ?2)",
                rusqlite::params![id, tag_id],
            )
            .map_err(|e| e.to_string())?;
        }
    }

    query_task(conn, &id)
}

#[tauri::command]
pub fn create_task(
    db: tauri::State<'_, Database>,
    params: CreateTaskParams,
) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let task = create_task_impl(tx, params)?;
        // Send minimal JSON with id + title for the Go API
        let create_body = serde_json::json!({"id": task.id, "title": task.title}).to_string();
        queue_pending_op(tx, "POST", "/tasks", Some(&create_body))?;
        // Follow-up PATCH must use the narrow body contract (see patch_body.rs).
        let patch_body = task_patch_body(&task);
        queue_pending_op(tx, "PATCH", &format!("/tasks/{}", task.id), Some(&patch_body))?;
        Ok(task)
    })
}

// --- Recurrence helpers ---

fn days_in_month(year: i32, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 => {
            if year % 400 == 0 || (year % 4 == 0 && year % 100 != 0) {
                29
            } else {
                28
            }
        }
        _ => 30,
    }
}

fn compute_next_date(base: &str, interval: i64, unit: &str) -> Option<String> {
    let base_date = NaiveDate::parse_from_str(base, "%Y-%m-%d").ok()?;
    let next = match unit {
        "day" => base_date + Duration::days(interval),
        "week" => base_date + Duration::weeks(interval),
        "month" => {
            let month = base_date.month0() as i32 + interval as i32;
            let year = base_date.year() + month / 12;
            let month = (month % 12) as u32 + 1;
            let day = base_date.day().min(days_in_month(year, month));
            NaiveDate::from_ymd_opt(year, month, day)?
        }
        "year" => NaiveDate::from_ymd_opt(
            base_date.year() + interval as i32,
            base_date.month(),
            base_date.day(),
        )?,
        _ => return None,
    };
    Some(next.format("%Y-%m-%d").to_string())
}

// --- Mutation commands ---

pub(crate) fn complete_task_impl(conn: &rusqlite::Connection, id: &str) -> Result<Task, String> {
    let now = Utc::now().to_rfc3339();
    let today = Utc::now().format("%Y-%m-%d").to_string();

    // Fetch the task before completing it so we can read repeatRule / startDate.
    let task = query_task(conn, id)?;

    conn.execute(
        "UPDATE tasks SET status = 1, completedAt = ?1, updatedAt = ?2, syncStatus = 1 WHERE id = ?3",
        rusqlite::params![now, now, id],
    )
    .map_err(|e| e.to_string())?;

    // Recurrence: if the task has a repeatRule, create the next occurrence.
    if let Some(repeat_rule_json) = &task.repeat_rule {
        if let Ok(rule) = serde_json::from_str::<serde_json::Value>(repeat_rule_json) {
            let rule_type = rule["type"].as_str().unwrap_or("");
            let interval = rule["interval"].as_i64().unwrap_or(1);
            let unit = rule["unit"].as_str().unwrap_or("day");

            let base_date = match rule_type {
                "fixed" => task.start_date.as_deref().unwrap_or(&today).to_string(),
                "afterCompletion" => today.clone(),
                _ => today.clone(),
            };

            if let Some(next_date) = compute_next_date(&base_date, interval, unit) {
                let new_id = uuid::Uuid::new_v4().to_string();
                let now2 = Utc::now().to_rfc3339();

                conn.execute(
                    "INSERT INTO tasks (id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, createdAt, updatedAt, syncStatus, repeatRule) VALUES (?1, ?2, ?3, 0, ?4, ?5, NULL, NULL, 0, NULL, ?6, ?7, ?8, ?9, ?10, ?10, 0, ?11)",
                    rusqlite::params![
                        new_id,
                        task.title,
                        task.notes,
                        task.schedule,
                        next_date,
                        task.time_slot,
                        task.project_id,
                        task.section_id,
                        task.area_id,
                        now2,
                        repeat_rule_json,
                    ],
                )
                .map_err(|e| e.to_string())?;

                // Copy tag associations.
                let mut tag_stmt = conn
                    .prepare("SELECT tagId FROM taskTags WHERE taskId = ?1")
                    .map_err(|e| e.to_string())?;
                let tag_ids: Vec<String> = tag_stmt
                    .query_map(rusqlite::params![id], |row| row.get(0))
                    .map_err(|e| e.to_string())?
                    .collect::<Result<_, _>>()
                    .map_err(|e| e.to_string())?;

                for tag_id in tag_ids {
                    conn.execute(
                        "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?1, ?2)",
                        rusqlite::params![new_id, tag_id],
                    )
                    .map_err(|e| e.to_string())?;
                }
            }
        }
    }

    query_task(conn, id)
}

#[tauri::command]
pub fn complete_task(db: tauri::State<'_, Database>, id: String) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let task = complete_task_impl(tx, &id)?;
        queue_pending_op(tx, "POST", &format!("/tasks/{}/complete", id), None)?;
        Ok(task)
    })
}

pub(crate) fn cancel_task_impl(conn: &rusqlite::Connection, id: &str) -> Result<Task, String> {
    let now = Utc::now().to_rfc3339();

    conn.execute(
        "UPDATE tasks SET status = 2, completedAt = ?1, updatedAt = ?2, syncStatus = 1 WHERE id = ?3",
        rusqlite::params![now, now, id],
    )
    .map_err(|e| e.to_string())?;

    query_task(conn, id)
}

#[tauri::command]
pub fn cancel_task(db: tauri::State<'_, Database>, id: String) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let task = cancel_task_impl(tx, &id)?;
        queue_pending_op(tx, "POST", &format!("/tasks/{}/cancel", id), None)?;
        Ok(task)
    })
}

pub(crate) fn reopen_task_impl(conn: &rusqlite::Connection, id: &str) -> Result<Task, String> {
    let now = Utc::now().to_rfc3339();

    conn.execute(
        "UPDATE tasks SET status = 0, completedAt = NULL, updatedAt = ?1, syncStatus = 1 WHERE id = ?2",
        rusqlite::params![now, id],
    )
    .map_err(|e| e.to_string())?;

    query_task(conn, id)
}

#[tauri::command]
pub fn reopen_task(db: tauri::State<'_, Database>, id: String) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let task = reopen_task_impl(tx, &id)?;
        queue_pending_op(tx, "POST", &format!("/tasks/{}/reopen", id), None)?;
        Ok(task)
    })
}

// --- New commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateTaskParams {
    pub id: String,
    pub title: Option<String>,
    pub notes: Option<String>,
    pub schedule: Option<i32>,
    pub start_date: Option<Option<String>>,
    pub deadline: Option<Option<String>>,
    pub time_slot: Option<Option<String>>,
    pub project_id: Option<Option<String>>,
    pub section_id: Option<Option<String>>,
    pub area_id: Option<Option<String>>,
    pub repeat_rule: Option<Option<String>>,
    pub tag_ids: Option<Vec<String>>,
}

#[tauri::command]
pub fn update_task(
    db: tauri::State<'_, Database>,
    params: UpdateTaskParams,
) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        // Shadow `conn` so the existing body writes land on the transaction.
        let conn = tx;
        let now = Utc::now().to_rfc3339();

        // Track whether repeat_rule was provided so we can queue a separate
        // recurrence sync op (Go PATCH does not handle repeatRule).
        let repeat_rule_changed = params.repeat_rule.is_some();

    let mut sets: Vec<String> = Vec::new();
    let mut values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

    if let Some(v) = params.title {
        sets.push(format!("title = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.notes {
        sets.push(format!("notes = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.schedule {
        sets.push(format!("schedule = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.start_date {
        sets.push(format!("startDate = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.deadline {
        sets.push(format!("deadline = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.time_slot {
        sets.push(format!("timeSlot = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.project_id {
        sets.push(format!("projectId = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.section_id {
        sets.push(format!("sectionId = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.area_id {
        sets.push(format!("areaId = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.repeat_rule {
        sets.push(format!("repeatRule = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }

    // Always update updatedAt and syncStatus
    sets.push(format!("updatedAt = ?{}", sets.len() + 1));
    values.push(Box::new(now));
    sets.push(format!("syncStatus = ?{}", sets.len() + 1));
    values.push(Box::new(1i32));

    // WHERE id = ?N
    let id_param_idx = values.len() + 1;
    values.push(Box::new(params.id.clone()));

    let sql = format!(
        "UPDATE tasks SET {} WHERE id = ?{}",
        sets.join(", "),
        id_param_idx
    );

    let params_refs: Vec<&dyn rusqlite::types::ToSql> = values.iter().map(|v| v.as_ref()).collect();
    conn.execute(&sql, params_refs.as_slice())
        .map_err(|e| e.to_string())?;

    // Replace tags if provided
    if let Some(tag_ids) = &params.tag_ids {
        conn.execute(
            "DELETE FROM taskTags WHERE taskId = ?1",
            rusqlite::params![params.id],
        )
        .map_err(|e| e.to_string())?;
        for tag_id in tag_ids {
            conn.execute(
                "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?1, ?2)",
                rusqlite::params![params.id, tag_id],
            )
            .map_err(|e| e.to_string())?;
        }
    }

        let task = query_task(conn, &params.id)?;
        let body = task_patch_body(&task);
        queue_pending_op(conn, "PATCH", &format!("/tasks/{}", task.id), Some(&body))?;

        // The Go PATCH endpoint does not handle repeatRule — recurrence uses a
    // dedicated PUT /tasks/{id}/recurrence endpoint that accepts the rule
    // object or null (to clear).  Queue this as a separate pending op when
    // the caller changed the repeat rule.
        if repeat_rule_changed {
            let recurrence_body = match &task.repeat_rule {
                Some(json_str) => json_str.clone(),
                None => "null".to_string(),
            };
            queue_pending_op(
                conn,
                "PUT",
                &format!("/tasks/{}/recurrence", task.id),
                Some(&recurrence_body),
            )?;
        }

        Ok(task)
    })
}

pub(crate) fn duplicate_task_impl(conn: &rusqlite::Connection, id: &str) -> Result<Task, String> {
    let now = Utc::now().to_rfc3339();
    let new_id = uuid::Uuid::new_v4().to_string();

    let task = query_task(conn, id)?;

    conn.execute(
        "INSERT INTO tasks (id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, createdAt, updatedAt, syncStatus, repeatRule) VALUES (?1, ?2, ?3, 0, ?4, ?5, ?6, NULL, ?7, NULL, ?8, ?9, ?10, ?11, ?12, ?12, 0, ?13)",
        rusqlite::params![
            new_id,
            task.title,
            task.notes,
            task.schedule,
            task.start_date,
            task.deadline,
            task.index,
            task.time_slot,
            task.project_id,
            task.section_id,
            task.area_id,
            now,
            task.repeat_rule,
        ],
    )
    .map_err(|e| e.to_string())?;

    // Copy tag associations
    let mut tag_stmt = conn
        .prepare("SELECT tagId FROM taskTags WHERE taskId = ?1")
        .map_err(|e| e.to_string())?;
    let tag_ids: Vec<String> = tag_stmt
        .query_map(rusqlite::params![id], |row| row.get(0))
        .map_err(|e| e.to_string())?
        .collect::<Result<_, _>>()
        .map_err(|e| e.to_string())?;

    for tag_id in tag_ids {
        conn.execute(
            "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?1, ?2)",
            rusqlite::params![new_id, tag_id],
        )
        .map_err(|e| e.to_string())?;
    }

    query_task(conn, &new_id)
}

#[tauri::command]
pub fn duplicate_task(db: tauri::State<'_, Database>, id: String) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let task = duplicate_task_impl(tx, &id)?;
        let create_body = serde_json::json!({"id": task.id, "title": task.title}).to_string();
        queue_pending_op(tx, "POST", "/tasks", Some(&create_body))?;
        let patch_body = task_patch_body(&task);
        queue_pending_op(tx, "PATCH", &format!("/tasks/{}", task.id), Some(&patch_body))?;
        Ok(task)
    })
}

pub(crate) fn delete_task_impl(conn: &rusqlite::Connection, id: &str) -> Result<(), String> {
    conn.execute(
        "DELETE FROM taskTags WHERE taskId = ?1",
        rusqlite::params![id],
    )
    .map_err(|e| e.to_string())?;

    conn.execute(
        "DELETE FROM checklistItems WHERE taskId = ?1",
        rusqlite::params![id],
    )
    .map_err(|e| e.to_string())?;

    conn.execute("DELETE FROM tasks WHERE id = ?1", rusqlite::params![id])
        .map_err(|e| e.to_string())?;

    Ok(())
}

#[tauri::command]
pub fn delete_task(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        delete_task_impl(tx, &id)?;
        queue_pending_op(tx, "DELETE", &format!("/tasks/{}", id), None)?;
        Ok(())
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReorderMove {
    pub id: String,
    pub index: i32,
}

#[tauri::command]
pub fn reorder_tasks(
    db: tauri::State<'_, Database>,
    moves: Vec<ReorderMove>,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        for m in &moves {
            tx.execute(
                "UPDATE tasks SET \"index\" = ?1, updatedAt = ?2, syncStatus = 1 WHERE id = ?3",
                rusqlite::params![m.index, chrono::Utc::now().to_rfc3339(), m.id],
            )
            .map_err(|e| e.to_string())?;
            let index_json = serde_json::json!({"index": m.index}).to_string();
            queue_pending_op(
                tx,
                "PUT",
                &format!("/tasks/{}/reorder", m.id),
                Some(&index_json),
            )?;
        }
        Ok(())
    })
}

#[tauri::command]
pub fn set_today_index(
    db: tauri::State<'_, Database>,
    id: String,
    index: i32,
) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE tasks SET todayIndex = ?1, updatedAt = ?2 WHERE id = ?3",
            rusqlite::params![index, now, id],
        )
        .map_err(|e| e.to_string())?;

        let index_json = serde_json::json!({"index": index}).to_string();
        queue_pending_op(
            tx,
            "PUT",
            &format!("/tasks/{}/today-index", id),
            Some(&index_json),
        )?;

        query_task(tx, &id)
    })
}

#[tauri::command]
pub fn move_task_to_section(
    db: tauri::State<'_, Database>,
    task_id: String,
    section_id: Option<String>,
) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE tasks SET sectionId = ?1, updatedAt = ?2, syncStatus = 1 WHERE id = ?3",
            rusqlite::params![section_id, now, task_id],
        )
        .map_err(|e| e.to_string())?;

        let section_json = match &section_id {
            Some(sid) => serde_json::json!({"id": sid}).to_string(),
            None => serde_json::json!({"id": null}).to_string(),
        };
        queue_pending_op(
            tx,
            "PUT",
            &format!("/tasks/{}/section", task_id),
            Some(&section_json),
        )?;

        query_task(tx, &task_id)
    })
}

// --- Row-reading helpers ---

fn read_project(conn: &rusqlite::Connection, id: &str) -> Result<Project, String> {
    conn.query_row(
        "SELECT id, title, notes, status, color, areaId, \"index\", completedAt, createdAt, updatedAt FROM projects WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            Ok(Project {
                id: row.get(0)?,
                title: row.get(1)?,
                notes: row.get(2)?,
                status: row.get(3)?,
                color: row.get(4)?,
                area_id: row.get(5)?,
                index: row.get(6)?,
                completed_at: row.get(7)?,
                created_at: row.get(8)?,
                updated_at: row.get(9)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

fn read_area(conn: &rusqlite::Connection, id: &str) -> Result<Area, String> {
    conn.query_row(
        "SELECT id, title, \"index\", archived, createdAt, updatedAt FROM areas WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            let archived: i32 = row.get(3)?;
            Ok(Area {
                id: row.get(0)?,
                title: row.get(1)?,
                index: row.get(2)?,
                archived: archived != 0,
                created_at: row.get(4)?,
                updated_at: row.get(5)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

fn read_section(conn: &rusqlite::Connection, id: &str) -> Result<Section, String> {
    conn.query_row(
        "SELECT id, title, projectId, \"index\", archived, collapsed, createdAt, updatedAt FROM sections WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            let archived: i32 = row.get(4)?;
            let collapsed: i32 = row.get(5)?;
            Ok(Section {
                id: row.get(0)?,
                title: row.get(1)?,
                project_id: row.get(2)?,
                index: row.get(3)?,
                archived: archived != 0,
                collapsed: collapsed != 0,
                created_at: row.get(6)?,
                updated_at: row.get(7)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

fn read_tag(conn: &rusqlite::Connection, id: &str) -> Result<Tag, String> {
    conn.query_row(
        "SELECT id, title, \"index\", createdAt, updatedAt FROM tags WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            Ok(Tag {
                id: row.get(0)?,
                title: row.get(1)?,
                index: row.get(2)?,
                created_at: row.get(3)?,
                updated_at: row.get(4)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

fn read_checklist_item(conn: &rusqlite::Connection, id: &str) -> Result<ChecklistItem, String> {
    conn.query_row(
        "SELECT id, title, status, taskId, \"index\", createdAt, updatedAt FROM checklistItems WHERE id = ?1",
        rusqlite::params![id],
        |row| {
            Ok(ChecklistItem {
                id: row.get(0)?,
                title: row.get(1)?,
                status: row.get(2)?,
                task_id: row.get(3)?,
                index: row.get(4)?,
                created_at: row.get(5)?,
                updated_at: row.get(6)?,
            })
        },
    )
    .map_err(|e| e.to_string())
}

// --- Project Commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateProjectParams {
    pub title: String,
    pub color: Option<String>,
    pub area_id: Option<String>,
}

pub(crate) fn create_project_impl(
    conn: &rusqlite::Connection,
    params: CreateProjectParams,
) -> Result<Project, String> {
    let id = uuid::Uuid::new_v4().to_string();
    let now = Utc::now().to_rfc3339();
    let color = params.color.unwrap_or_default();

    conn.execute(
        "INSERT INTO projects (id, title, notes, status, color, areaId, \"index\", completedAt, createdAt, updatedAt) VALUES (?1, ?2, '', 0, ?3, ?4, 0, NULL, ?5, ?5)",
        rusqlite::params![id, params.title, color, params.area_id, now],
    )
    .map_err(|e| e.to_string())?;

    read_project(conn, &id)
}

#[tauri::command]
pub fn create_project(
    db: tauri::State<'_, Database>,
    params: CreateProjectParams,
) -> Result<Project, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let project = create_project_impl(tx, params)?;
        let create_body =
            serde_json::json!({"id": project.id, "title": project.title}).to_string();
        queue_pending_op(tx, "POST", "/projects", Some(&create_body))?;
        let patch_body = project_patch_body(&project);
        queue_pending_op(
            tx,
            "PATCH",
            &format!("/projects/{}", project.id),
            Some(&patch_body),
        )?;
        Ok(project)
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateProjectParams {
    pub id: String,
    pub title: Option<String>,
    pub notes: Option<String>,
    pub color: Option<String>,
    pub area_id: Option<Option<String>>,
}

#[tauri::command]
pub fn update_project(
    db: tauri::State<'_, Database>,
    params: UpdateProjectParams,
) -> Result<Project, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let conn = tx;
        let now = Utc::now().to_rfc3339();

    let mut sets: Vec<String> = Vec::new();
    let mut values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

    if let Some(v) = params.title {
        sets.push(format!("title = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.notes {
        sets.push(format!("notes = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.color {
        sets.push(format!("color = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }
    if let Some(v) = params.area_id {
        sets.push(format!("areaId = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }

    sets.push(format!("updatedAt = ?{}", sets.len() + 1));
    values.push(Box::new(now));

    let id_param_idx = values.len() + 1;
    values.push(Box::new(params.id.clone()));

    let sql = format!(
        "UPDATE projects SET {} WHERE id = ?{}",
        sets.join(", "),
        id_param_idx
    );

    let params_refs: Vec<&dyn rusqlite::types::ToSql> = values.iter().map(|v| v.as_ref()).collect();
    conn.execute(&sql, params_refs.as_slice())
        .map_err(|e| e.to_string())?;

        let project = read_project(conn, &params.id)?;
        let body = project_patch_body(&project);
        queue_pending_op(
            conn,
            "PATCH",
            &format!("/projects/{}", project.id),
            Some(&body),
        )?;
        Ok(project)
    })
}

#[tauri::command]
pub fn complete_project(db: tauri::State<'_, Database>, id: String) -> Result<Project, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();
        tx.execute(
            "UPDATE projects SET status = 1, completedAt = ?1, updatedAt = ?1 WHERE id = ?2",
            rusqlite::params![now, id],
        )
        .map_err(|e| e.to_string())?;
        queue_pending_op(tx, "POST", &format!("/projects/{}/complete", id), None)?;
        read_project(tx, &id)
    })
}

#[tauri::command]
pub fn reopen_project(db: tauri::State<'_, Database>, id: String) -> Result<Project, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();
        tx.execute(
            "UPDATE projects SET status = 0, completedAt = NULL, updatedAt = ?1 WHERE id = ?2",
            rusqlite::params![now, id],
        )
        .map_err(|e| e.to_string())?;
        queue_pending_op(tx, "POST", &format!("/projects/{}/reopen", id), None)?;
        read_project(tx, &id)
    })
}

#[tauri::command]
pub fn cancel_project(db: tauri::State<'_, Database>, id: String) -> Result<Project, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();
        tx.execute(
            "UPDATE projects SET status = 2, completedAt = ?1, updatedAt = ?1 WHERE id = ?2",
            rusqlite::params![now, id],
        )
        .map_err(|e| e.to_string())?;
        queue_pending_op(tx, "POST", &format!("/projects/{}/cancel", id), None)?;
        read_project(tx, &id)
    })
}

pub(crate) fn delete_project_impl(conn: &rusqlite::Connection, id: &str) -> Result<(), String> {
    // Nullify projectId on tasks in this project
    conn.execute(
        "UPDATE tasks SET projectId = NULL, sectionId = NULL WHERE projectId = ?1",
        rusqlite::params![id],
    )
    .map_err(|e| e.to_string())?;

    // Delete sections belonging to this project
    conn.execute(
        "DELETE FROM sections WHERE projectId = ?1",
        rusqlite::params![id],
    )
    .map_err(|e| e.to_string())?;

    conn.execute("DELETE FROM projects WHERE id = ?1", rusqlite::params![id])
        .map_err(|e| e.to_string())?;

    Ok(())
}

#[tauri::command]
pub fn delete_project(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        delete_project_impl(tx, &id)?;
        queue_pending_op(tx, "DELETE", &format!("/projects/{}", id), None)?;
        Ok(())
    })
}

#[tauri::command]
pub fn move_project_to_area(
    db: tauri::State<'_, Database>,
    project_id: String,
    area_id: Option<String>,
) -> Result<Project, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE projects SET areaId = ?1, updatedAt = ?2 WHERE id = ?3",
            rusqlite::params![area_id, now, project_id],
        )
        .map_err(|e| e.to_string())?;

        let area_json = match &area_id {
            Some(aid) => serde_json::json!({"id": aid}).to_string(),
            None => serde_json::json!({"id": null}).to_string(),
        };
        queue_pending_op(
            tx,
            "PUT",
            &format!("/projects/{}/area", project_id),
            Some(&area_json),
        )?;

        read_project(tx, &project_id)
    })
}

#[tauri::command]
pub fn reorder_projects(
    db: tauri::State<'_, Database>,
    moves: Vec<ReorderMove>,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        for m in &moves {
            tx.execute(
                "UPDATE projects SET \"index\" = ?1, updatedAt = ?2 WHERE id = ?3",
                rusqlite::params![m.index, Utc::now().to_rfc3339(), m.id],
            )
            .map_err(|e| e.to_string())?;
            let index_json = serde_json::json!({"index": m.index}).to_string();
            queue_pending_op(
                tx,
                "PUT",
                &format!("/projects/{}/reorder", m.id),
                Some(&index_json),
            )?;
        }
        Ok(())
    })
}

// --- Area Commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateAreaParams {
    pub title: String,
}

#[tauri::command]
pub fn create_area(
    db: tauri::State<'_, Database>,
    params: CreateAreaParams,
) -> Result<Area, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let id = uuid::Uuid::new_v4().to_string();
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "INSERT INTO areas (id, title, \"index\", archived, createdAt, updatedAt) VALUES (?1, ?2, 0, 0, ?3, ?3)",
            rusqlite::params![id, params.title, now],
        )
        .map_err(|e| e.to_string())?;

        let area = read_area(tx, &id)?;
        let body = serde_json::json!({"id": area.id, "title": area.title}).to_string();
        queue_pending_op(tx, "POST", "/areas", Some(&body))?;
        Ok(area)
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateAreaParams {
    pub id: String,
    pub title: String,
}

#[tauri::command]
pub fn update_area(
    db: tauri::State<'_, Database>,
    params: UpdateAreaParams,
) -> Result<Area, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE areas SET title = ?1, updatedAt = ?2 WHERE id = ?3",
            rusqlite::params![params.title, now, params.id],
        )
        .map_err(|e| e.to_string())?;

        let area = read_area(tx, &params.id)?;
        let body = area_patch_body(&area);
        queue_pending_op(tx, "PATCH", &format!("/areas/{}", area.id), Some(&body))?;
        Ok(area)
    })
}

#[tauri::command]
pub fn delete_area(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        // Nullify areaId on projects and tasks in this area
        tx.execute(
            "UPDATE projects SET areaId = NULL WHERE areaId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        tx.execute(
            "UPDATE tasks SET areaId = NULL WHERE areaId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        tx.execute("DELETE FROM areas WHERE id = ?1", rusqlite::params![id])
            .map_err(|e| e.to_string())?;

        queue_pending_op(tx, "DELETE", &format!("/areas/{}", id), None)?;

        Ok(())
    })
}

#[tauri::command]
pub fn toggle_area_archived(db: tauri::State<'_, Database>, id: String) -> Result<Area, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE areas SET archived = CASE WHEN archived = 0 THEN 1 ELSE 0 END, updatedAt = ?1 WHERE id = ?2",
            rusqlite::params![now, id],
        )
        .map_err(|e| e.to_string())?;

        let area = read_area(tx, &id)?;
        let action = if area.archived {
            "archive"
        } else {
            "unarchive"
        };
        queue_pending_op(tx, "POST", &format!("/areas/{}/{}", id, action), None)?;
        Ok(area)
    })
}

#[tauri::command]
pub fn reorder_areas(
    db: tauri::State<'_, Database>,
    moves: Vec<ReorderMove>,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        for m in &moves {
            tx.execute(
                "UPDATE areas SET \"index\" = ?1, updatedAt = ?2 WHERE id = ?3",
                rusqlite::params![m.index, Utc::now().to_rfc3339(), m.id],
            )
            .map_err(|e| e.to_string())?;
            let index_json = serde_json::json!({"index": m.index}).to_string();
            queue_pending_op(
                tx,
                "PUT",
                &format!("/areas/{}/reorder", m.id),
                Some(&index_json),
            )?;
        }
        Ok(())
    })
}

// --- Section Commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateSectionParams {
    pub title: String,
    pub project_id: String,
}

pub(crate) fn create_section_impl(
    conn: &rusqlite::Connection,
    params: CreateSectionParams,
) -> Result<Section, String> {
    let id = uuid::Uuid::new_v4().to_string();
    let now = Utc::now().to_rfc3339();

    conn.execute(
        "INSERT INTO sections (id, title, projectId, \"index\", archived, collapsed, createdAt, updatedAt) VALUES (?1, ?2, ?3, 0, 0, 0, ?4, ?4)",
        rusqlite::params![id, params.title, params.project_id, now],
    )
    .map_err(|e| e.to_string())?;

    read_section(conn, &id)
}

#[tauri::command]
pub fn create_section(
    db: tauri::State<'_, Database>,
    params: CreateSectionParams,
) -> Result<Section, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let project_id = params.project_id.clone();
        let section = create_section_impl(tx, params)?;
        let body = serde_json::json!({"id": section.id, "title": section.title}).to_string();
        queue_pending_op(
            tx,
            "POST",
            &format!("/projects/{}/sections", project_id),
            Some(&body),
        )?;
        Ok(section)
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateSectionParams {
    pub id: String,
    pub title: Option<String>,
}

#[tauri::command]
pub fn update_section(
    db: tauri::State<'_, Database>,
    params: UpdateSectionParams,
) -> Result<Section, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let conn = tx;
        let now = Utc::now().to_rfc3339();

    let mut sets: Vec<String> = Vec::new();
    let mut values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

    if let Some(v) = params.title {
        sets.push(format!("title = ?{}", sets.len() + 1));
        values.push(Box::new(v));
    }

    sets.push(format!("updatedAt = ?{}", sets.len() + 1));
    values.push(Box::new(now));

    let id_param_idx = values.len() + 1;
    values.push(Box::new(params.id.clone()));

    let sql = format!(
        "UPDATE sections SET {} WHERE id = ?{}",
        sets.join(", "),
        id_param_idx
    );

    let params_refs: Vec<&dyn rusqlite::types::ToSql> = values.iter().map(|v| v.as_ref()).collect();
    conn.execute(&sql, params_refs.as_slice())
        .map_err(|e| e.to_string())?;

        let section = read_section(conn, &params.id)?;
        let body = serde_json::json!({"title": section.title}).to_string();
        queue_pending_op(
            conn,
            "PUT",
            &format!("/projects/{}/sections/{}", section.project_id, section.id),
            Some(&body),
        )?;
        Ok(section)
    })
}

#[tauri::command]
pub fn delete_section(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let section = read_section(tx, &id)?;

    // Nullify sectionId on tasks in this section
        tx.execute(
            "UPDATE tasks SET sectionId = NULL WHERE sectionId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        tx.execute("DELETE FROM sections WHERE id = ?1", rusqlite::params![id])
            .map_err(|e| e.to_string())?;

        queue_pending_op(
            tx,
            "DELETE",
            &format!("/projects/{}/sections/{}", section.project_id, id),
            None,
        )?;

        Ok(())
    })
}

pub(crate) fn toggle_section_collapsed_impl(
    conn: &rusqlite::Connection,
    id: &str,
) -> Result<Section, String> {
    let now = Utc::now().to_rfc3339();

    conn.execute(
        "UPDATE sections SET collapsed = CASE WHEN collapsed = 0 THEN 1 ELSE 0 END, updatedAt = ?1 WHERE id = ?2",
        rusqlite::params![now, id],
    )
    .map_err(|e| e.to_string())?;

    read_section(conn, id)
}

#[tauri::command]
pub fn toggle_section_collapsed(
    db: tauri::State<'_, Database>,
    id: String,
) -> Result<Section, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    // Collapsed is local-only UI state, no pending op needed
    toggle_section_collapsed_impl(&conn, &id)
}

#[tauri::command]
pub fn toggle_section_archived(
    db: tauri::State<'_, Database>,
    id: String,
) -> Result<Section, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE sections SET archived = CASE WHEN archived = 0 THEN 1 ELSE 0 END, updatedAt = ?1 WHERE id = ?2",
            rusqlite::params![now, id],
        )
        .map_err(|e| e.to_string())?;

        read_section(tx, &id)
    })
}

#[tauri::command]
pub fn reorder_sections(
    db: tauri::State<'_, Database>,
    project_id: String,
    moves: Vec<ReorderMove>,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        for m in &moves {
            tx.execute(
                "UPDATE sections SET \"index\" = ?1, updatedAt = ?2 WHERE id = ?3 AND projectId = ?4",
                rusqlite::params![m.index, Utc::now().to_rfc3339(), m.id, project_id],
            )
            .map_err(|e| e.to_string())?;
            let index_json = serde_json::json!({"index": m.index}).to_string();
            queue_pending_op(
                tx,
                "PUT",
                &format!("/projects/{}/sections/{}/reorder", project_id, m.id),
                Some(&index_json),
            )?;
        }
        Ok(())
    })
}

// --- Tag Commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateTagParams {
    pub title: String,
}

pub(crate) fn create_tag_impl(
    conn: &rusqlite::Connection,
    params: CreateTagParams,
) -> Result<Option<Tag>, String> {
    let id = uuid::Uuid::new_v4().to_string();
    let now = Utc::now().to_rfc3339();

    let result = conn.execute(
        "INSERT INTO tags (id, title, \"index\", createdAt, updatedAt) VALUES (?1, ?2, 0, ?3, ?3)",
        rusqlite::params![id, params.title, now],
    );

    match result {
        Ok(_) => Ok(Some(read_tag(conn, &id)?)),
        Err(rusqlite::Error::SqliteFailure(err, _))
            if err.code == rusqlite::ErrorCode::ConstraintViolation =>
        {
            Err(format!("Tag with title '{}' already exists", params.title))
        }
        Err(e) => Err(e.to_string()),
    }
}

#[tauri::command]
pub fn create_tag(
    db: tauri::State<'_, Database>,
    params: CreateTagParams,
) -> Result<Option<Tag>, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
    let tag = create_tag_impl(tx, params)?;
    if let Some(ref t) = tag {
        let body = serde_json::json!({"id": t.id, "title": t.title}).to_string();
        queue_pending_op(tx, "POST", "/tags", Some(&body))?;
    }
    Ok(tag)
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateTagParams {
    pub id: String,
    pub title: String,
}

#[tauri::command]
pub fn update_tag(db: tauri::State<'_, Database>, params: UpdateTagParams) -> Result<Tag, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
    let now = Utc::now().to_rfc3339();

    tx.execute(
        "UPDATE tags SET title = ?1, updatedAt = ?2 WHERE id = ?3",
        rusqlite::params![params.title, now, params.id],
    )
    .map_err(|e| e.to_string())?;

    let tag = read_tag(tx, &params.id)?;
    let body = serde_json::json!({"title": tag.title}).to_string();
    queue_pending_op(tx, "PUT", &format!("/tags/{}", tag.id), Some(&body))?;
    Ok(tag)
    })
}

#[tauri::command]
pub fn delete_tag(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
    tx.execute(
        "DELETE FROM taskTags WHERE tagId = ?1",
        rusqlite::params![id],
    )
    .map_err(|e| e.to_string())?;

    tx.execute("DELETE FROM tags WHERE id = ?1", rusqlite::params![id])
        .map_err(|e| e.to_string())?;
    queue_pending_op(tx, "DELETE", &format!("/tags/{}", id), None)?;

    Ok(())
    })
}

#[tauri::command]
pub fn add_tag_to_task(
    db: tauri::State<'_, Database>,
    task_id: String,
    tag_id: String,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {

    tx.execute(
        "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?1, ?2)",
        rusqlite::params![task_id, tag_id],
    )
    .map_err(|e| e.to_string())?;

    queue_pending_op(
        tx,
        "POST",
        &format!("/tasks/{}/tags/{}", task_id, tag_id),
        None,
    )?;

    Ok(())
    })
}

#[tauri::command]
pub fn remove_tag_from_task(
    db: tauri::State<'_, Database>,
    task_id: String,
    tag_id: String,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {

    tx.execute(
        "DELETE FROM taskTags WHERE taskId = ?1 AND tagId = ?2",
        rusqlite::params![task_id, tag_id],
    )
    .map_err(|e| e.to_string())?;

    queue_pending_op(
        tx,
        "DELETE",
        &format!("/tasks/{}/tags/{}", task_id, tag_id),
        None,
    )?;

    Ok(())
    })
}

#[tauri::command]
pub fn add_task_link(
    db: tauri::State<'_, Database>,
    task_id: String,
    linked_task_id: String,
) -> Result<(), String> {
    if task_id == linked_task_id {
        return Err("task cannot link to itself".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        // Insert both directions locally to match the server's bidirectional
        // storage; otherwise the local DB disagrees with what the delta pull
        // will bring back and $linksByTaskId flickers on next sync.
        tx.execute(
            "INSERT OR IGNORE INTO taskLinks (taskId, linkedTaskId) VALUES (?1, ?2)",
            rusqlite::params![task_id, linked_task_id],
        )
        .map_err(|e| e.to_string())?;
        tx.execute(
            "INSERT OR IGNORE INTO taskLinks (taskId, linkedTaskId) VALUES (?1, ?2)",
            rusqlite::params![linked_task_id, task_id],
        )
        .map_err(|e| e.to_string())?;

        queue_pending_op(
            tx,
            "POST",
            &format!("/tasks/{}/links/{}", task_id, linked_task_id),
            None,
        )?;

        Ok(())
    })
}

#[tauri::command]
pub fn remove_task_link(
    db: tauri::State<'_, Database>,
    task_id: String,
    linked_task_id: String,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        // Delete BOTH mirrored rows so an immediate $linksByTaskId read does
        // not still show the reverse direction lingering.
        tx.execute(
            "DELETE FROM taskLinks WHERE (taskId = ?1 AND linkedTaskId = ?2) OR (taskId = ?2 AND linkedTaskId = ?1)",
            rusqlite::params![task_id, linked_task_id],
        )
        .map_err(|e| e.to_string())?;

        queue_pending_op(
            tx,
            "DELETE",
            &format!("/tasks/{}/links/{}", task_id, linked_task_id),
            None,
        )?;

        Ok(())
    })
}

#[tauri::command]
pub fn add_tag_to_project(
    db: tauri::State<'_, Database>,
    project_id: String,
    tag_id: String,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {

    tx.execute(
        "INSERT OR IGNORE INTO projectTags (projectId, tagId) VALUES (?1, ?2)",
        rusqlite::params![project_id, tag_id],
    )
    .map_err(|e| e.to_string())?;

    queue_pending_op(
        tx,
        "POST",
        &format!("/projects/{}/tags/{}", project_id, tag_id),
        None,
    )?;

    Ok(())
    })
}

#[tauri::command]
pub fn remove_tag_from_project(
    db: tauri::State<'_, Database>,
    project_id: String,
    tag_id: String,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {

    tx.execute(
        "DELETE FROM projectTags WHERE projectId = ?1 AND tagId = ?2",
        rusqlite::params![project_id, tag_id],
    )
    .map_err(|e| e.to_string())?;

    queue_pending_op(
        tx,
        "DELETE",
        &format!("/projects/{}/tags/{}", project_id, tag_id),
        None,
    )?;

    Ok(())
    })
}

// --- Checklist Commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateChecklistItemParams {
    pub title: String,
    pub task_id: String,
}

pub(crate) fn create_checklist_item_impl(
    conn: &rusqlite::Connection,
    params: CreateChecklistItemParams,
) -> Result<ChecklistItem, String> {
    let id = uuid::Uuid::new_v4().to_string();
    let now = Utc::now().to_rfc3339();

    conn.execute(
        "INSERT INTO checklistItems (id, title, status, taskId, \"index\", createdAt, updatedAt) VALUES (?1, ?2, 0, ?3, 0, ?4, ?4)",
        rusqlite::params![id, params.title, params.task_id, now],
    )
    .map_err(|e| e.to_string())?;

    read_checklist_item(conn, &id)
}

#[tauri::command]
pub fn create_checklist_item(
    db: tauri::State<'_, Database>,
    params: CreateChecklistItemParams,
) -> Result<ChecklistItem, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let task_id = params.task_id.clone();
        let item = create_checklist_item_impl(tx, params)?;
        let body = serde_json::json!({"title": item.title}).to_string();
        queue_pending_op(
            tx,
            "POST",
            &format!("/tasks/{}/checklist", task_id),
            Some(&body),
        )?;
        Ok(item)
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateChecklistItemParams {
    pub id: String,
    pub title: String,
}

#[tauri::command]
pub fn update_checklist_item(
    db: tauri::State<'_, Database>,
    params: UpdateChecklistItemParams,
) -> Result<ChecklistItem, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let now = Utc::now().to_rfc3339();
        tx.execute(
            "UPDATE checklistItems SET title = ?1, updatedAt = ?2 WHERE id = ?3",
            rusqlite::params![params.title, now, params.id],
        )
        .map_err(|e| e.to_string())?;
        let item = read_checklist_item(tx, &params.id)?;
        let body = serde_json::json!({"title": item.title}).to_string();
        queue_pending_op(
            tx,
            "PUT",
            &format!("/tasks/{}/checklist/{}", item.task_id, item.id),
            Some(&body),
        )?;
        Ok(item)
    })
}

pub(crate) fn toggle_checklist_item_impl(
    conn: &rusqlite::Connection,
    id: &str,
) -> Result<ChecklistItem, String> {
    let now = Utc::now().to_rfc3339();

    conn.execute(
        "UPDATE checklistItems SET status = CASE WHEN status = 0 THEN 1 ELSE 0 END, updatedAt = ?1 WHERE id = ?2",
        rusqlite::params![now, id],
    )
    .map_err(|e| e.to_string())?;

    read_checklist_item(conn, id)
}

#[tauri::command]
pub fn toggle_checklist_item(
    db: tauri::State<'_, Database>,
    id: String,
) -> Result<ChecklistItem, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let item = toggle_checklist_item_impl(tx, &id)?;
        let action = if item.status == 1 {
            "complete"
        } else {
            "uncomplete"
        };
        queue_pending_op(
            tx,
            "POST",
            &format!("/tasks/{}/checklist/{}/{}", item.task_id, item.id, action),
            None,
        )?;
        Ok(item)
    })
}

pub(crate) fn delete_checklist_item_impl(
    conn: &rusqlite::Connection,
    id: &str,
) -> Result<(), String> {
    conn.execute(
        "DELETE FROM checklistItems WHERE id = ?1",
        rusqlite::params![id],
    )
    .map_err(|e| e.to_string())?;

    Ok(())
}

#[tauri::command]
pub fn delete_checklist_item(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        let item = read_checklist_item(tx, &id)?;
        delete_checklist_item_impl(tx, &id)?;
        queue_pending_op(
            tx,
            "DELETE",
            &format!("/tasks/{}/checklist/{}", item.task_id, item.id),
            None,
        )?;
        Ok(())
    })
}

#[tauri::command]
pub fn reorder_checklist_items(
    db: tauri::State<'_, Database>,
    task_id: String,
    moves: Vec<ReorderMove>,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        for m in &moves {
            tx.execute(
                "UPDATE checklistItems SET \"index\" = ?1, updatedAt = ?2 WHERE id = ?3 AND taskId = ?4",
                rusqlite::params![m.index, Utc::now().to_rfc3339(), m.id, task_id],
            )
            .map_err(|e| e.to_string())?;
            let index_json = serde_json::json!({"index": m.index}).to_string();
            queue_pending_op(
                tx,
                "PUT",
                &format!("/tasks/{}/checklist/{}/reorder", task_id, m.id),
                Some(&index_json),
            )?;
        }
        Ok(())
    })
}

// --- Settings commands ---

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Settings {
    pub server_url: String,
    pub api_key: String,
    pub sync_enabled: bool,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateSettingsParams {
    pub server_url: Option<String>,
    pub api_key: Option<String>,
    pub sync_enabled: Option<bool>,
}

#[tauri::command]
pub fn get_settings(db: tauri::State<'_, Database>) -> Result<Settings, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;

    let server_url = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'server_url'",
            [],
            |row| row.get::<_, String>(0),
        )
        .unwrap_or_default();

    let api_key = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'api_key'",
            [],
            |row| row.get::<_, String>(0),
        )
        .unwrap_or_default();

    let sync_enabled = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'sync_enabled'",
            [],
            |row| row.get::<_, String>(0),
        )
        .unwrap_or_else(|_| "false".to_string())
        == "true";

    Ok(Settings {
        server_url,
        api_key,
        sync_enabled,
    })
}

#[tauri::command]
pub fn update_settings(
    db: tauri::State<'_, Database>,
    params: UpdateSettingsParams,
) -> Result<Settings, String> {
    {
        let conn = db.conn.lock().map_err(|e| e.to_string())?;

        if let Some(url) = &params.server_url {
            conn.execute(
                "INSERT INTO settings (key, value) VALUES ('server_url', ?1) ON CONFLICT(key) DO UPDATE SET value = ?1",
                [url],
            ).map_err(|e| e.to_string())?;
        }

        if let Some(key) = &params.api_key {
            conn.execute(
                "INSERT INTO settings (key, value) VALUES ('api_key', ?1) ON CONFLICT(key) DO UPDATE SET value = ?1",
                [key],
            ).map_err(|e| e.to_string())?;
        }

        if let Some(enabled) = params.sync_enabled {
            let val = if enabled { "true" } else { "false" };
            conn.execute(
                "INSERT INTO settings (key, value) VALUES ('sync_enabled', ?1) ON CONFLICT(key) DO UPDATE SET value = ?1",
                [val],
            ).map_err(|e| e.to_string())?;
        }
    }

    get_settings(db)
}

#[tauri::command]
pub fn get_task_activities(db: tauri::State<'_, Database>, task_id: String) -> Result<Vec<Activity>, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    let mut stmt = conn
        .prepare("SELECT id, taskId, actorId, actorType, type, content, createdAt FROM activities WHERE taskId = ?1 ORDER BY createdAt ASC")
        .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map([&task_id], |row| {
            Ok(Activity {
                id: row.get(0)?,
                task_id: row.get(1)?,
                actor_id: row.get(2)?,
                actor_type: row.get(3)?,
                activity_type: row.get(4)?,
                content: row.get(5)?,
                created_at: row.get(6)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>().map_err(|e| e.to_string())
}

#[tauri::command]
pub fn create_activity(
    db: tauri::State<'_, Database>,
    task_id: String,
    actor_type: String,
    activity_type: String,
    content: String,
) -> Result<Activity, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let id = uuid::Uuid::new_v4().to_string();
        let now = chrono::Utc::now().to_rfc3339();

        tx.execute(
            "INSERT INTO activities (id, taskId, actorId, actorType, type, content, createdAt) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
            rusqlite::params![id, task_id, "local", actor_type, activity_type, content, now],
        )
        .map_err(|e| e.to_string())?;

        let body = serde_json::json!({
            "actor_type": actor_type,
            "type": activity_type,
            "content": content,
        })
        .to_string();
        queue_pending_op(tx, "POST", &format!("/tasks/{}/activity", task_id), Some(&body))?;

        Ok(Activity {
            id,
            task_id,
            actor_id: Some("local".to_string()),
            actor_type,
            activity_type,
            content,
            created_at: now,
        })
    })
}

// --- Location Commands ---

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateLocationParams {
    pub name: String,
}

#[tauri::command]
pub fn create_location(
    db: tauri::State<'_, Database>,
    params: CreateLocationParams,
) -> Result<Location, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let id = uuid::Uuid::new_v4().to_string();
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "INSERT INTO locations (id, name, createdAt, updatedAt) VALUES (?1, ?2, ?3, ?3)",
            rusqlite::params![id, params.name, now],
        )
        .map_err(|e| e.to_string())?;

        let loc = read_location(tx, &id)?;
        let body = serde_json::json!({"id": loc.id, "name": loc.name}).to_string();
        queue_pending_op(tx, "POST", "/locations", Some(&body))?;
        Ok(loc)
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateLocationParams {
    pub id: String,
    pub name: String,
}

#[tauri::command]
pub fn update_location(
    db: tauri::State<'_, Database>,
    params: UpdateLocationParams,
) -> Result<Location, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE locations SET name = ?1, updatedAt = ?2 WHERE id = ?3",
            rusqlite::params![params.name, now, params.id],
        )
        .map_err(|e| e.to_string())?;

        let loc = read_location(tx, &params.id)?;
        let body = serde_json::json!({"name": loc.name}).to_string();
        queue_pending_op(tx, "PUT", &format!("/locations/{}", loc.id), Some(&body))?;
        Ok(loc)
    })
}

#[tauri::command]
pub fn delete_location(db: tauri::State<'_, Database>, id: String) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, |tx| {
        // Clear locationId on tasks that reference this location (must be
        // in the same tx so we don't leave dangling references if the delete
        // or the pending-op enqueue fails partway).
        tx.execute(
            "UPDATE tasks SET locationId = NULL WHERE locationId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        tx.execute("DELETE FROM locations WHERE id = ?1", rusqlite::params![id])
            .map_err(|e| e.to_string())?;

        queue_pending_op(tx, "DELETE", &format!("/locations/{}", id), None)?;

        Ok(())
    })
}

#[tauri::command]
pub fn set_task_location(
    db: tauri::State<'_, Database>,
    task_id: String,
    location_id: Option<String>,
) -> Result<Task, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    with_tx(&conn, move |tx| {
        let now = Utc::now().to_rfc3339();

        tx.execute(
            "UPDATE tasks SET locationId = ?1, updatedAt = ?2 WHERE id = ?3",
            rusqlite::params![location_id, now, task_id],
        )
        .map_err(|e| e.to_string())?;

        let task = query_task(tx, &task_id)?;
        let body = serde_json::json!({"id": location_id}).to_string();
        queue_pending_op(
            tx,
            "PUT",
            &format!("/tasks/{}/location", task_id),
            Some(&body),
        )?;
        Ok(task)
    })
}

#[tauri::command]
pub fn reset_database(db: tauri::State<'_, Database>) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    conn.execute_batch(
        "DELETE FROM activities;
         DELETE FROM checklistItems;
         DELETE FROM taskTags;
         DELETE FROM taskLinks;
         DELETE FROM projectTags;
         DELETE FROM tasks;
         DELETE FROM sections;
         DELETE FROM projects;
         DELETE FROM areas;
         DELETE FROM tags;
         DELETE FROM locations;
         DELETE FROM pendingOps;"
    )
    .map_err(|e| e.to_string())
}
