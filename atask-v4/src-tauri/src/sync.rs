use crate::auth::{AuthTokens, KEYRING_SERVICE};
use rusqlite::Connection;
use serde::Deserialize;
use std::sync::{Arc, Mutex};
use tauri::{Emitter, Manager};

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
    /// Legacy static API key (agents / headless). May be empty when the user
    /// authenticates via a Bearer token instead.
    pub api_key: String,
    /// Authenticated user id, used to namespace the sync cursor. Empty string
    /// for anonymous / api-key-only mode.
    pub user_id: String,
    /// Authenticated user email, used as the keychain account for token refresh.
    /// `None` when there is no signed-in user (api-key-only / anonymous).
    pub user_email: Option<String>,
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
    // api_key is now OPTIONAL: a signed-in user syncs via their Bearer token, so
    // an empty api_key no longer disables sync. `auth_header` decides which
    // credential to send (Bearer preferred, api_key fallback, else None).
    let api_key = get("api_key").unwrap_or_default();
    let user_id = get("user_id").unwrap_or_default();
    let user_email = get("user_email").filter(|s| !s.is_empty());
    Some(SyncConfig {
        server_url,
        api_key,
        user_id,
        user_email,
    })
}

// --- Cursor: namespaced by (server_url, user_id) so account switching never
// corrupts cursors (HARD RULE 4). Anonymous / api-key-only mode uses an empty
// user_id, which remains a stable key for that state.

fn cursor_key(server_url: &str, user_id: &str) -> String {
    format!("sync_cursor:{}:{}", server_url, user_id)
}

fn read_cursor(conn: &Connection, server_url: &str, user_id: &str) -> i64 {
    let key = cursor_key(server_url, user_id);
    conn.query_row(
        "SELECT value FROM settings WHERE key = ?1",
        [&key],
        |row| row.get::<_, String>(0),
    )
    .ok()
    .and_then(|v| v.parse().ok())
    .unwrap_or(0)
}

