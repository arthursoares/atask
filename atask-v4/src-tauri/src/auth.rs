//! Authentication state and Tauri commands (Task 19).
//!
//! HARD RULE 1: No auth token of any kind is written to Tauri SQLite. The
//! canonical token lives in the OS keychain; a working copy lives in the
//! in-memory `AuthTokens.access_token` mutex held by Tauri `State`. The
//! `settings` table stores only profile cache (`user_id`, `user_email`,
//! `user_name`, `server_url`) — never token material.
//!
//! Note on async vs blocking: the sync engine (`sync.rs`) is entirely blocking
//! (`reqwest::blocking`) and runs on plain `std::thread`s with no tokio runtime
//! in scope. The brief sketched the single-flight refresh with
//! `tokio::sync::Mutex` + `async`, but `.await` on those threads would have no
//! runtime. We therefore implement the identical single-flight semantics with a
//! blocking `std::sync::Mutex`, which is correct and architecturally consistent.
//! See `sync::refresh_access_token` for the coordinator.

use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use tauri::State;

use crate::db::Database;

/// Tokens held only in memory. Lost on app restart by design — `refresh_on_launch`
/// re-derives the access token from the keychain-stored token.
///
/// `refresh_in_progress` is the single-flight coordinator: if multiple sync
/// paths see a 401 simultaneously (e.g. the background worker and a
/// `trigger_sync` command overlapping), only the first holder rotates the token;
/// the rest park on this lock and detect the rotation (their failed token no
/// longer matches the keychain) and skip the redundant network call.
#[derive(Default)]
pub struct AuthTokens {
    pub access_token: Mutex<Option<String>>,
    pub refresh_in_progress: Mutex<()>,
}

/// Profile cache surfaced to the frontend. Never carries token material — the
/// `authenticated` bool is the only signal of session presence.
///
/// `rename_all = "camelCase"` matches every other Serialize/Deserialize model
/// in this crate (see models.rs) so the Tauri IPC JSON boundary is consistent
/// (`userId`, `userEmail`, `userName`, `serverUrl`) — Task 20 (frontend) reads
/// this shape directly.
#[derive(Serialize, Deserialize, Clone, Default)]
#[serde(rename_all = "camelCase")]
pub struct AuthState {
    pub user_id: Option<String>,
    pub user_email: Option<String>,
    pub user_name: Option<String>,
    pub server_url: Option<String>,
    pub authenticated: bool,
}

/// Keychain service name. The account is the user's email.
pub const KEYRING_SERVICE: &str = "atask-refresh-token";

fn get_setting(conn: &rusqlite::Connection, key: &str) -> Option<String> {
    conn.query_row("SELECT value FROM settings WHERE key = ?1", [key], |r| {
        r.get(0)
    })
    .ok()
}

/// Log in against the server, storing the returned token in the OS keychain
/// (canonical) and the in-memory cache (copy). Only profile fields are written
/// to SQLite — NEVER the token.
#[tauri::command]
pub fn login(
    db: State<Database>,
    tokens: State<AuthTokens>,
    server_url: String,
    email: String,
    password: String,
) -> Result<AuthState, String> {
    let client = reqwest::blocking::Client::new();
    let resp = client
        .post(format!("{}/auth/login", server_url))
        .json(&serde_json::json!({"email": email, "password": password}))
        .send()
        .map_err(|e| e.to_string())?;

    if !resp.status().is_success() {
        return Err(format!("Login failed: {}", resp.status()));
    }

    let body: serde_json::Value = resp.json().map_err(|e| e.to_string())?;
    // PocketBase / the Go server issues a single auth token (not a separate
    // access/refresh pair). The keychain holds the canonical copy; the in-memory
    // cache holds a working copy. The 401 handler rotates both.
    let token = body["token"].as_str().ok_or("missing token")?.to_string();
    let user_id = body["user"]["id"].as_str().unwrap_or("").to_string();
    let user_email = body["user"]["email"]
        .as_str()
        .filter(|s| !s.is_empty())
        .unwrap_or(&email)
        .to_string();
    let user_name = body["user"]["name"].as_str().unwrap_or("").to_string();

    // Token -> OS keychain (canonical) then in-memory cache (copy).
    let entry = keyring::Entry::new(KEYRING_SERVICE, &user_email).map_err(|e| e.to_string())?;
    entry.set_password(&token).map_err(|e| e.to_string())?;
    *tokens.access_token.lock().unwrap() = Some(token);

    // Profile cache -> SQLite (NO TOKEN MATERIAL).
    {
        let conn = db.conn.lock().map_err(|e| e.to_string())?;
        for (key, value) in [
            ("user_id", &user_id),
            ("user_email", &user_email),
            ("user_name", &user_name),
            ("server_url", &server_url),
        ] {
            conn.execute(
                "INSERT OR REPLACE INTO settings (key, value) VALUES (?1, ?2)",
                rusqlite::params![key, value],
            )
            .ok();
        }
    }

    Ok(AuthState {
        user_id: Some(user_id),
        user_email: Some(user_email),
        user_name: Some(user_name),
        server_url: Some(server_url),
        authenticated: true,
    })
}

