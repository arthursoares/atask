use rusqlite::Connection;
use serde::Deserialize;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tauri::Emitter;

// --- Types ---

#[derive(Debug)]
struct PendingOp {
    id: i64,
    method: String,
    path: String,
    body: Option<String>,
}

#[derive(Debug, Clone)]
pub(crate) struct SyncConfig {
    pub server_url: String,
    pub api_key: String,
}

// Go serializes sql.NullString as {"String":"...","Valid":true}
#[derive(Debug, Deserialize)]
struct NullString {
    #[serde(rename = "String")]
    string: String,
    #[allow(dead_code)]
    #[serde(rename = "Valid")]
    valid: bool,
}

#[derive(Debug, Deserialize)]
struct NullInt64 {
    #[serde(rename = "Int64")]
    int64: i64,
    #[allow(dead_code)]
    #[serde(rename = "Valid")]
    valid: bool,
}

#[derive(Debug, Deserialize)]
struct DeltaEvent {
    id: i64,
    entity_type: NullString,
    entity_id: NullString,
    action: NullInt64, // 0=created, 1=modified, 2=deleted
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum DeltaDisposition {
    Fetch,
    Delete,
}

// --- Settings helpers ---

pub(crate) fn read_sync_config(conn: &Connection) -> Option<SyncConfig> {
    let get = |key: &str| -> Option<String> {
        conn.query_row("SELECT value FROM settings WHERE key = ?1", [key], |row| {
            row.get(0)
        })
        .ok()
    };
    if get("sync_enabled")? != "true" {
        return None;
    }
    let server_url = get("server_url").filter(|s| !s.is_empty())?;
    let api_key = get("api_key").filter(|s| !s.is_empty())?;
    Some(SyncConfig {
        server_url,
        api_key,
    })
}

fn get_sync_cursor(conn: &Connection) -> i64 {
    conn.query_row(
        "SELECT value FROM settings WHERE key = 'sync_cursor'",
        [],
        |row| {
            let s: String = row.get(0)?;
            Ok(s.parse::<i64>().unwrap_or(0))
        },
    )
    .unwrap_or(0)
}

fn set_sync_cursor(conn: &Connection, cursor: i64) -> Result<(), String> {
    conn.execute(
        "INSERT INTO settings (key, value) VALUES ('sync_cursor', ?1) ON CONFLICT(key) DO UPDATE SET value = ?1",
        [cursor.to_string()],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

fn set_last_sync_at(conn: &Connection) -> Result<(), String> {
    let now = chrono::Utc::now().to_rfc3339();
    conn.execute(
        "INSERT INTO settings (key, value) VALUES ('last_sync_at', ?1) ON CONFLICT(key) DO UPDATE SET value = ?1",
        [&now],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

fn set_last_sync_error(conn: &Connection, err: Option<&str>) -> Result<(), String> {
    match err {
        Some(msg) => conn.execute(
                "INSERT INTO settings (key, value) VALUES ('last_sync_error', ?1) ON CONFLICT(key) DO UPDATE SET value = ?1",
                [msg],
            )
            .map(|_| ())
            .map_err(|e| e.to_string()),
        None => conn.execute("DELETE FROM settings WHERE key = 'last_sync_error'", [])
            .map(|_| ())
            .map_err(|e| e.to_string()),
    }
}

// --- Pending ops: outbound flush ---

fn read_pending_ops(conn: &Connection, limit: i64) -> Result<Vec<PendingOp>, String> {
    let mut stmt = conn.prepare(
        "SELECT id, method, path, body FROM pendingOps WHERE synced = 0 ORDER BY id ASC LIMIT ?1",
    )
    .map_err(|e| e.to_string())?;
    let rows = stmt
        .query_map(rusqlite::params![limit], |row| {
            Ok(PendingOp {
                id: row.get(0)?,
                method: row.get(1)?,
                path: row.get(2)?,
                body: row.get(3)?,
            })
        })
        .map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())
}

fn mark_synced(conn: &Connection, id: i64) -> Result<(), String> {
    conn.execute(
        "UPDATE pendingOps SET synced = 1 WHERE id = ?1",
        rusqlite::params![id],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

fn plan_delta_applications(
    deltas: &[DeltaEvent],
    cursor: i64,
) -> (i64, Vec<(String, String)>, Vec<(String, String)>) {
    let mut max_cursor = cursor;
    let mut latest_by_entity: HashMap<(String, String), DeltaDisposition> = HashMap::new();

    for delta in deltas {
        if delta.id > max_cursor {
            max_cursor = delta.id;
        }

        let disposition = if delta.action.int64 == 2 {
            DeltaDisposition::Delete
        } else {
            DeltaDisposition::Fetch
        };

        latest_by_entity.insert(
            (
                delta.entity_type.string.clone(),
                delta.entity_id.string.clone(),
            ),
            disposition,
        );
    }

    let mut to_fetch = Vec::new();
    let mut to_delete = Vec::new();
    for ((entity_type, entity_id), disposition) in latest_by_entity {
        match disposition {
            DeltaDisposition::Fetch => to_fetch.push((entity_type, entity_id)),
            DeltaDisposition::Delete => to_delete.push((entity_type, entity_id)),
        }
    }

    to_fetch.sort();
    to_delete.sort();

    (max_cursor, to_fetch, to_delete)
}

fn delete_entity(conn: &Connection, entity_type: &str, entity_id: &str) -> Result<(), String> {
    let table = match entity_type {
        "task" => "tasks",
        "project" => "projects",
        "area" => "areas",
        "section" => "sections",
        "tag" => "tags",
        "checklist_item" => "checklistItems",
        "activity" => "activities",
        _ => return Ok(()),
    };

    conn.execute(&format!("DELETE FROM {} WHERE id = ?1", table), [entity_id])
        .map(|_| ())
        .map_err(|e| e.to_string())
}

fn flush_pending_ops_blocking(
    client: &reqwest::blocking::Client,
    conn: &Mutex<Connection>,
    config: &SyncConfig,
) -> Result<usize, String> {
    let ops = {
        let c = conn.lock().map_err(|e| e.to_string())?;
        read_pending_ops(&c, 50)?
    };

    if ops.is_empty() {
        return Ok(0);
    }

    let mut flushed = 0;
    let mut consecutive_failures = 0u32;

    for op in &ops {
        let url = format!("{}{}", config.server_url, op.path);
        let mut req = match op.method.as_str() {
            "POST" => client.post(&url),
            "PUT" => client.put(&url),
            "DELETE" => client.delete(&url),
            "PATCH" => client.patch(&url),
            _ => client.post(&url),
        };
        req = req.header("Authorization", format!("ApiKey {}", config.api_key));
        if let Some(body) = &op.body {
            req = req
                .header("Content-Type", "application/json")
                .body(body.clone());
        }

        match req.send() {
            Ok(resp) => {
                let status = resp.status().as_u16();
                let c = conn.lock().map_err(|e| e.to_string())?;
                if (200..300).contains(&status) || (400..500).contains(&status) {
                    // Success or client error (conflict lost) — mark done
                    mark_synced(&c, op.id)?;
                    flushed += 1;
                    consecutive_failures = 0;
                } else {
                    consecutive_failures += 1;
                    if consecutive_failures >= 3 {
                        break;
                    }
                }
            }
            Err(_) => {
                consecutive_failures += 1;
                if consecutive_failures >= 3 {
                    break;
                }
            }
        }
    }

    Ok(flushed)
}

// --- Delta pull: inbound sync ---

fn pull_deltas_blocking(
    client: &reqwest::blocking::Client,
    conn: &Mutex<Connection>,
    config: &SyncConfig,
) -> Result<usize, String> {
    let cursor = {
        let c = conn.lock().map_err(|e| e.to_string())?;
        get_sync_cursor(&c)
    };

    let url = format!("{}/sync/deltas?since={}", config.server_url, cursor);
    let resp = client
        .get(&url)
        .header("Authorization", format!("ApiKey {}", config.api_key))
        .send()
        .map_err(|e| e.to_string())?;

    if !resp.status().is_success() {
        return Err(format!("delta pull failed: {}", resp.status()));
    }

    let deltas: Vec<DeltaEvent> = resp.json().map_err(|e| e.to_string())?;
    if deltas.is_empty() {
        return Ok(0);
    }

    let (max_cursor, to_fetch, to_delete) = plan_delta_applications(&deltas, cursor);

    let applied = to_fetch.len() + to_delete.len();

    // Apply deletes
    {
        let c = conn.lock().map_err(|e| e.to_string())?;
        for (entity_type, entity_id) in &to_delete {
            delete_entity(&c, entity_type, entity_id)?;
        }
    }

    // Fetch and upsert entities
    for (entity_type, entity_id) in &to_fetch {
        let plural = match entity_type.as_str() {
            "task" => "tasks",
            "project" => "projects",
            "area" => "areas",
            "section" => "sections",
            "tag" => "tags",
            "activity" => "activities",
            _ => continue,
        };

        let url = format!("{}/{}/{}", config.server_url, plural, entity_id);
        let resp = client
            .get(&url)
            .header("Authorization", format!("ApiKey {}", config.api_key))
            .send();

        let resp = resp.map_err(|e| e.to_string())?;
        if !resp.status().is_success() {
            eprintln!(
                "entity fetch failed for {entity_type}/{entity_id}: {} — skipping",
                resp.status()
            );
            continue;
        }

        let json = resp
            .json::<serde_json::Value>()
            .map_err(|e| e.to_string())?;
        let c = conn.lock().map_err(|e| e.to_string())?;
        match entity_type.as_str() {
            "task" => upsert_task(&c, &json)?,
            "project" => upsert_project(&c, &json)?,
            "area" => upsert_area(&c, &json)?,
            "section" => upsert_section(&c, &json)?,
            "tag" => upsert_tag(&c, &json)?,
            "activity" => upsert_activity(&c, &json)?,
            _ => {}
        }
    }

    // Update cursor
    {
        let c = conn.lock().map_err(|e| e.to_string())?;
        set_sync_cursor(&c, max_cursor)?;
    }

    Ok(applied)
}

// --- Public sync_now: flush + pull ---

/// Perform a full sync cycle: flush pending ops then pull deltas.
/// Called from Tauri commands (sync_now, after settings save, etc.)
pub fn sync_now_blocking(
    conn: &Arc<Mutex<Connection>>,
    app_handle: &tauri::AppHandle,
) -> Result<(), String> {
    let config = {
        let c = conn.lock().map_err(|e| e.to_string())?;
        read_sync_config(&c)
    };
    let Some(config) = config else {
        return Ok(()); // sync not enabled
    };

    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(15))
        .build()
        .map_err(|e| e.to_string())?;

    // 1. Flush outbound
    flush_pending_ops_blocking(&client, conn, &config)?;

    // 2. Pull inbound deltas
    let pulled = pull_deltas_blocking(&client, conn, &config)?;

    // 3. Update timestamps
    {
        let c = conn.lock().map_err(|e| e.to_string())?;
        set_last_sync_at(&c)?;
        set_last_sync_error(&c, None)?;
    }

    // 4. Notify React if we pulled changes
    if pulled > 0 {
        let _ = app_handle.emit("store-changed", ());
    }
    let _ = app_handle.emit("sync-flushed", ());

    Ok(())
}

// --- Background fallback timer ---

/// Spawns a background thread that syncs every 5 minutes as a safety net.
/// The primary sync is triggered by React on mutations, focus, and view changes.
pub fn spawn_sync_worker(conn: Arc<Mutex<Connection>>, app_handle: tauri::AppHandle) {
    std::thread::spawn(move || {
        loop {
            std::thread::sleep(std::time::Duration::from_secs(300)); // 5 minutes
            if let Err(e) = sync_now_blocking(&conn, &app_handle) {
                if let Ok(c) = conn.lock() {
                    let _ = set_last_sync_error(&c, Some(&e));
                }
                eprintln!("[sync] background sync error: {}", e);
            }
        }
    });
}

// --- Upsert functions ---

/// Upsert a task from server JSON (camelCase/PascalCase) into local DB (camelCase columns).
pub fn upsert_task(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    // Handle both camelCase (new Go) and PascalCase (old data) field names
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };
    let opt_s = |key: &str, alt: &str| -> Option<&str> {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| if v.is_null() { None } else { v.as_str() })
    };
    let i = |key: &str, alt: &str| -> i64 {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_i64())
            .unwrap_or(0)
    };
    let opt_i = |key: &str, alt: &str| -> Option<i64> {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| if v.is_null() { None } else { v.as_i64() })
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }

    let repeat_rule = j
        .get("repeatRule")
        .or_else(|| j.get("RecurrenceRule"))
        .and_then(|v| {
            if v.is_null() {
                None
            } else {
                Some(v.to_string())
            }
        });

    conn.execute(
        "INSERT INTO tasks (id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, createdAt, updatedAt, syncStatus, repeatRule) \
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,0,?17) \
         ON CONFLICT(id) DO UPDATE SET \
         title=?2, notes=?3, status=?4, schedule=?5, startDate=?6, deadline=?7, completedAt=?8, \"index\"=?9, todayIndex=?10, timeSlot=?11, projectId=?12, sectionId=?13, areaId=?14, updatedAt=?16, syncStatus=0, repeatRule=?17",
        rusqlite::params![
            id,
            s("title", "Title"),
            s("notes", "Notes"),
            i("status", "Status"),
            i("schedule", "Schedule"),
            opt_s("startDate", "StartDate"),
            opt_s("deadline", "Deadline"),
            opt_s("completedAt", "CompletedAt"),
            i("index", "Index"),
            opt_i("todayIndex", "TodayIndex"),
            opt_s("timeSlot", "TimeSlot"),
            opt_s("projectId", "ProjectID"),
            opt_s("sectionId", "SectionID"),
            opt_s("areaId", "AreaID"),
            s("createdAt", "CreatedAt"),
            s("updatedAt", "UpdatedAt"),
            repeat_rule,
        ],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

pub fn upsert_project(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };
    let opt_s = |key: &str, alt: &str| -> Option<&str> {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| if v.is_null() { None } else { v.as_str() })
    };
    let i = |key: &str, alt: &str| -> i64 {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_i64())
            .unwrap_or(0)
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }

