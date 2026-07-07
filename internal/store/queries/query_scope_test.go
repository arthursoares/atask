package queries_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// userOwnedTables: every read/write of these MUST include user_id in its WHERE.
var userOwnedTables = []string{
	"tasks", "projects", "areas", "sections", "tags",
	"locations", "checklist_items", "activities",
	"task_tags", "project_tags", "task_links",
	"delta_events", "domain_events",
}

// hasUserIDPredicate looks for `user_id` followed by a comparison operator
// somewhere in the body (after the first FROM/UPDATE/DELETE keyword).
// It is intentionally lenient about positioning — the goal is "is user_id
// constrained anywhere in this statement" — and strict about *form* (must
// be a predicate, not just a column reference).
var userIDPredicate = regexp.MustCompile(`(?i)user_id\s*(=|in\s*\()`)

// statementOpKind classifies the SQL statement.
func statementOpKind(body string) string {
	trim := strings.TrimSpace(strings.ToLower(body))
	switch {
	case strings.HasPrefix(trim, "insert"):
		return "insert"
	case strings.HasPrefix(trim, "select"):
		return "select"
	case strings.HasPrefix(trim, "update"):
		return "update"
	case strings.HasPrefix(trim, "delete"):
		return "delete"
	default:
		return "other"
	}
}

// touchesUserOwned: returns the matched table name, or "" if none.
func touchesUserOwned(body string) string {
	lower := strings.ToLower(body)
	for _, tbl := range userOwnedTables {
		// word-boundary match: " tasks " or " tasks\n" etc., not "tasks_x"
		idx := strings.Index(lower, tbl)
		for idx >= 0 {
			before := byte(' ')
			after := byte(' ')
			if idx > 0 {
				before = lower[idx-1]
			}
			if idx+len(tbl) < len(lower) {
				after = lower[idx+len(tbl)]
			}
			if !isIdentChar(before) && !isIdentChar(after) {
				return tbl
			}
			next := strings.Index(lower[idx+1:], tbl)
			if next < 0 {
				break
			}
			idx = idx + 1 + next
		}
	}
	return ""
}

func isIdentChar(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func TestAllUserOwnedQueriesScopeByUserID(t *testing.T) {
	skip := map[string]bool{
		"auth.sql":    true, // legacy users table queries are deleted in Task 1.5;
		                     // remaining api_keys queries scope by key_hash, not user_id
		"invites.sql": true, // invites are claimed by token, not scoped by user_id
	}

	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	for _, f := range files {
		if skip[f] {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		queries := strings.Split(string(data), "-- name:")
		for _, q := range queries[1:] {
			lines := strings.SplitN(q, "\n", 2)
			name := strings.TrimSpace(strings.Split(lines[0], ":")[0])
			if len(lines) < 2 {
				continue
			}
			body := lines[1]

			tbl := touchesUserOwned(body)
			if tbl == "" {
				continue
			}
			op := statementOpKind(body)

			switch op {
			case "insert":
				// INSERT must include user_id in the column list
				lower := strings.ToLower(body)
				colsStart := strings.Index(lower, "(")
				colsEnd := strings.Index(lower, ")")
				if colsStart < 0 || colsEnd < 0 || colsEnd < colsStart {
					t.Errorf("%s/%s: INSERT touches %q but column list unparseable", f, name, tbl)
					continue
				}
				cols := lower[colsStart:colsEnd]
				if !strings.Contains(cols, "user_id") {
					t.Errorf("%s/%s: INSERT into %q missing user_id column", f, name, tbl)
				}
			case "select", "update", "delete":
				if !userIDPredicate.MatchString(body) {
					t.Errorf("%s/%s: %s on %q has no `user_id =` or `user_id IN` predicate", f, name, strings.ToUpper(op), tbl)
				}
			}
		}
	}
}

func TestScannerCatchesJoinThroughWithoutPredicate(t *testing.T) {
	// Synthetic ListTaskTags-style query: joins tags via task_tags, but
	// the WHERE clause only filters by task_id — no user_id predicate.
	body := `
SELECT t.* FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = ?;`
	if userIDPredicate.MatchString(body) {
		t.Fatal("scanner should NOT find user_id predicate in this body")
	}
	if touchesUserOwned(body) == "" {
		t.Fatal("scanner should detect that this body touches a user-owned table")
	}
}

func TestScannerAcceptsScopedJoinThrough(t *testing.T) {
	body := `
SELECT t.* FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id AND tt.user_id = ?
WHERE tt.task_id = ? AND tt.user_id = ?;`
	if !userIDPredicate.MatchString(body) {
		t.Fatal("scanner should accept this body as scoped")
	}
}

func TestScannerRejectsProjectionOnly(t *testing.T) {
	// `SELECT user_id, ...` mentions user_id as a column but does not constrain it.
	body := `SELECT user_id, id, title FROM tasks WHERE id = ?;`
	if userIDPredicate.MatchString(body) {
		t.Fatal("scanner should NOT count `SELECT user_id` as a predicate")
	}
}
