//! Integration tests for the atask API client.
//!
//! These tests run against a live Go server at localhost:8080.
//! Start the server before running:
//!
//!   cd /path/to/openthings && go run ./cmd/atask serve
//!
//! Run with:
//!   cargo test --test api_integration -- --test-threads=1

use atask::api::client::ApiClient;

/// Helper: create client, register a unique user, login, return authenticated client.
async fn setup_client() -> ApiClient {
    let mut client = ApiClient::new("http://localhost:8080");
    let email = format!("test-{}@test.com", uuid::Uuid::new_v4());
    client
        .register(&email, "testpass", "Test User")
        .await
        .unwrap();
    let token = client.login(&email, "testpass").await.unwrap();
    client.set_token(token);
    client
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

#[tokio::test]
async fn test_auth_flow() {
    let client = setup_client().await;
    assert!(client.has_token());
}

#[tokio::test]
async fn test_login_wrong_password() {
    let client = ApiClient::new("http://localhost:8080");
    let email = format!("test-{}@test.com", uuid::Uuid::new_v4());
    client
        .register(&email, "testpass", "Test User")
        .await
        .unwrap();

    let result = client.login(&email, "wrongpass").await;
    assert!(result.is_err());
}

// ---------------------------------------------------------------------------
// Task CRUD
// ---------------------------------------------------------------------------

#[tokio::test]
async fn test_task_crud() {
    let client = setup_client().await;

    // Create
    let task = client.create_task("Integration test task").await.unwrap();
    assert_eq!(task.title, "Integration test task");
    assert_eq!(task.status, 0); // pending

    // Update title
    client
        .update_task_title(&task.id, "Updated title")
        .await
        .unwrap();

    // Update notes
    client
        .update_task_notes(&task.id, "Some notes")
        .await
        .unwrap();

    // Complete
    client.complete_task(&task.id).await.unwrap();

    // Should appear in logbook
    let logbook = client.list_logbook().await.unwrap();
    assert!(logbook.iter().any(|t| t.id == task.id));

    // Delete
    client.delete_task(&task.id).await.unwrap();
}

// ---------------------------------------------------------------------------
// Schedule management
// ---------------------------------------------------------------------------

#[tokio::test]
async fn test_schedule_management() {
    let client = setup_client().await;

    let task = client.create_task("Schedule test").await.unwrap();
    assert_eq!(task.schedule, 0); // inbox default

    // Should be in inbox
    let inbox = client.list_inbox().await.unwrap();
    assert!(inbox.iter().any(|t| t.id == task.id));

    // Move to anytime (today view)
    client
        .update_task_schedule(&task.id, "anytime")
        .await
        .unwrap();

    // Should be in today
    let today = client.list_today().await.unwrap();
    assert!(today.iter().any(|t| t.id == task.id));

    // Move to someday
    client
        .update_task_schedule(&task.id, "someday")
        .await
        .unwrap();

    let someday = client.list_someday().await.unwrap();
    assert!(someday.iter().any(|t| t.id == task.id));

    // Cleanup
    client.delete_task(&task.id).await.unwrap();
}

// ---------------------------------------------------------------------------
// Project workflow
// ---------------------------------------------------------------------------

#[tokio::test]
async fn test_project_workflow() {
    let client = setup_client().await;

    // Create project
    let project = client.create_project("Test Project").await.unwrap();
    assert_eq!(project.title, "Test Project");

    // Create task and move to project
    let task = client.create_task("Project task").await.unwrap();
    client
        .move_task_to_project(&task.id, Some(&project.id))
        .await
        .unwrap();

    // List tasks by project
    let tasks = client.list_tasks_by_project(&project.id).await.unwrap();
    assert!(tasks.iter().any(|t| t.id == task.id));

    // Cleanup
    client.delete_task(&task.id).await.unwrap();
}

// ---------------------------------------------------------------------------
// Checklist
// ---------------------------------------------------------------------------

#[tokio::test]
async fn test_checklist() {
    let client = setup_client().await;

    let task = client.create_task("Checklist test").await.unwrap();

    // Add checklist item
    let item = client
        .add_checklist_item(&task.id, "Step 1")
        .await
        .unwrap();
    assert_eq!(item.title, "Step 1");
    assert_eq!(item.status, 0); // pending

    // Complete it
    client
        .complete_checklist_item(&task.id, &item.id)
        .await
        .unwrap();

    // Verify
    let items = client.list_checklist(&task.id).await.unwrap();
    assert!(items.iter().any(|i| i.id == item.id && i.is_completed()));

    // Cleanup
    client.delete_task(&task.id).await.unwrap();
}

// ---------------------------------------------------------------------------
// Views return data
// ---------------------------------------------------------------------------

#[tokio::test]
async fn test_views_return_data() {
    let client = setup_client().await;

    // These should all succeed (even if empty)
    client.list_inbox().await.unwrap();
    client.list_today().await.unwrap();
    client.list_upcoming().await.unwrap();
    client.list_someday().await.unwrap();
    client.list_logbook().await.unwrap();
}