    conn.execute(
        "INSERT INTO projects (id, title, notes, status, color, areaId, \"index\", completedAt, createdAt, updatedAt) \
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10) \
         ON CONFLICT(id) DO UPDATE SET \
         title=?2, notes=?3, status=?4, color=?5, areaId=?6, \"index\"=?7, completedAt=?8, updatedAt=?10",
        rusqlite::params![
            id, s("title", "Title"), s("notes", "Notes"), i("status", "Status"),
            s("color", "Color"), opt_s("areaId", "AreaID"), i("index", "Index"),
            opt_s("completedAt", "CompletedAt"), s("createdAt", "CreatedAt"), s("updatedAt", "UpdatedAt"),
        ],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

pub fn upsert_area(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };
    let i = |key: &str, alt: &str| -> i64 {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_i64())
            .unwrap_or(0)
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }
    let archived = if j
        .get("archived")
        .or_else(|| j.get("Archived"))
        .and_then(|v| v.as_bool())
        .unwrap_or(false)
    {
        1
    } else {
        0
    };

    conn.execute(
        "INSERT INTO areas (id, title, \"index\", archived, createdAt, updatedAt) \
         VALUES (?1,?2,?3,?4,?5,?6) \
         ON CONFLICT(id) DO UPDATE SET \
         title=?2, \"index\"=?3, archived=?4, updatedAt=?6",
        rusqlite::params![
            id,
            s("title", "Title"),
            i("index", "Index"),
            archived,
            s("createdAt", "CreatedAt"),
            s("updatedAt", "UpdatedAt")
        ],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

pub fn upsert_section(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };
    let i = |key: &str, alt: &str| -> i64 {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_i64())
            .unwrap_or(0)
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }
    let archived = if j
        .get("archived")
        .or_else(|| j.get("Archived"))
        .and_then(|v| v.as_bool())
        .unwrap_or(false)
    {
        1
    } else {
        0
    };
    let collapsed = if j
        .get("collapsed")
        .or_else(|| j.get("Collapsed"))
        .and_then(|v| v.as_bool())
        .unwrap_or(false)
    {
        1
    } else {
        0
    };