/// On launch, restore the in-memory access token from the keychain with a
/// SINGLE read — no network refresh and, crucially, no keychain WRITE. The
/// token is used as-is until a sync request gets a 401, at which point
/// `sync::handle_401` does a single-flight refresh (and only then rotates +
/// rewrites the keychain). This keeps app launch to one keychain access
/// instead of a read+write: on an unsigned/dev build every keychain touch
/// triggers a macOS authorization prompt, and a write (ACL modification) is
/// the one that escalates to a full password prompt — so dropping the launch
/// write removes the most intrusive prompt. It also avoids rotating a
/// still-valid token on every start.
///
/// Trade-off: a token that is already expired at launch is loaded optimistically
/// (AuthState reports authenticated); the first sync then 401s and, since an
/// expired token can't be refreshed, surfaces as a sync error prompting re-login
/// — the same end state the old eager refresh reached, just discovered lazily.
#[tauri::command]
pub fn refresh_on_launch(
    db: State<Database>,
    tokens: State<AuthTokens>,
) -> Result<AuthState, String> {
    let (server_url, user_email) = {
        let conn = db.conn.lock().map_err(|e| e.to_string())?;
        let server_url = match get_setting(&conn, "server_url") {
            Some(v) => v,
            None => return Ok(AuthState::default()),
        };
        let user_email = match get_setting(&conn, "user_email") {
            Some(v) => v,
            None => return Ok(AuthState::default()),
        };
        (server_url, user_email)
    };

    // Single keychain read; load straight into the in-memory cache. No refresh
    // network call, no keychain write.
    let entry = keyring::Entry::new(KEYRING_SERVICE, &user_email).map_err(|e| e.to_string())?;
    let token = match entry.get_password() {
        Ok(t) => t,
        Err(_) => return Ok(AuthState::default()), // not signed in
    };
    *tokens.access_token.lock().unwrap() = Some(token);

    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    Ok(AuthState {
        user_id: get_setting(&conn, "user_id"),
        user_email: Some(user_email),
        user_name: get_setting(&conn, "user_name"),
        server_url: Some(server_url),
        authenticated: true,
    })
}

/// Sign out: clear the keychain token, the in-memory cache, the profile cache,
/// all namespaced sync cursors, and all local domain data.
#[tauri::command]
pub fn logout(db: State<Database>, tokens: State<AuthTokens>) -> Result<(), String> {
    let conn = db.conn.lock().map_err(|e| e.to_string())?;

    // Clear keychain token.
    if let Some(email) = get_setting(&conn, "user_email") {
        if let Ok(entry) = keyring::Entry::new(KEYRING_SERVICE, &email) {
            entry.delete_credential().ok();
        }
    }

    // Clear in-memory access token.
    *tokens.access_token.lock().unwrap() = None;

    // Clear profile cache.
    for key in ["user_id", "user_email", "user_name", "server_url"] {
        conn.execute("DELETE FROM settings WHERE key = ?1", [key]).ok();
    }

    // Clear ALL per-user/per-server cursor keys (namespaced; see sync::cursor_key).
    conn.execute("DELETE FROM settings WHERE key LIKE 'sync_cursor:%'", [])
        .ok();

    // Wipe local domain data (uses the actual local table names).
    for table in [
        "tasks",
        "projects",
        "areas",
        "sections",
        "tags",
        "locations",
        "checklistItems",
        "activities",
        "taskTags",
        "projectTags",
        "taskLinks",
    ] {
        conn.execute(&format!("DELETE FROM {}", table), []).ok();
    }
    conn.execute("DELETE FROM pendingOps", []).ok();

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn auth_state_default_is_unauthenticated() {
        let s = AuthState::default();
        assert!(!s.authenticated);
        assert!(s.user_id.is_none());
        assert!(s.server_url.is_none());
    }
}
