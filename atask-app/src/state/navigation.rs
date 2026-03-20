#[derive(Debug, Clone, PartialEq)]
pub enum ActiveView {
    Inbox,
    Today,
    Upcoming,
    Someday,
    Logbook,
    Project(String),
}

impl Default for ActiveView {
    fn default() -> Self {
        Self::Today
    }
}
