use rusqlite::{Connection, Result};
use std::path::PathBuf;
use std::sync::{Arc, Mutex};

pub struct Database {
    pub conn: Arc<Mutex<Connection>>,
}

impl Database {
    pub fn new(path: PathBuf) -> Result<Self> {
        let conn = Connection::open(&path)?;
        conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")?;
        let db = Self {
            conn: Arc::new(Mutex::new(conn)),
        };
        db.migrate()?;
        Ok(db)
    }

    pub fn new_in_memory() -> Result<Self> {
        let conn = Connection::open_in_memory()?;
        conn.execute_batch("PRAGMA foreign_keys=ON;")?;
        let db = Self {
            conn: Arc::new(Mutex::new(conn)),
        };
        db.migrate()?;
        Ok(db)
    }

    fn migrate(&self) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute_batch(include_str!("migrations/001_schema.sql"))?;
        conn.execute_batch(include_str!("migrations/002_settings.sql"))?;
        conn.execute_batch(include_str!("migrations/003_activities.sql"))?;
        conn.execute_batch(include_str!("migrations/004_locations.sql"))?;
        // Add locationId column to tasks if it doesn't exist yet
        Self::add_column_if_missing(&conn, "tasks", "locationId", "TEXT REFERENCES locations(id)")?;
        conn.execute_batch(include_str!("migrations/005_project_tags.sql"))?;
        Ok(())
    }

    fn add_column_if_missing(
        conn: &Connection,
        table: &str,
        column: &str,
        col_type: &str,
    ) -> Result<()> {
        let mut stmt = conn.prepare(&format!("PRAGMA table_info({})", table))?;
        let has_column = stmt
            .query_map([], |row| row.get::<_, String>(1))?
            .any(|name| name.as_deref() == Ok(column));
        if !has_column {
            conn.execute_batch(&format!(
                "ALTER TABLE {} ADD COLUMN {} {}",
                table, column, col_type
            ))?;
        }
        Ok(())
    }
}
