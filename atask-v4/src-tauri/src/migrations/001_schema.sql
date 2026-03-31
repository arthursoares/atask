CREATE TABLE IF NOT EXISTS areas (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    "index" INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    color TEXT NOT NULL DEFAULT '',
    areaId TEXT REFERENCES areas(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    completedAt TEXT,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sections (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    projectId TEXT NOT NULL REFERENCES projects(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    collapsed INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    schedule INTEGER NOT NULL DEFAULT 0,
    startDate TEXT,
    deadline TEXT,
    completedAt TEXT,
    "index" INTEGER NOT NULL DEFAULT 0,
    todayIndex INTEGER,
    timeSlot TEXT,
    projectId TEXT REFERENCES projects(id),
    sectionId TEXT REFERENCES sections(id),
    areaId TEXT REFERENCES areas(id),
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL,
    syncStatus INTEGER NOT NULL DEFAULT 0,
    repeatRule TEXT
);

CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    "index" INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS taskTags (
    taskId TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tagId TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (taskId, tagId)
);

CREATE TABLE IF NOT EXISTS checklistItems (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    taskId TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    "index" INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pendingOps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    body TEXT,
    createdAt TEXT NOT NULL,
    synced INTEGER NOT NULL DEFAULT 0
);
