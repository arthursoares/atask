#[cfg(test)]
mod tests {
    use crate::commands::*;
    use crate::db::Database;

    fn setup() -> Database {
        Database::new_in_memory().expect("in-memory db")
    }

    // ── helpers ──────────────────────────────────────────────────────────────

    fn make_task(title: &str) -> CreateTaskParams {
        CreateTaskParams {
            title: title.to_string(),
            notes: None,
            schedule: None,
            start_date: None,
            deadline: None,
            time_slot: None,
            project_id: None,
            section_id: None,
            area_id: None,
            tag_ids: None,
            repeat_rule: None,
        }
    }

    // ── 1. load_all_empty ────────────────────────────────────────────────────

    #[test]
    fn test_load_all_empty() {
        let db = setup();
        let conn = db.conn.lock().unwrap();
        let state = load_all_impl(&conn).unwrap();
        assert!(state.tasks.is_empty());
        assert!(state.projects.is_empty());
        assert!(state.areas.is_empty());
        assert!(state.sections.is_empty());
        assert!(state.tags.is_empty());
        assert!(state.task_tags.is_empty());
        assert!(state.checklist_items.is_empty());
    }

    // ── 2. create and load task ──────────────────────────────────────────────

    #[test]
    fn test_create_and_load_task() {
        let db = setup();
        let conn = db.conn.lock().unwrap();
        let task = create_task_impl(&conn, make_task("Test task")).unwrap();
        assert_eq!(task.title, "Test task");
        assert_eq!(task.status, 0);

        let state = load_all_impl(&conn).unwrap();
        assert_eq!(state.tasks.len(), 1);
        assert_eq!(state.tasks[0].id, task.id);
    }

    // ── 3. create task with tags ─────────────────────────────────────────────

    #[test]
    fn test_create_task_with_tags() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        // First create a tag so the FK is satisfied
        let tag = create_tag_impl(
            &conn,
            CreateTagParams {
                title: "urgent".to_string(),
            },
        )
        .unwrap()
        .unwrap();

        let task = create_task_impl(
            &conn,
            CreateTaskParams {
                tag_ids: Some(vec![tag.id.clone()]),
                ..make_task("Tagged task")
            },
        )
        .unwrap();

