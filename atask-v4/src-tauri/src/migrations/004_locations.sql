CREATE TABLE IF NOT EXISTS locations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    latitude REAL,
    longitude REAL,
    radius INTEGER,
    address TEXT,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);
