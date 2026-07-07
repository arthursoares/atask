package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// statsEntityTables lists the domain tables the admin-statistics module
// aggregates. Hardcoded whitelist, never derived from user input, matching
// the discipline established by OrphanableTables in orphan_check.go. Each
// table has a `deleted` column, so counts filter `deleted = 0`.
var statsEntityTables = []string{"tasks", "projects", "areas", "tags"}

// EntityCounts is the per-account breakdown of live (non-deleted) domain
// entities, returned by PerUserCounts.
type EntityCounts struct {
	Tasks    int
	Projects int
	Areas    int
	Tags     int
}

// Activity is a user's engagement summary derived from domain_events.
type Activity struct {
	LastActiveAt *time.Time
	EventCount   int
}

// Event is a single domain_events row, normalized for display: nullable
// text columns collapse to "" and a nullable timestamp collapses to the
// zero time.Time.
type Event struct {
	Type       string
	EntityType string
	EntityID   string
	ActorID    string
	UserID     string
	Timestamp  time.Time
}

// DayBucket is one day's worth of entity-creation activity, used for the
// growth chart. Date is formatted "YYYY-MM-DD".
type DayBucket struct {
	Date  string
	Count int
}

// SystemStats returns system-wide totals across every account: live task/
// project/area/tag counts and the total number of domain_events ever
// recorded (events are never soft-deleted, so no `deleted` filter applies).
//
// The return type is an anonymous struct (rather than a named `SystemStats`
// type) because Go does not allow a package-level function and a
// package-level type to share the same identifier; the field set below is
// the "SystemStats" shape described in the design spec.
func SystemStats(ctx context.Context, db *sql.DB) (struct {
	Tasks    int
	Projects int
	Areas    int
	Tags     int
	Events   int
}, error) {
	var out struct {
		Tasks    int
		Projects int
		Areas    int
		Tags     int
		Events   int
	}

	for _, table := range statsEntityTables {
		var n int
		// #nosec G201 -- table is one of the hardcoded literals in
		// statsEntityTables above, never user input.
		q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE deleted = 0`, table)
		if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
			return out, fmt.Errorf("count %s: %w", table, err)
		}
		switch table {
		case "tasks":
			out.Tasks = n
		case "projects":
			out.Projects = n
		case "areas":
			out.Areas = n
		case "tags":
			out.Tags = n
		}
	}

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM domain_events`).Scan(&out.Events); err != nil {
		return out, fmt.Errorf("count domain_events: %w", err)
	}

	return out, nil
}

// PerUserCounts returns live entity counts (tasks/projects/areas/tags)
// grouped by user_id. Users with zero entities in a given table simply
// don't contribute to that field (it stays at the zero value).
func PerUserCounts(ctx context.Context, db *sql.DB) (map[string]EntityCounts, error) {
	out := make(map[string]EntityCounts)

	for _, table := range statsEntityTables {
		// #nosec G201 -- table is one of the hardcoded literals in
		// statsEntityTables above, never user input.
		q := fmt.Sprintf(`SELECT user_id, COUNT(*) FROM %s WHERE deleted = 0 GROUP BY user_id`, table)
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("count %s by user: %w", table, err)
		}

		for rows.Next() {
			var userID string
			var n int
			if err := rows.Scan(&userID, &n); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan %s count: %w", table, err)
			}
			c := out[userID]
			switch table {
			case "tasks":
				c.Tasks = n
			case "projects":
				c.Projects = n
			case "areas":
				c.Areas = n
			case "tags":
				c.Tags = n
			}
			out[userID] = c
		}
		if err := rows.Close(); err != nil {
			return nil, fmt.Errorf("close %s rows: %w", table, err)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate %s rows: %w", table, err)
		}
	}

	return out, nil
}

