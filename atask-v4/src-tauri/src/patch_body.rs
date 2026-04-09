//! Builders for `PATCH` request bodies sent to the Go backend.
//!
//! The Go PATCH handlers use `DisallowUnknownFields`, so any pending op that
//! queues a body containing fields outside the handler's allowed set is
//! rejected with HTTP 400 and the update is silently lost. These helpers
//! produce the exact-shape JSON each handler accepts.
//!
//! Field contracts (must stay in sync with `internal/api/{tasks,projects,areas}.go`):
//! - `PATCH /tasks/{id}`    -> title, notes, schedule, startDate, deadline, projectId, sectionId, areaId
//! - `PATCH /projects/{id}` -> title, notes, color, areaId
//! - `PATCH /areas/{id}`    -> title
//!
//! `Option<String>` fields (start_date, deadline, project_id, ...) are emitted
//! as empty strings (`""`) when `None` -- the Go handler interprets `""` as
//! "clear this field" via its `*string` + empty-string-sentinel convention.

use crate::models::{Area, Project, Task};

/// Build a `PATCH /tasks/{id}` body that exactly matches the Go handler's
/// allowed-fields set. `Option<String>` fields emit `""` when `None` so the
/// Go handler's empty-string-sentinel clears the corresponding server field.
pub fn task_patch_body(task: &Task) -> String {
    #[derive(serde::Serialize)]
    #[serde(rename_all = "camelCase")]
    struct Body<'a> {
        title: &'a str,
        notes: &'a str,
        schedule: i32,
        start_date: &'a str,
        deadline: &'a str,
        project_id: &'a str,
        section_id: &'a str,
        area_id: &'a str,
    }
    let body = Body {
        title: &task.title,
        notes: &task.notes,
        schedule: task.schedule,
        start_date: task.start_date.as_deref().unwrap_or(""),
        deadline: task.deadline.as_deref().unwrap_or(""),
        project_id: task.project_id.as_deref().unwrap_or(""),
        section_id: task.section_id.as_deref().unwrap_or(""),
        area_id: task.area_id.as_deref().unwrap_or(""),
    };
    serde_json::to_string(&body).unwrap_or_default()
}

/// Build a `PATCH /projects/{id}` body. The Go handler also accepts
/// `deadline`, but the Rust `Project` model does not currently store it, so
/// it is omitted here. Add it if/when the model gains the field.
pub fn project_patch_body(project: &Project) -> String {
    #[derive(serde::Serialize)]
    #[serde(rename_all = "camelCase")]
    struct Body<'a> {
        title: &'a str,
        notes: &'a str,
        color: &'a str,
        area_id: &'a str,
    }
    let body = Body {
        title: &project.title,
        notes: &project.notes,
        color: &project.color,
        area_id: project.area_id.as_deref().unwrap_or(""),
    };
    serde_json::to_string(&body).unwrap_or_default()
}

