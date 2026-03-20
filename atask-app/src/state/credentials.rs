use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct Credentials {
    pub token: Option<String>,
    pub api_url: Option<String>,
}

fn credentials_path() -> PathBuf {
    let home = std::env::var("HOME").unwrap_or_else(|_| ".".to_string());
    PathBuf::from(home).join(".config/atask/credentials.json")
}

pub fn load() -> Credentials {
    let path = credentials_path();
    match std::fs::read_to_string(&path) {
        Ok(contents) => serde_json::from_str(&contents).unwrap_or_default(),
        Err(_) => Credentials::default(),
    }
}

pub fn save(creds: &Credentials) {
    let path = credentials_path();
    if let Some(parent) = path.parent() {
        let _ = std::fs::create_dir_all(parent);
    }
    if let Ok(json) = serde_json::to_string_pretty(creds) {
        let _ = std::fs::write(&path, json);
    }
}

pub fn clear() {
    let path = credentials_path();
    let _ = std::fs::remove_file(&path);
}
