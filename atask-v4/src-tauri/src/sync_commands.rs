use crate::db::Database;

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncStatus {
    pub is_syncing: bool,
    pub last_sync_at: Option<String>,
    pub last_error: Option<String>,
    pub pending_ops_count: i64,
}

#[tauri::command]
pub fn get_sync_status(db: tauri::State<'_, Database>) -> Result<SyncStatus, String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;

    let pending_ops_count: i64 = conn
        .query_row(
            "SELECT COUNT(*) FROM pendingOps WHERE synced = 0",
            [],
            |row| row.get(0),
        )
        .unwrap_or(0);

    let last_sync_at: Option<String> = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'last_sync_at'",
            [],
            |row| row.get(0),
        )
        .ok();

    let last_error: Option<String> = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'last_sync_error'",
            [],
            |row| row.get(0),
        )
        .ok()
        .filter(|s: &String| !s.is_empty());

    Ok(SyncStatus {
        is_syncing: false,
        last_sync_at,
        last_error,
        pending_ops_count,
    })
}

#[tauri::command]
pub fn trigger_sync(
    db: tauri::State<'_, Database>,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // Perform a full sync cycle: flush pending ops + pull deltas
    crate::sync::sync_now_blocking(&db.conn, &app_handle)
}

#[tauri::command]
pub fn test_connection(
    db: tauri::State<'_, Database>,
    tokens: tauri::State<'_, crate::auth::AuthTokens>,
) -> Result<bool, String> {
    let (server_url, api_key) = {
        let conn = db.conn.lock().map_err(|e| e.to_string())?;

        let server_url: String = conn
            .query_row(
                "SELECT value FROM settings WHERE key = 'server_url'",
                [],
                |row| row.get(0),
            )
            .unwrap_or_default();

        let api_key: String = conn
            .query_row(
                "SELECT value FROM settings WHERE key = 'api_key'",
                [],
                |row| row.get(0),
            )
            .unwrap_or_default();

        (server_url, api_key)
    };

    if server_url.is_empty() {
        return Err("server_url must be set in settings".to_string());
    }

    // Bearer (signed-in access token) is preferred; the legacy api_key is the
    // fallback. Reuse the same resolution the delta-sync engine uses so a
    // logged-in user with no api_key can still test/sync.
    let token = tokens.access_token.lock().map_err(|e| e.to_string())?.clone();
    let auth = crate::sync::auth_header(token.as_deref(), &api_key)
        .ok_or_else(|| "sign in or set an api key first".to_string())?;

    // Hit an authenticated endpoint so this actually validates the credential
    // (the public /health route would succeed regardless of auth).
    let url = format!("{}/tasks", server_url);
    let client = reqwest::blocking::Client::new();
    let resp = client
        .get(&url)
        .header("Authorization", auth)
        .send()
        .map_err(|e| e.to_string())?;

    Ok(resp.status().is_success())
}

#[derive(serde::Deserialize)]
pub struct InitialSyncParams {
    pub mode: String,
}

// `auth` is the fully-formed Authorization header value ("Bearer …" or
// "ApiKey …"), resolved once by the caller via `sync::auth_header` so the
// Bearer-preferred credential selection stays consistent with the delta path.
fn pull_all_from_server(
    conn: &rusqlite::Connection,
    server_url: &str,
    auth: &str,
) -> Result<(), String> {
    let client = reqwest::blocking::Client::new();

    // Fetch tasks
    let tasks: Vec<serde_json::Value> = client
        .get(format!("{}/tasks?status=all", server_url))
        .header("Authorization", auth)
        .send()
        .map_err(|e| e.to_string())?
        .json()
        .map_err(|e| e.to_string())?;

    for task in &tasks {
        crate::sync::upsert_task(conn, task)?;
    }

    // Fetch projects
    let projects: Vec<serde_json::Value> = client
        .get(format!("{}/projects?status=all", server_url))
        .header("Authorization", auth)
        .send()
        .map_err(|e| e.to_string())?
        .json()
        .map_err(|e| e.to_string())?;

    for project in &projects {
        crate::sync::upsert_project(conn, project)?;
    }

    // Fetch areas
    let areas: Vec<serde_json::Value> = client
        .get(format!("{}/areas?include_archived=true", server_url))
        .header("Authorization", auth)
        .send()
        .map_err(|e| e.to_string())?
        .json()
        .map_err(|e| e.to_string())?;

    for area in &areas {
        crate::sync::upsert_area(conn, area)?;
    }

    // Fetch sections. There is no bare `/sections` collection route on the
    // server — sections are nested under their project — so pull them
    // per-project from the projects we just fetched.
    for project in &projects {
        let Some(project_id) = project["id"].as_str() else {
            continue;
        };
        let sections: Vec<serde_json::Value> = client
            .get(format!("{}/projects/{}/sections", server_url, project_id))
            .header("Authorization", auth)
            .send()
            .map_err(|e| e.to_string())?
            .json()
            .map_err(|e| e.to_string())?;

        for section in &sections {
            crate::sync::upsert_section(conn, section)?;
        }
    }

    // Fetch tags
    let tags: Vec<serde_json::Value> = client
        .get(format!("{}/tags", server_url))
        .header("Authorization", auth)
        .send()
        .map_err(|e| e.to_string())?
        .json()
        .map_err(|e| e.to_string())?;

    for tag in &tags {
        crate::sync::upsert_tag(conn, tag)?;
    }

    Ok(())
}

#[tauri::command]
pub fn initial_sync(
    params: InitialSyncParams,
    db: tauri::State<'_, Database>,
    tokens: tauri::State<'_, crate::auth::AuthTokens>,
) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;

    let server_url: String = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'server_url'",
            [],
            |row| row.get(0),
        )
        .unwrap_or_default();

    let api_key: String = conn
        .query_row(
            "SELECT value FROM settings WHERE key = 'api_key'",
            [],
            |row| row.get(0),
        )
        .unwrap_or_default();

    if server_url.is_empty() {
        return Err("server_url must be set in settings".to_string());
    }

    // Bearer (signed-in access token) preferred, api_key fallback — same
    // resolution as the delta-sync engine, so a logged-in user syncs without
    // needing an api_key.
    let token = tokens.access_token.lock().map_err(|e| e.to_string())?.clone();
    let auth = crate::sync::auth_header(token.as_deref(), &api_key)
        .ok_or_else(|| "sign in or set an api key first".to_string())?;

    match params.mode.as_str() {
        "fresh" => {
            // Delete all local data, then pull from server
            conn.execute_batch(
                "DELETE FROM taskTags;
                 DELETE FROM checklistItems;
                 DELETE FROM tasks;
                 DELETE FROM sections;
                 DELETE FROM projects;
                 DELETE FROM areas;
                 DELETE FROM tags;
                 DELETE FROM pendingOps;",
            )
            .map_err(|e| e.to_string())?;

            pull_all_from_server(&conn, &server_url, &auth)?;
        }
        "merge" => {
            // Pull server entities, upsert by ID (server wins on conflict)
            pull_all_from_server(&conn, &server_url, &auth)?;
        }
        "push" => {
            // Push logic is complex — handled by the pending ops sync worker
            // Nothing to do here for now
        }
        _ => {
            return Err(format!("unknown sync mode: {}", params.mode));
        }
    }

    // Record last_sync_at
    let now = chrono::Utc::now().to_rfc3339();
    conn.execute(
        "INSERT INTO settings (key, value) VALUES ('last_sync_at', ?1)
         ON CONFLICT(key) DO UPDATE SET value = ?1",
        rusqlite::params![now],
    )
    .map_err(|e| e.to_string())?;

    Ok(())
}
