CREATE TABLE IF NOT EXISTS taskLinks (
    taskId TEXT NOT NULL,
    linkedTaskId TEXT NOT NULL,
    PRIMARY KEY (taskId, linkedTaskId),
    FOREIGN KEY (taskId) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (linkedTaskId) REFERENCES tasks(id) ON DELETE CASCADE
);
