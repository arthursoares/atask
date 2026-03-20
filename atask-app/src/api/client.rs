use reqwest::Client;
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};

use super::types::*;

#[derive(Debug)]
pub enum ApiError {
    Network(reqwest::Error),
    Api { status: u16, message: String },
}

impl std::fmt::Display for ApiError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ApiError::Network(e) => write!(f, "network error: {e}"),
            ApiError::Api { status, message } => write!(f, "API error {status}: {message}"),
        }
    }
}

impl From<reqwest::Error> for ApiError {
    fn from(e: reqwest::Error) -> Self {
        ApiError::Network(e)
    }
}

#[derive(Clone)]
pub struct ApiClient {
    base_url: String,
    token: Option<String>,
    client: Client,
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------
impl ApiClient {
    fn auth_token(&self) -> &str {
        self.token.as_deref().unwrap_or("")
    }

    async fn get_json<T: DeserializeOwned>(&self, path: &str) -> Result<T, ApiError> {
        let resp = self
            .client
            .get(format!("{}{}", self.base_url, path))
            .bearer_auth(self.auth_token())
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let msg = resp.text().await.unwrap_or_default();
            return Err(ApiError::Api {
                status,
                message: msg,
            });
        }
        Ok(resp.json().await?)
    }

    async fn post_json<T: DeserializeOwned>(
        &self,
        path: &str,
        body: &impl Serialize,
    ) -> Result<T, ApiError> {
        let resp = self
            .client
            .post(format!("{}{}", self.base_url, path))
            .bearer_auth(self.auth_token())
            .json(body)
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let msg = resp.text().await.unwrap_or_default();
            return Err(ApiError::Api {
                status,
                message: msg,
            });
        }
        Ok(resp.json().await?)
    }

    async fn post_action(&self, path: &str) -> Result<(), ApiError> {
        let resp = self
            .client
            .post(format!("{}{}", self.base_url, path))
            .bearer_auth(self.auth_token())
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let msg = resp.text().await.unwrap_or_default();
            return Err(ApiError::Api {
                status,
                message: msg,
            });
        }
        Ok(())
    }

    async fn put_json(&self, path: &str, body: &impl Serialize) -> Result<(), ApiError> {
        let resp = self
            .client
            .put(format!("{}{}", self.base_url, path))
            .bearer_auth(self.auth_token())
            .json(body)
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let msg = resp.text().await.unwrap_or_default();
            return Err(ApiError::Api {
                status,
                message: msg,
            });
        }
        Ok(())
    }

    async fn delete_action(&self, path: &str) -> Result<(), ApiError> {
        let resp = self
            .client
            .delete(format!("{}{}", self.base_url, path))
            .bearer_auth(self.auth_token())
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let msg = resp.text().await.unwrap_or_default();
            return Err(ApiError::Api {
                status,
                message: msg,
            });
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------
impl ApiClient {
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            token: None,
            client: Client::new(),
        }
    }

    pub fn set_token(&mut self, token: String) {
        self.token = Some(token);
    }

    pub fn has_token(&self) -> bool {
        self.token.is_some()
    }

    pub fn base_url(&self) -> &str {
        &self.base_url
    }

    pub fn token(&self) -> Option<&str> {
        self.token.as_deref()
    }

    // -- Auth ---------------------------------------------------------------

    pub async fn login(&self, email: &str, password: &str) -> Result<String, ApiError> {
        #[derive(Deserialize)]
        struct LoginResponse {
            token: String,
        }
        let resp: LoginResponse = self
            .post_json(
                "/auth/login",
                &serde_json::json!({"email": email, "password": password}),
            )
            .await?;
        Ok(resp.token)
    }

    pub async fn register(
        &self,
        email: &str,
        password: &str,
        name: &str,
    ) -> Result<(), ApiError> {
        let resp = self
            .client
            .post(format!("{}/auth/register", self.base_url))
            .json(&serde_json::json!({"email": email, "password": password, "name": name}))
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let msg = resp.text().await.unwrap_or_default();
            return Err(ApiError::Api {
                status,
                message: msg,
            });
        }
        Ok(())
    }

    // -- Views (bare arrays) ------------------------------------------------

    pub async fn list_inbox(&self) -> Result<Vec<Task>, ApiError> {
        self.get_json("/views/inbox").await
    }

    pub async fn list_today(&self) -> Result<Vec<Task>, ApiError> {
        self.get_json("/views/today").await
    }

    pub async fn list_upcoming(&self) -> Result<Vec<Task>, ApiError> {
        self.get_json("/views/upcoming").await
    }

    pub async fn list_someday(&self) -> Result<Vec<Task>, ApiError> {
        self.get_json("/views/someday").await
    }

    pub async fn list_logbook(&self) -> Result<Vec<Task>, ApiError> {
        self.get_json("/views/logbook").await
    }

    // -- Tasks --------------------------------------------------------------

    pub async fn create_task(&self, title: &str) -> Result<Task, ApiError> {
        let envelope: EventEnvelope<Task> = self
            .post_json("/tasks", &serde_json::json!({"title": title}))
            .await?;
        Ok(envelope.data)
    }

    pub async fn complete_task(&self, id: &str) -> Result<(), ApiError> {
        self.post_action(&format!("/tasks/{id}/complete")).await
    }

    pub async fn cancel_task(&self, id: &str) -> Result<(), ApiError> {
        self.post_action(&format!("/tasks/{id}/cancel")).await
    }

    pub async fn delete_task(&self, id: &str) -> Result<(), ApiError> {
        self.delete_action(&format!("/tasks/{id}")).await
    }

    pub async fn update_task_title(&self, id: &str, title: &str) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/title"),
            &serde_json::json!({"title": title}),
        )
        .await
    }

    pub async fn update_task_notes(&self, id: &str, notes: &str) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/notes"),
            &serde_json::json!({"notes": notes}),
        )
        .await
    }

    pub async fn update_task_schedule(&self, id: &str, schedule: &str) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/schedule"),
            &serde_json::json!({"schedule": schedule}),
        )
        .await
    }

    pub async fn set_task_start_date(
        &self,
        id: &str,
        date: Option<&str>,
    ) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/start-date"),
            &serde_json::json!({"date": date}),
        )
        .await
    }

    pub async fn set_task_deadline(&self, id: &str, date: Option<&str>) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/deadline"),
            &serde_json::json!({"date": date}),
        )
        .await
    }

    pub async fn reorder_task(&self, id: &str, new_index: i32) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/reorder"),
            &serde_json::json!({"index": new_index}),
        )
        .await
    }

    pub async fn move_task_to_project(
        &self,
        id: &str,
        project_id: Option<&str>,
    ) -> Result<(), ApiError> {
        self.put_json(
            &format!("/tasks/{id}/project"),
            &serde_json::json!({"id": project_id}),
        )
        .await
    }

    // -- Projects -----------------------------------------------------------

    pub async fn list_projects(&self) -> Result<Vec<Project>, ApiError> {
        self.get_json("/projects").await
    }

    pub async fn get_project(&self, id: &str) -> Result<Project, ApiError> {
        self.get_json(&format!("/projects/{id}")).await
    }

    pub async fn list_tasks_by_project(&self, project_id: &str) -> Result<Vec<Task>, ApiError> {
        self.get_json(&format!("/tasks?project_id={project_id}"))
            .await
    }

    pub async fn list_sections(&self, project_id: &str) -> Result<Vec<Section>, ApiError> {
        self.get_json(&format!("/projects/{project_id}/sections"))
            .await
    }

    // -- Areas & Tags -------------------------------------------------------

    pub async fn list_areas(&self) -> Result<Vec<Area>, ApiError> {
        self.get_json("/areas").await
    }

    pub async fn list_tags(&self) -> Result<Vec<Tag>, ApiError> {
        self.get_json("/tags").await
    }

    // -- Checklist ----------------------------------------------------------

    pub async fn list_checklist(&self, task_id: &str) -> Result<Vec<ChecklistItem>, ApiError> {
        self.get_json(&format!("/tasks/{task_id}/checklist")).await
    }

    pub async fn add_checklist_item(
        &self,
        task_id: &str,
        title: &str,
    ) -> Result<ChecklistItem, ApiError> {
        let envelope: EventEnvelope<ChecklistItem> = self
            .post_json(
                &format!("/tasks/{task_id}/checklist"),
                &serde_json::json!({"title": title}),
            )
            .await?;
        Ok(envelope.data)
    }

    pub async fn complete_checklist_item(
        &self,
        task_id: &str,
        item_id: &str,
    ) -> Result<(), ApiError> {
        self.post_action(&format!("/tasks/{task_id}/checklist/{item_id}/complete"))
            .await
    }

    pub async fn uncomplete_checklist_item(
        &self,
        task_id: &str,
        item_id: &str,
    ) -> Result<(), ApiError> {
        self.post_action(&format!("/tasks/{task_id}/checklist/{item_id}/uncomplete"))
            .await
    }

    // -- Activity -----------------------------------------------------------

    pub async fn list_activity(&self, task_id: &str) -> Result<Vec<Activity>, ApiError> {
        self.get_json(&format!("/tasks/{task_id}/activity")).await
    }
}