// UserActivity returns, per user_id, the last time they generated a
// domain_event and how many they've generated in total.
//
// This deliberately avoids `MAX(timestamp)` in SQL: the modernc.org/sqlite
// driver only converts a DATETIME column to time.Time when the column's
// declared type is visible to it (a plain `SELECT timestamp FROM ...`); an
// aggregate like MAX() erases that type information and the driver hands
// back a raw string that a *time.Time/sql.NullTime destination cannot
// consume ("unsupported Scan ... storing driver.Value type string into
// type *time.Time"). Reading raw (user_id, timestamp) rows and computing
// max-per-user in Go sidesteps the issue entirely and was verified against
// this driver version (modernc.org/sqlite v1.52.0).
func UserActivity(ctx context.Context, db *sql.DB) (map[string]Activity, error) {
	rows, err := db.QueryContext(ctx, `SELECT user_id, timestamp FROM domain_events`)
	if err != nil {
		return nil, fmt.Errorf("select domain_events: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	last := make(map[string]time.Time)
	for rows.Next() {
		var userID string
		var ts sql.NullTime
		if err := rows.Scan(&userID, &ts); err != nil {
			return nil, fmt.Errorf("scan domain_events row: %w", err)
		}
		counts[userID]++
		if ts.Valid {
			if cur, ok := last[userID]; !ok || ts.Time.After(cur) {
				last[userID] = ts.Time
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domain_events rows: %w", err)
	}

	out := make(map[string]Activity, len(counts))
	for userID, n := range counts {
		a := Activity{EventCount: n}
		if t, ok := last[userID]; ok {
			t := t // copy for the pointer
			a.LastActiveAt = &t
		}
		out[userID] = a
	}
	return out, nil
}

// scanEvents reads the common (type, entity_type, entity_id, actor_id,
// user_id, timestamp) row shape shared by RecentEvents and
// RecentEventsByUser, normalizing NULLs away.
func scanEvents(rows *sql.Rows) ([]Event, error) {
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var (
			eventType  sql.NullString
			entityType sql.NullString
			entityID   sql.NullString
			actorID    sql.NullString
			userID     string
			ts         sql.NullTime
		)
		if err := rows.Scan(&eventType, &entityType, &entityID, &actorID, &userID, &ts); err != nil {
			return nil, fmt.Errorf("scan domain_events row: %w", err)
		}
		ev := Event{
			Type:       eventType.String,
			EntityType: entityType.String,
			EntityID:   entityID.String,
			ActorID:    actorID.String,
			UserID:     userID,
		}
		if ts.Valid {
			ev.Timestamp = ts.Time
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domain_events rows: %w", err)
	}
	return events, nil
}

// RecentEvents returns the most recent domain_events across all users,
// newest first, capped at limit.
func RecentEvents(ctx context.Context, db *sql.DB, limit int) ([]Event, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT type, entity_type, entity_id, actor_id, user_id, timestamp
		FROM domain_events
		ORDER BY id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("select recent domain_events: %w", err)
	}
	return scanEvents(rows)
}

// RecentEventsByUser returns the most recent domain_events for a single
// user_id, newest first, capped at limit.
func RecentEventsByUser(ctx context.Context, db *sql.DB, userID string, limit int) ([]Event, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT type, entity_type, entity_id, actor_id, user_id, timestamp
		FROM domain_events
		WHERE user_id = ?
		ORDER BY id DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select recent domain_events for user: %w", err)
	}
	return scanEvents(rows)
}

// CreationGrowth buckets domain_events by calendar day for the last `days`
// days (today inclusive), zero-filling any day with no events so the
// returned slice is continuous, oldest first, with exactly `days` entries.
// days <= 0 defaults to 30, matching the design spec.
//
// Like UserActivity, this reads raw timestamp rows and buckets in Go rather
// than using SQL's date()/strftime(): a probe against this driver showed
// date(timestamp) returns NULL for the RFC3339Nano-with-6-digit-fractional-
// -seconds strings time.Time values are stored as here (SQLite's date()
// expects exactly 3 fractional digits), so it silently drops every row.
func CreationGrowth(ctx context.Context, db *sql.DB, days int) ([]DayBucket, error) {
	if days <= 0 {
		days = 30
	}

	rows, err := db.QueryContext(ctx, `SELECT timestamp FROM domain_events WHERE timestamp IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("select domain_events timestamps: %w", err)
	}
	defer rows.Close()

	const dayLayout = "2006-01-02"
	counts := make(map[string]int)
	for rows.Next() {
		var ts sql.NullTime
		if err := rows.Scan(&ts); err != nil {
			return nil, fmt.Errorf("scan timestamp: %w", err)
		}
		if !ts.Valid {
			continue
		}
		counts[ts.Time.Format(dayLayout)]++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domain_events rows: %w", err)
	}

	out := make([]DayBucket, days)
	end := time.Now()
	for i := 0; i < days; i++ {
		day := end.AddDate(0, 0, -(days - 1 - i))
		key := day.Format(dayLayout)
		out[i] = DayBucket{Date: key, Count: counts[key]}
	}
	return out, nil
}