/// Build a `PATCH /areas/{id}` body. The Go handler only accepts `title`.
pub fn area_patch_body(area: &Area) -> String {
    #[derive(serde::Serialize)]
    struct Body<'a> {
        title: &'a str,
    }
    let body = Body { title: &area.title };
    serde_json::to_string(&body).unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::{Area, Project, Task};
    use std::collections::BTreeSet;

    fn json_keys(s: &str) -> BTreeSet<String> {
        let v: serde_json::Value = serde_json::from_str(s).expect("valid JSON");
        v.as_object()
            .expect("JSON object")
            .keys()
            .cloned()
            .collect()
    }

    fn sample_task() -> Task {
        Task {
            id: "t1".into(),
            title: "Task title".into(),
            notes: "Task notes".into(),
            status: 0,
            schedule: 2,
            start_date: Some("2026-04-10".into()),
            deadline: Some("2026-04-15".into()),
            completed_at: None,
            index: 3,
            today_index: Some(1),
            time_slot: Some("morning".into()),
            project_id: Some("p1".into()),
            section_id: Some("s1".into()),
            area_id: Some("a1".into()),
            location_id: Some("l1".into()),
            created_at: "2026-04-09T00:00:00Z".into(),
            updated_at: "2026-04-09T00:00:00Z".into(),
            sync_status: 1,
            repeat_rule: Some("{\"type\":\"daily\"}".into()),
        }
    }

    fn sample_project() -> Project {
        Project {
            id: "p1".into(),
            title: "Project title".into(),
            notes: "Project notes".into(),
            status: 0,
            color: "#ff0000".into(),
            area_id: Some("a1".into()),
            index: 5,
            completed_at: None,
            created_at: "2026-04-09T00:00:00Z".into(),
            updated_at: "2026-04-09T00:00:00Z".into(),
        }
    }

    fn sample_area() -> Area {
        Area {
            id: "a1".into(),
            title: "Area title".into(),
            index: 1,
            archived: false,
            created_at: "2026-04-09T00:00:00Z".into(),
            updated_at: "2026-04-09T00:00:00Z".into(),
        }
    }

    #[test]
    fn task_patch_body_has_exactly_go_patch_fields() {
        let body = task_patch_body(&sample_task());
        let keys = json_keys(&body);
        let expected: BTreeSet<String> = [
            "title",
            "notes",
            "schedule",
            "startDate",
            "deadline",
            "projectId",
            "sectionId",
            "areaId",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect();
        assert_eq!(
            keys, expected,
            "task PATCH body keys must exactly match Go handler's allowed fields"
        );
    }

    #[test]
    fn task_patch_body_excludes_server_only_fields() {
        let body = task_patch_body(&sample_task());
        for forbidden in [
            "id",
            "status",
            "index",
            "todayIndex",
            "timeSlot",
            "locationId",
            "createdAt",
            "updatedAt",
            "syncStatus",
            "repeatRule",
            "completedAt",
        ] {
            assert!(
                !body.contains(&format!("\"{}\"", forbidden)),
                "task PATCH body must not contain server-only field `{}` (got: {})",
                forbidden,
                body
            );
        }
    }

    #[test]
    fn task_patch_body_serializes_values() {
        let body = task_patch_body(&sample_task());
        let v: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(v["title"], "Task title");
        assert_eq!(v["notes"], "Task notes");
        assert_eq!(v["schedule"], 2);
        assert_eq!(v["startDate"], "2026-04-10");
        assert_eq!(v["deadline"], "2026-04-15");
        assert_eq!(v["projectId"], "p1");
        assert_eq!(v["sectionId"], "s1");
        assert_eq!(v["areaId"], "a1");
    }

    #[test]
    fn task_patch_body_emits_empty_string_for_none_options() {
        let mut task = sample_task();
        task.start_date = None;
        task.deadline = None;
        task.project_id = None;
        task.section_id = None;
        task.area_id = None;

        let body = task_patch_body(&task);
        let v: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(v["startDate"], "");
        assert_eq!(v["deadline"], "");
        assert_eq!(v["projectId"], "");
        assert_eq!(v["sectionId"], "");
        assert_eq!(v["areaId"], "");
    }

    #[test]
    fn project_patch_body_has_exactly_go_patch_fields() {
        let body = project_patch_body(&sample_project());
        let keys = json_keys(&body);
        let expected: BTreeSet<String> = ["title", "notes", "color", "areaId"]
            .iter()
            .map(|s| s.to_string())
            .collect();
        assert_eq!(keys, expected);
    }

    #[test]
    fn project_patch_body_excludes_server_only_fields() {
        let body = project_patch_body(&sample_project());
        for forbidden in [
            "id",
            "status",
            "index",
            "completedAt",
            "createdAt",
            "updatedAt",
        ] {
            assert!(
                !body.contains(&format!("\"{}\"", forbidden)),
                "project PATCH body must not contain `{}`",
                forbidden
            );
        }
    }

    #[test]
    fn project_patch_body_serializes_values() {
        let body = project_patch_body(&sample_project());
        let v: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(v["title"], "Project title");
        assert_eq!(v["notes"], "Project notes");
        assert_eq!(v["color"], "#ff0000");
        assert_eq!(v["areaId"], "a1");
    }

    #[test]
    fn project_patch_body_emits_empty_string_for_none_area() {
        let mut project = sample_project();
        project.area_id = None;
        let body = project_patch_body(&project);
        let v: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(v["areaId"], "");
    }

    #[test]
    fn area_patch_body_has_only_title() {
        let body = area_patch_body(&sample_area());
        let keys = json_keys(&body);
        let expected: BTreeSet<String> = ["title"].iter().map(|s| s.to_string()).collect();
        assert_eq!(keys, expected);
    }

    #[test]
    fn area_patch_body_serializes_title() {
        let body = area_patch_body(&sample_area());
        let v: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(v["title"], "Area title");
    }
}
