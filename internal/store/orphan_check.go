package store

import (
	"context"
	"database/sql"
	"fmt"
)

// OrphanableTables lists every table migration 005
// (internal/store/migrations/005_multi_user.sql) added a user_id column to:
// 11 domain tables (including join tables) + 2 event tables. Pre-multi-user
// rows in these tables carry user_id = '' and become invisible to every user
// once user-scoped filtering is enforced (Task 6), until claimed via
// `atask admin assign-data`.
//
// This is the single canonical list: OrphanCounts (below) and
// `atask admin assign-data` (cmd/atask/admin_commands.go) both use it, so the
// two can never silently drift apart.
var OrphanableTables = []string{
	// Root domain tables
	"tasks", "projects", "areas", "sections", "tags",
	"locations", "checklist_items", "activities",
	// Join tables (orphaned rows here mean tags/links survive but their
	// ownership relationship is invisible — equally bad)
	"task_tags", "project_tags", "task_links",
	// Event tables (orphaned events would otherwise replay to no one)
	"delta_events", "domain_events",
}

// OrphanCounts returns the row count per table in OrphanableTables where
// user_id = ''. Tables with zero orphaned rows are omitted from the result
// (an empty map means a clean, fully-claimed database). A non-zero count for
// any table indicates pre-multi-user data that has not been claimed via
// `atask admin assign-data`.
func OrphanCounts(ctx context.Context, db *sql.DB) (map[string]int, error) {
	out := make(map[string]int, len(OrphanableTables))
	for _, t := range OrphanableTables {
		var n int
		// #nosec G201 -- table names come from the constant OrphanableTables
		// whitelist above, never from user input.
		q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE user_id = ''`, t)
		if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
			return nil, fmt.Errorf("count %s: %w", t, err)
		}
		if n > 0 {
			out[t] = n
		}
	}
	return out, nil
}

// OrphanTotal sums the per-table counts returned by OrphanCounts. Shared by
// the startup warning (cmd/atask/main.go) and the admin dashboard banner
// (internal/api/admin.go) so both report the same number.
func OrphanTotal(counts map[string]int) int {
	total := 0
	for _, n := range counts {
		total += n
	}
	return total
}