        let state = load_all_impl(&conn).unwrap();
        assert_eq!(state.task_tags.len(), 1);
        assert_eq!(state.task_tags[0].task_id, task.id);
        assert_eq!(state.task_tags[0].tag_id, tag.id);
    }

    // ── 4. complete task ─────────────────────────────────────────────────────

    #[test]
    fn test_complete_task() {
        let db = setup();
        let conn = db.conn.lock().unwrap();
        let task = create_task_impl(&conn, make_task("Do it")).unwrap();

        let completed = complete_task_impl(&conn, &task.id).unwrap();
        assert_eq!(completed.status, 1);
        assert!(completed.completed_at.is_some());
    }

    // ── 5. complete repeating task ───────────────────────────────────────────

    #[test]
    fn test_complete_repeating_task() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let repeat_rule = r#"{"type":"fixed","interval":1,"unit":"week"}"#;
        let task = create_task_impl(
            &conn,
            CreateTaskParams {
                start_date: Some("2026-01-01".to_string()),
                repeat_rule: Some(repeat_rule.to_string()),
                ..make_task("Weekly task")
            },
        )
        .unwrap();

        complete_task_impl(&conn, &task.id).unwrap();

        let state = load_all_impl(&conn).unwrap();
        // Original (completed) + new occurrence
        assert_eq!(state.tasks.len(), 2);

        let new_task = state
            .tasks
            .iter()
            .find(|t| t.id != task.id)
            .expect("new task should exist");
        assert_eq!(new_task.status, 0);
        // Next week after 2026-01-01 = 2026-01-08
        assert_eq!(new_task.start_date.as_deref(), Some("2026-01-08"));
    }

    // ── 6. cancel task ───────────────────────────────────────────────────────

    #[test]
    fn test_cancel_task() {
        let db = setup();
        let conn = db.conn.lock().unwrap();
        let task = create_task_impl(&conn, make_task("Abandon")).unwrap();

        let cancelled = cancel_task_impl(&conn, &task.id).unwrap();
        assert_eq!(cancelled.status, 2);
    }

    // ── 7. reopen task ───────────────────────────────────────────────────────

    #[test]
    fn test_reopen_task() {
        let db = setup();
        let conn = db.conn.lock().unwrap();
        let task = create_task_impl(&conn, make_task("Back again")).unwrap();
        complete_task_impl(&conn, &task.id).unwrap();

        let reopened = reopen_task_impl(&conn, &task.id).unwrap();
        assert_eq!(reopened.status, 0);
        assert!(reopened.completed_at.is_none());
    }

    // ── 8. duplicate task ────────────────────────────────────────────────────

    #[test]
    fn test_duplicate_task() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let tag = create_tag_impl(
            &conn,
            CreateTagParams {
                title: "mytag".to_string(),
            },
        )
        .unwrap()
        .unwrap();

        let task = create_task_impl(
            &conn,
            CreateTaskParams {
                tag_ids: Some(vec![tag.id.clone()]),
                ..make_task("Original")
            },
        )
        .unwrap();

        let dup = duplicate_task_impl(&conn, &task.id).unwrap();
        assert_ne!(dup.id, task.id);
        assert_eq!(dup.title, task.title);

        // Tags should be copied
        let state = load_all_impl(&conn).unwrap();
        let dup_tags: Vec<_> = state
            .task_tags
            .iter()
            .filter(|tt| tt.task_id == dup.id)
            .collect();
        assert_eq!(dup_tags.len(), 1);
        assert_eq!(dup_tags[0].tag_id, tag.id);
    }

    // ── 9. delete task cascades ──────────────────────────────────────────────

    #[test]
    fn test_delete_task_cascades() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let tag = create_tag_impl(
            &conn,
            CreateTagParams {
                title: "cascadetag".to_string(),
            },
        )
        .unwrap()
        .unwrap();

        let task = create_task_impl(
            &conn,
            CreateTaskParams {
                tag_ids: Some(vec![tag.id.clone()]),
                ..make_task("To delete")
            },
        )
        .unwrap();

        create_checklist_item_impl(
            &conn,
            CreateChecklistItemParams {
                title: "step 1".to_string(),
                task_id: task.id.clone(),
            },
        )
        .unwrap();

        delete_task_impl(&conn, &task.id).unwrap();

        let state = load_all_impl(&conn).unwrap();
        assert!(state.tasks.is_empty());
        assert!(state.task_tags.is_empty());
        assert!(state.checklist_items.is_empty());
    }

    // ── 10. create project ───────────────────────────────────────────────────

    #[test]
    fn test_create_project() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let project = create_project_impl(
            &conn,
            CreateProjectParams {
                title: "My Project".to_string(),
                color: Some("blue".to_string()),
                area_id: None,
            },
        )
        .unwrap();

        assert_eq!(project.title, "My Project");
        assert_eq!(project.color, "blue");

        let state = load_all_impl(&conn).unwrap();
        assert_eq!(state.projects.len(), 1);
    }

    // ── 11. delete project cascades ──────────────────────────────────────────

    #[test]
    fn test_delete_project_cascades() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let project = create_project_impl(
            &conn,
            CreateProjectParams {
                title: "Doomed Project".to_string(),
                color: None,
                area_id: None,
            },
        )
        .unwrap();

        let task = create_task_impl(
            &conn,
            CreateTaskParams {
                project_id: Some(project.id.clone()),
                ..make_task("Project task")
            },
        )
        .unwrap();

        delete_project_impl(&conn, &project.id).unwrap();

        let state = load_all_impl(&conn).unwrap();
        assert!(state.projects.is_empty());

        // Task should still exist but with projectId = null
        assert_eq!(state.tasks.len(), 1);
        assert_eq!(state.tasks[0].id, task.id);
        assert!(state.tasks[0].project_id.is_none());
    }

    // ── 12. create tag unique ────────────────────────────────────────────────

    #[test]
    fn test_create_tag_unique() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        create_tag_impl(
            &conn,
            CreateTagParams {
                title: "unique".to_string(),
            },
        )
        .unwrap();

        let result = create_tag_impl(
            &conn,
            CreateTagParams {
                title: "unique".to_string(),
            },
        );

        assert!(result.is_err());
        let msg = result.unwrap_err();
        assert!(
            msg.contains("already exists"),
            "expected 'already exists' in error: {msg}"
        );
    }

    // ── 13. checklist CRUD ───────────────────────────────────────────────────

    #[test]
    fn test_checklist_crud() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let task = create_task_impl(&conn, make_task("Task with checklist")).unwrap();

        let item = create_checklist_item_impl(
            &conn,
            CreateChecklistItemParams {
                title: "step 1".to_string(),
                task_id: task.id.clone(),
            },
        )
        .unwrap();

        assert_eq!(item.status, 0);

        // Toggle on
        let toggled = toggle_checklist_item_impl(&conn, &item.id).unwrap();
        assert_eq!(toggled.status, 1);

        // Toggle off
        let toggled2 = toggle_checklist_item_impl(&conn, &item.id).unwrap();
        assert_eq!(toggled2.status, 0);

        // Delete
        delete_checklist_item_impl(&conn, &item.id).unwrap();

        let state = load_all_impl(&conn).unwrap();
        assert!(state.checklist_items.is_empty());
    }

    // ── 14. section collapse toggle ──────────────────────────────────────────

    #[test]
    fn test_section_collapse_toggle() {
        let db = setup();
        let conn = db.conn.lock().unwrap();

        let project = create_project_impl(
            &conn,
            CreateProjectParams {
                title: "Proj".to_string(),
                color: None,
                area_id: None,
            },
        )
        .unwrap();

        let section = create_section_impl(
            &conn,
            CreateSectionParams {
                title: "Section A".to_string(),
                project_id: project.id.clone(),
            },
        )
        .unwrap();

        assert!(!section.collapsed);

        let collapsed = toggle_section_collapsed_impl(&conn, &section.id).unwrap();
        assert!(collapsed.collapsed);

        let expanded = toggle_section_collapsed_impl(&conn, &section.id).unwrap();
        assert!(!expanded.collapsed);
    }
}
