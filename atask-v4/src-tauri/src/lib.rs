mod commands;
mod db;
mod models;
mod sync;
mod sync_commands;
#[cfg(test)]
mod tests;

use db::Database;
use tauri::menu::{Menu, PredefinedMenuItem, Submenu};
use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut builder = tauri::Builder::default().plugin(tauri_plugin_shell::init());

    #[cfg(debug_assertions)]
    {
        builder = builder.plugin(tauri_plugin_webdriver_automation::init());
    }

    builder
        .menu(|app| {
            Menu::with_items(
                app,
                &[
                    &Submenu::with_items(
                        app,
                        "atask",
                        true,
                        &[
                            &PredefinedMenuItem::about(app, None, None)?,
                            &PredefinedMenuItem::separator(app)?,
                            &PredefinedMenuItem::hide(app, None)?,
                            &PredefinedMenuItem::hide_others(app, None)?,
                            &PredefinedMenuItem::show_all(app, None)?,
                            &PredefinedMenuItem::separator(app)?,
                            &PredefinedMenuItem::quit(app, None)?,
                        ],
                    )?,
                    &Submenu::with_items(
                        app,
                        "Edit",
                        true,
                        &[
                            &PredefinedMenuItem::undo(app, None)?,
                            &PredefinedMenuItem::redo(app, None)?,
                            &PredefinedMenuItem::separator(app)?,
                            &PredefinedMenuItem::cut(app, None)?,
                            &PredefinedMenuItem::copy(app, None)?,
                            &PredefinedMenuItem::paste(app, None)?,
                            &PredefinedMenuItem::select_all(app, None)?,
                        ],
                    )?,
                    &Submenu::with_items(
                        app,
                        "Window",
                        true,
                        &[
                            &PredefinedMenuItem::minimize(app, None)?,
                            &PredefinedMenuItem::maximize(app, None)?,
                            &PredefinedMenuItem::separator(app)?,
                            &PredefinedMenuItem::close_window(app, None)?,
                        ],
                    )?,
                ],
            )
        })
        .setup(|app| {
            let app_dir = app.path().app_data_dir().expect("app data dir");
            std::fs::create_dir_all(&app_dir)?;
            let db_path = app_dir.join("atask.sqlite");
            let database = Database::new(db_path).expect("init database");
            let conn_for_sync = database.conn.clone();
            app.manage(database);
            sync::spawn_sync_worker(conn_for_sync, app.handle().clone());
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::load_all,
            commands::create_task,
            commands::complete_task,
            commands::cancel_task,
            commands::reopen_task,
            commands::update_task,
            commands::duplicate_task,
            commands::delete_task,
            commands::reorder_tasks,
            commands::set_today_index,
            commands::move_task_to_section,
            // Project commands
            commands::create_project,
            commands::update_project,
            commands::complete_project,
            commands::reopen_project,
            commands::delete_project,
            commands::move_project_to_area,
            commands::reorder_projects,
            // Area commands
            commands::create_area,
            commands::update_area,
            commands::delete_area,
            commands::toggle_area_archived,
            commands::reorder_areas,
            // Section commands
            commands::create_section,
            commands::update_section,
            commands::delete_section,
            commands::toggle_section_collapsed,
            commands::toggle_section_archived,
            commands::reorder_sections,
            // Tag commands
            commands::create_tag,
            commands::update_tag,
            commands::delete_tag,
            commands::add_tag_to_task,
            commands::remove_tag_from_task,
            // Checklist commands
            commands::create_checklist_item,
            commands::update_checklist_item,
            commands::toggle_checklist_item,
            commands::delete_checklist_item,
            commands::reorder_checklist_items,
            // Settings commands
            commands::get_settings,
            commands::update_settings,
            // Sync commands
            sync_commands::get_sync_status,
            sync_commands::trigger_sync,
            sync_commands::test_connection,
            sync_commands::initial_sync,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