    conn.execute(
        "INSERT INTO sections (id, title, projectId, \"index\", archived, collapsed, createdAt, updatedAt) \
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8) \
         ON CONFLICT(id) DO UPDATE SET \
         title=?2, projectId=?3, \"index\"=?4, archived=?5, collapsed=?6, updatedAt=?8",
        rusqlite::params![id, s("title", "Title"), s("projectId", "ProjectID"), i("index", "Index"), archived, collapsed, s("createdAt", "CreatedAt"), s("updatedAt", "UpdatedAt")],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

pub fn upsert_activity(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }

    conn.execute(
        "INSERT INTO activities (id, taskId, actorId, actorType, type, content, createdAt) \
         VALUES (?1,?2,?3,?4,?5,?6,?7) \
         ON CONFLICT(id) DO UPDATE SET \
         taskId=?2, actorId=?3, actorType=?4, type=?5, content=?6, createdAt=?7",
        rusqlite::params![
            id,
            s("taskId", "TaskID"),
            s("actorId", "ActorID"),
            s("actorType", "ActorType"),
            s("type", "Type"),
            s("content", "Content"),
            s("createdAt", "CreatedAt")
        ],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

pub fn upsert_tag(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };
    let i = |key: &str, alt: &str| -> i64 {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_i64())
            .unwrap_or(0)
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }

    conn.execute(
        "INSERT INTO tags (id, title, \"index\", createdAt, updatedAt) \
         VALUES (?1,?2,?3,?4,?5) \
         ON CONFLICT(id) DO UPDATE SET \
         title=?2, \"index\"=?3, updatedAt=?5",
        rusqlite::params![
            id,
            s("title", "Title"),
            i("index", "Index"),
            s("createdAt", "CreatedAt"),
            s("updatedAt", "UpdatedAt")
        ],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use rusqlite::Connection;

    fn setup_test_conn() -> Connection {
        let conn = Connection::open_in_memory().expect("in-memory sqlite");
        conn.execute_batch(include_str!("migrations/001_schema.sql"))
            .expect("schema migration");
        conn.execute_batch(include_str!("migrations/002_settings.sql"))
            .expect("settings migration");
        conn.execute_batch(include_str!("migrations/003_activities.sql"))
            .expect("activities migration");
        conn
    }

    #[test]
    fn latest_delta_wins_when_entity_changes_multiple_times() {
        let deltas = vec![
            DeltaEvent {
                id: 10,
                entity_type: NullString {
                    string: "task".into(),
                    valid: true,
                },
                entity_id: NullString {
                    string: "task-1".into(),
                    valid: true,
                },
                action: NullInt64 {
                    int64: 1,
                    valid: true,
                },
            },
            DeltaEvent {
                id: 11,
                entity_type: NullString {
                    string: "task".into(),
                    valid: true,
                },
                entity_id: NullString {
                    string: "task-1".into(),
                    valid: true,
                },
                action: NullInt64 {
                    int64: 2,
                    valid: true,
                },
            },
        ];

        let (cursor, to_fetch, to_delete) = plan_delta_applications(&deltas, 5);

        assert_eq!(cursor, 11);
        assert!(to_fetch.is_empty());
        assert_eq!(to_delete, vec![("task".into(), "task-1".into())]);
    }

    #[test]
    fn pending_ops_can_be_read_and_marked_synced() {
        let conn = setup_test_conn();
        conn.execute(
            "INSERT INTO pendingOps (method, path, body, createdAt, synced) VALUES (?1, ?2, ?3, ?4, 0)",
            rusqlite::params!["POST", "/tasks", "{\"title\":\"Test\"}", "2026-01-01T00:00:00Z"],
        )
        .expect("insert pending op");

        let ops = read_pending_ops(&conn, 10).expect("read pending ops");
        assert_eq!(ops.len(), 1);
        assert_eq!(ops[0].path, "/tasks");

        mark_synced(&conn, ops[0].id).expect("mark synced");

        let ops = read_pending_ops(&conn, 10).expect("read pending ops after sync");
        assert!(ops.is_empty());
    }
}