fn write_cursor(conn: &Connection, server_url: &str, user_id: &str, cursor: i64) -> Result<(), String> {
    let key = cursor_key(server_url, user_id);
    conn.execute(
        "INSERT INTO settings (key, value) VALUES (?1, ?2) ON CONFLICT(key) DO UPDATE SET value = ?2",
        rusqlite::params![key, cursor.to_string()],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

// --- Auth: ONE header helper shared by all three sync paths (HARD RULE 2).
// Bearer (in-memory access token) is preferred; the legacy api_key is the
// fallback; otherwise no credential.

fn auth_header(tokens: &AuthTokens, api_key: &str) -> Option<String> {
    if let Some(ref t) = *tokens.access_token.lock().unwrap() {
        return Some(format!("Bearer {}", t));
    }
    if !api_key.is_empty() {
        return Some(format!("ApiKey {}", api_key));
    }
    None
}

/// Single-flight token refresh (blocking). The `refresh_in_progress` mutex
/// serializes concurrent 401 handlers. `failed_token` is the access token the
/// caller used in the request that got 401; if the keychain no longer holds it,
/// another caller already rotated while we waited for the lock, so we adopt the
/// rotated token and skip the network call.
///
/// (Deviation from brief: the brief compared the in-memory cache to the
/// keychain, but the rotator writes the cache too, so a second waiter would see
/// cache == keychain and rotate AGAIN — invalidating the fresh token. Comparing
/// against the caller's *failed* token is the correct rotation test.)
fn refresh_access_token(
    tokens: &AuthTokens,
    server_url: &str,
    user_email: &str,
    failed_token: &str,
) -> Result<(), String> {
    let _guard = tokens
        .refresh_in_progress
        .lock()
        .map_err(|e| e.to_string())?;

    let entry = keyring::Entry::new(KEYRING_SERVICE, user_email).map_err(|e| e.to_string())?;
    let keychain_token = entry.get_password().map_err(|e| e.to_string())?;

    // Someone else already rotated while we waited for the lock.
    if keychain_token != failed_token {
        *tokens.access_token.lock().unwrap() = Some(keychain_token);
        return Ok(());
    }

    // We are the rotator. The server rotates: the old token is invalidated and a
    // new one returned.
    let resp = reqwest::blocking::Client::new()
        .post(format!("{}/auth/refresh", server_url))
        .header("Authorization", format!("Bearer {}", keychain_token))
        .send()
        .map_err(|e| e.to_string())?;
    if !resp.status().is_success() {
        return Err(format!("refresh failed: {}", resp.status()));
    }
    let body: serde_json::Value = resp.json().map_err(|e| e.to_string())?;
    let new_token = body["token"].as_str().ok_or("missing token")?.to_string();
    // Persist to keychain BEFORE updating the in-memory cache (crash-safety).
    entry.set_password(&new_token).map_err(|e| e.to_string())?;
    *tokens.access_token.lock().unwrap() = Some(new_token);
    Ok(())
}

/// 401 policy shared by all three paths. Returns Ok(true) if the token was
/// refreshed and the caller should retry; Ok(false) if there is nothing to
/// refresh (anonymous / api-key-only); Err if the refresh failed and sync
/// should pause.
fn handle_401(
    tokens: &AuthTokens,
    server_url: &str,
    user_email: Option<&str>,
    failed_token: Option<&str>,
) -> Result<bool, String> {
    // No signed-in user, or the failed request used an api_key rather than a
    // bearer token — refreshing cannot help.
    let (Some(email), Some(failed)) = (user_email, failed_token) else {
        return Ok(false);
    };
    refresh_access_token(tokens, server_url, email, failed).map(|_| true)
}

/// Outcome of an authenticated request that already applied the 401 policy.
enum AuthOutcome {
    /// A non-401 response (may still be a 4xx/5xx handled by the caller).
    Response(reqwest::blocking::Response),
    /// Nothing to refresh (anonymous / api-key-only) — the caller should stop
    /// this path without advancing the cursor or marking ops synced.
    Pause,
}

/// Send an authenticated request, transparently handling a single 401 with a
/// single-flight refresh + one retry. `build` receives the current auth header
/// value (recomputed each attempt so a refreshed token is used on retry).
fn send_authed(
    tokens: &AuthTokens,
    config: &SyncConfig,
    build: impl Fn(Option<&str>) -> reqwest::blocking::RequestBuilder,
) -> Result<AuthOutcome, String> {
    // At most 2 attempts: original + one retry after a refresh.
    for _ in 0..2 {
        let used_token = tokens.access_token.lock().unwrap().clone();
        let header = auth_header(tokens, &config.api_key);
        let resp = build(header.as_deref())
            .send()
            .map_err(|e| e.to_string())?;
        if resp.status() == reqwest::StatusCode::UNAUTHORIZED {
            match handle_401(
                tokens,
                &config.server_url,
                config.user_email.as_deref(),
                used_token.as_deref(),
            )? {
                true => continue,                    // refreshed — retry
                false => return Ok(AuthOutcome::Pause),
            }
        }
        return Ok(AuthOutcome::Response(resp));
    }
    Err("authentication expired: still unauthorized after refresh".to_string())
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

fn delete_entity(conn: &Connection, entity_type: &str, entity_id: &str) -> Result<(), String> {
    let table = match entity_type {
        "task" => "tasks",
        "project" => "projects",
        "area" => "areas",
        "section" => "sections",
        "tag" => "tags",
        "checklist_item" => "checklistItems",
        "activity" => "activities",
        "location" => "locations",
        _ => return Ok(()),
    };

    // Some parent-delete deltas do not carry the child updates that the
    // server cascaded (e.g. locations: the server clears task.locationId but
    // only emits a location delta). Replicate the cascade locally BEFORE
    // deleting the parent so we don't leave dangling references / violate
    // local FK constraints.
    if entity_type == "location" {
        conn.execute(
            "UPDATE tasks SET locationId = NULL WHERE locationId = ?1",
            [entity_id],
        )
        .map_err(|e| e.to_string())?;
    }

    conn.execute(&format!("DELETE FROM {} WHERE id = ?1", table), [entity_id])
        .map(|_| ())
        .map_err(|e| e.to_string())
}

fn flush_pending_ops_blocking(
    client: &reqwest::blocking::Client,
    conn: &Mutex<Connection>,
    config: &SyncConfig,
    tokens: &AuthTokens,
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
        // Auth header is applied inside the closure (recomputed per attempt) so
        // send_authed can retry with a refreshed token. HARD RULE 2: the header
        // comes from the single auth_header() helper, never constructed here.
        let outcome = send_authed(tokens, config, |auth| {
            let mut req = match op.method.as_str() {
                "POST" => client.post(&url),
                "PUT" => client.put(&url),
                "DELETE" => client.delete(&url),
                "PATCH" => client.patch(&url),
                _ => client.post(&url),
            };
            if let Some(a) = auth {
                req = req.header("Authorization", a);
            }
            if let Some(body) = &op.body {
                req = req
                    .header("Content-Type", "application/json")
                    .body(body.clone());
            }
            req
        });

        // 401 policy: never mark an op synced; refresh + retry, else pause.
        // Pause / refresh-failure stops the flush WITHOUT marking anything synced.
        let resp = match outcome {
            Ok(AuthOutcome::Response(r)) => r,
            Ok(AuthOutcome::Pause) => break,
            Err(e) => {
                if let Ok(c) = conn.lock() {
                    let _ = set_last_sync_error(&c, Some(&e));
                }
                break;
            }
        };

        {
            let status = resp.status().as_u16();
            // Capture response body for 4xx logging before we take the lock.
            let body_preview = if (400..500).contains(&status) {
                resp.text().unwrap_or_default()
            } else {
                String::new()
            };
            let c = conn.lock().map_err(|e| e.to_string())?;
            if (200..300).contains(&status) || status == 404 || status == 409 {
                // 2xx: success. 404: entity already gone (idempotent
                // delete/patch). 409: conflict, server wins — delta pull
                // will reconcile. All "desired state already reached".
                mark_synced(&c, op.id)?;
                flushed += 1;
                consecutive_failures = 0;
            } else if (400..500).contains(&status) {
                // Other 4xx (400/403/422/...) mean the op's body violates the
                // server contract. (401 is handled earlier by send_authed and
                // never reaches here.) Marking it synced avoids an infinite
                // retry loop, but we MUST surface the drift.
                eprintln!(
                    "sync: dropping pending op #{} ({} {}) — server returned {}: {}",
                    op.id, op.method, op.path, status, body_preview
                );
                let _ = set_last_sync_error(
                    &c,
                    Some(&format!(
                        "server rejected {} {} with {}: {}",
                        op.method, op.path, status, body_preview
                    )),
                );
                mark_synced(&c, op.id)?;
                flushed += 1;
                consecutive_failures = 0;
            } else {
                // 5xx / unexpected — retry on the next tick.
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

/// Map an entity type to its REST collection segment, or None if it has no
/// individual fetch endpoint (e.g. checklist_item creates/updates arrive
/// embedded in their parent task response).
fn entity_plural(entity_type: &str) -> Option<&'static str> {
    match entity_type {
        "task" => Some("tasks"),
        "project" => Some("projects"),
        "area" => Some("areas"),
        "section" => Some("sections"),
        "tag" => Some("tags"),
        "activity" => Some("activities"),
        "location" => Some("locations"),
        _ => None,
    }
}

fn upsert_entity(conn: &Connection, entity_type: &str, json: &serde_json::Value) -> Result<(), String> {
    match entity_type {
        "task" => upsert_task(conn, json),
        "project" => upsert_project(conn, json),
        "area" => upsert_area(conn, json),
        "section" => upsert_section(conn, json),
        "tag" => upsert_tag(conn, json),
        "activity" => upsert_activity(conn, json),
        "location" => upsert_location(conn, json),
        _ => Ok(()),
    }
}

fn pull_deltas_blocking(
    client: &reqwest::blocking::Client,
    conn: &Mutex<Connection>,
    config: &SyncConfig,
    tokens: &AuthTokens,
) -> Result<usize, String> {
    let mut cursor = {
        let c = conn.lock().map_err(|e| e.to_string())?;
        read_cursor(&c, &config.server_url, &config.user_id)
    };

    let deltas_url = format!("{}/sync/deltas?since={}", config.server_url, cursor);
    // HARD RULE 2 + 3: the deltas GET goes through send_authed. A 401 refreshes
    // and retries; on Pause/refresh-failure we return WITHOUT advancing the
    // cursor.
    let resp = match send_authed(tokens, config, |auth| {
        let mut req = client.get(&deltas_url);
        if let Some(a) = auth {
            req = req.header("Authorization", a);
        }
        req
    }) {
        Ok(AuthOutcome::Response(r)) => r,
        Ok(AuthOutcome::Pause) => return Ok(0),
        Err(e) => {
            if let Ok(c) = conn.lock() {
                let _ = set_last_sync_error(&c, Some(&e));
            }
            return Ok(0);
        }
    };

    if !resp.status().is_success() {
        return Err(format!("delta pull failed: {}", resp.status()));
    }

    let mut deltas: Vec<DeltaEvent> = resp.json().map_err(|e| e.to_string())?;
    if deltas.is_empty() {
        return Ok(0);
    }

    // HARD RULE 3: advance the cursor INSIDE each delta's success arm, in strict
    // id order, so a 401 (or refresh failure) mid-batch leaves the cursor at the
    // last fully-applied delta — never beyond it. (We process per-delta rather
    // than via the collapsed plan so the cursor is always precise; upserts are
    // idempotent, so any redundant fetch is harmless.)
    deltas.sort_by_key(|d| d.id);
    let mut applied = 0usize;

    for delta in &deltas {
        let entity_type = delta.entity_type.string.as_str();
        let entity_id = delta.entity_id.string.as_str();

        if delta.action.int64 == 2 {
            // Delete: applied locally, always succeeds.
            let c = conn.lock().map_err(|e| e.to_string())?;
            delete_entity(&c, entity_type, entity_id)?;
            applied += 1;
        } else {
            match entity_plural(entity_type) {
                None => { /* no individual fetch endpoint — skip, advance cursor */ }
                Some(plural) => {
                    let url = format!("{}/{}/{}", config.server_url, plural, entity_id);
                    let resp = match send_authed(tokens, config, |auth| {
                        let mut req = client.get(&url);
                        if let Some(a) = auth {
                            req = req.header("Authorization", a);
                        }
                        req
                    }) {
                        Ok(AuthOutcome::Response(r)) => r,
                        // 401 could not be recovered / anonymous: STOP without
                        // advancing past this delta (cursor stays at last success).
                        Ok(AuthOutcome::Pause) => return Ok(applied),
                        Err(e) => {
                            if let Ok(c) = conn.lock() {
                                let _ = set_last_sync_error(&c, Some(&e));
                            }
                            return Ok(applied);
                        }
                    };

                    if !resp.status().is_success() {
                        // Non-401 fetch failure (404/5xx). Log and skip this
                        // delta (advance cursor) to avoid wedging sync forever —
                        // matches the previous behavior. 401 never reaches here.
                        eprintln!(
                            "entity fetch failed for {entity_type}/{entity_id}: {} — skipping",
                            resp.status()
                        );
                    } else {
                        let json = resp
                            .json::<serde_json::Value>()
                            .map_err(|e| e.to_string())?;
                        let c = conn.lock().map_err(|e| e.to_string())?;
                        upsert_entity(&c, entity_type, &json)?;
                        applied += 1;
                    }
                }
            }
        }

        // Success arm: this delta is fully applied (or safely skipped). Advance
        // the cursor to its id so a later failure never re-loses earlier work.
        cursor = delta.id;
        let c = conn.lock().map_err(|e| e.to_string())?;
        write_cursor(&c, &config.server_url, &config.user_id, cursor)?;
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

    // The in-memory auth tokens live in Tauri managed state (Bearer token).
    let tokens = app_handle.state::<AuthTokens>();

    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(15))
        .build()
        .map_err(|e| e.to_string())?;

    // 1. Flush outbound
    flush_pending_ops_blocking(&client, conn, &config, &tokens)?;

    // 2. Pull inbound deltas
    let pulled = pull_deltas_blocking(&client, conn, &config, &tokens)?;

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
        "INSERT INTO tasks (id, title, notes, status, schedule, startDate, deadline, completedAt, \"index\", todayIndex, timeSlot, projectId, sectionId, areaId, locationId, createdAt, updatedAt, syncStatus, repeatRule) \
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,0,?18) \
         ON CONFLICT(id) DO UPDATE SET \
         title=?2, notes=?3, status=?4, schedule=?5, startDate=?6, deadline=?7, completedAt=?8, \"index\"=?9, todayIndex=?10, timeSlot=?11, projectId=?12, sectionId=?13, areaId=?14, locationId=?15, updatedAt=?17, syncStatus=0, repeatRule=?18",
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
            opt_s("locationId", "LocationID"),
            s("createdAt", "CreatedAt"),
            s("updatedAt", "UpdatedAt"),
            repeat_rule,
        ],
    )
    .map(|_| ())
    .map_err(|e| e.to_string())?;

    // Sync task-tag associations from server response.
    // The Go API returns tags as an array of tag ID strings.
    if let Some(tags) = j.get("tags").and_then(|v| v.as_array()) {
        conn.execute(
            "DELETE FROM taskTags WHERE taskId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        for tag_val in tags {
            if let Some(tag_id) = tag_val.as_str() {
                conn.execute(
                    "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?1, ?2)",
                    rusqlite::params![id, tag_id],
                )
                .map_err(|e| e.to_string())?;
            }
        }
    }

    // Sync task-link associations from server response.
    // The Go API returns linkedTaskIds as an array of task ID strings.
    if let Some(links) = j.get("linkedTaskIds").and_then(|v| v.as_array()) {
        conn.execute(
            "DELETE FROM taskLinks WHERE taskId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        for link_val in links {
            if let Some(linked_id) = link_val.as_str() {
                conn.execute(
                    "INSERT OR IGNORE INTO taskLinks (taskId, linkedTaskId) VALUES (?1, ?2)",
                    rusqlite::params![id, linked_id],
                )
                .map_err(|e| e.to_string())?;
            }
        }
    }

    Ok(())
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
    .map_err(|e| e.to_string())?;

    // Sync project-tag associations from server response.
    // The Go API returns tags as an array of tag ID strings.
    if let Some(tags) = j.get("tags").and_then(|v| v.as_array()) {
        conn.execute(
            "DELETE FROM projectTags WHERE projectId = ?1",
            rusqlite::params![id],
        )
        .map_err(|e| e.to_string())?;

        for tag_val in tags {
            if let Some(tag_id) = tag_val.as_str() {
                conn.execute(
                    "INSERT OR IGNORE INTO projectTags (projectId, tagId) VALUES (?1, ?2)",
                    rusqlite::params![id, tag_id],
                )
                .map_err(|e| e.to_string())?;
            }
        }
    }

    Ok(())
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

pub fn upsert_location(conn: &Connection, j: &serde_json::Value) -> Result<(), String> {
    let s = |key: &str, alt: &str| -> &str {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
    };
    let opt_f = |key: &str, alt: &str| -> Option<f64> {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| if v.is_null() { None } else { v.as_f64() })
    };
    let opt_i = |key: &str, alt: &str| -> Option<i64> {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| if v.is_null() { None } else { v.as_i64() })
    };
    let opt_s = |key: &str, alt: &str| -> Option<&str> {
        j.get(key)
            .or_else(|| j.get(alt))
            .and_then(|v| if v.is_null() { None } else { v.as_str() })
    };

    let id = s("id", "ID");
    if id.is_empty() {
        return Ok(());
    }

    conn.execute(
        "INSERT INTO locations (id, name, latitude, longitude, radius, address, createdAt, updatedAt) \
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8) \
         ON CONFLICT(id) DO UPDATE SET \
         name=?2, latitude=?3, longitude=?4, radius=?5, address=?6, updatedAt=?8",
        rusqlite::params![
            id,
            s("name", "Name"),
            opt_f("latitude", "Latitude"),
            opt_f("longitude", "Longitude"),
            opt_i("radius", "Radius"),
            opt_s("address", "Address"),
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
        conn.execute_batch(include_str!("migrations/004_locations.sql"))
            .expect("locations migration");
        // Add locationId to tasks (in-memory DB needs this explicitly)
        let _ = conn.execute_batch("ALTER TABLE tasks ADD COLUMN locationId TEXT REFERENCES locations(id)");
        conn
    }

    // HARD RULE 2: one auth_header() helper, Bearer preferred over api_key.
    #[test]
    fn auth_header_prefers_bearer_then_api_key_then_none() {
        let tokens = AuthTokens::default();

        // No bearer, no api_key -> None.
        assert_eq!(auth_header(&tokens, ""), None);

        // No bearer, api_key present -> ApiKey.
        assert_eq!(
            auth_header(&tokens, "secret-key"),
            Some("ApiKey secret-key".to_string())
        );

        // Bearer present -> Bearer wins even when an api_key is also configured.
        *tokens.access_token.lock().unwrap() = Some("access-123".to_string());
        assert_eq!(
            auth_header(&tokens, "secret-key"),
            Some("Bearer access-123".to_string())
        );
    }

    // HARD RULE 4: cursor keyed by (server_url, user_id).
    #[test]
    fn cursor_key_is_namespaced_by_server_and_user() {
        assert_eq!(
            cursor_key("https://api.example.com", "user-42"),
            "sync_cursor:https://api.example.com:user-42"
        );
        // Anonymous / api-key-only mode uses an empty user id.
        assert_eq!(
            cursor_key("https://api.example.com", ""),
            "sync_cursor:https://api.example.com:"
        );
    }

    #[test]
    fn cursor_read_write_roundtrips_and_isolates_users() {
        let conn = setup_test_conn();
        let server = "https://api.example.com";

        // Unset cursor defaults to 0.
        assert_eq!(read_cursor(&conn, server, "user-a"), 0);

        write_cursor(&conn, server, "user-a", 17).expect("write a");
        write_cursor(&conn, server, "user-b", 99).expect("write b");

        // Cursors for different users are isolated — no cross-account corruption.
        assert_eq!(read_cursor(&conn, server, "user-a"), 17);
        assert_eq!(read_cursor(&conn, server, "user-b"), 99);

        // Overwrite advances in place.
        write_cursor(&conn, server, "user-a", 42).expect("advance a");
        assert_eq!(read_cursor(&conn, server, "user-a"), 42);
    }

    // HARD RULE 3 (partial, testable without network/keychain): handle_401 does
    // NOT attempt a refresh — and therefore signals "pause, do not advance" — when
    // there is no signed-in user or the failed request used no bearer token.
    #[test]
    fn handle_401_is_noop_without_user_or_bearer() {
        let tokens = AuthTokens::default();

        // No user email (anonymous / api-key-only) -> Ok(false), no refresh.
        assert_eq!(
            handle_401(&tokens, "https://api.example.com", None, Some("tok")),
            Ok(false)
        );

        // Signed-in user but the failed request carried no bearer token (api_key
        // path) -> Ok(false); refreshing an api_key is meaningless.
        assert_eq!(
            handle_401(&tokens, "https://api.example.com", Some("me@example.com"), None),
            Ok(false)
        );
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
