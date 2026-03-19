package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	DB *sql.DB
}

func NewDB(path string) (*DB, error) {
	var dsn string
	if path == ":memory:" {
		dsn = ":memory:?_foreign_keys=on"
	} else {
		dsn = fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path)
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// SQLite supports only one writer at a time
	sqlDB.SetMaxOpenConns(1)

	return &DB{DB: sqlDB}, nil
}

func (d *DB) Migrate() error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(d.DB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func (d *DB) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

func (d *DB) Close() error {
	return d.DB.Close()
}
