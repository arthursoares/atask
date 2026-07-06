package store

import (
	"context"
	"testing"
	"time"
)

// --- seed helpers -----------------------------------------------------

func insertStatsTask(t *testing.T, db *DB, id, userID string, deleted int) {
	t.Helper()
	_, err := db.DB.Exec(`INSERT INTO tasks (id, user_id, title, deleted, created_at, updated_at)
		VALUES (?, ?, 'a task', ?, datetime('now'), datetime('now'))`, id, userID, deleted)
	if err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}

func insertStatsProject(t *testing.T, db *DB, id, userID string) {
	t.Helper()
	_, err := db.DB.Exec(`INSERT INTO projects (id, user_id, title, created_at, updated_at)
		VALUES (?, ?, 'a project', datetime('now'), datetime('now'))`, id, userID)
	if err != nil {
		t.Fatalf("insert project %s: %v", id, err)
	}
}

func insertStatsArea(t *testing.T, db *DB, id, userID string) {
	t.Helper()
	_, err := db.DB.Exec(`INSERT INTO areas (id, user_id, title, created_at, updated_at)
		VALUES (?, ?, 'an area', datetime('now'), datetime('now'))`, id, userID)
	if err != nil {
		t.Fatalf("insert area %s: %v", id, err)
	}
}

func insertStatsTag(t *testing.T, db *DB, id, userID, title string) {
	t.Helper()
	_, err := db.DB.Exec(`INSERT INTO tags (id, user_id, title, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, id, userID, title)
	if err != nil {
		t.Fatalf("insert tag %s: %v", id, err)
	}
}

// insertStatsEvent inserts a domain_events row. A zero ts is stored as NULL
// (timestamp unknown); an empty actorID is stored as NULL (no actor).
func insertStatsEvent(t *testing.T, db *DB, eventType, entityType, entityID, actorID, userID string, ts time.Time) {
	t.Helper()
	var tsArg any
	if ts.IsZero() {
		tsArg = nil
	} else {
		tsArg = ts
	}
	var actorArg any
	if actorID == "" {
		actorArg = nil
	} else {
		actorArg = actorID
	}
	_, err := db.DB.Exec(`INSERT INTO domain_events (type, entity_type, entity_id, actor_id, payload, timestamp, user_id)
		VALUES (?, ?, ?, ?, '{}', ?, ?)`, eventType, entityType, entityID, actorArg, tsArg, userID)
	if err != nil {
		t.Fatalf("insert domain_event %s/%s: %v", eventType, entityID, err)
	}
}

// seedStatsDB builds a fixture with two users (userA, userB) across all
// stats-relevant tables, plus a handful of domain_events with known
// timestamps relative to "now" (so CreationGrowth's day-bucketing lines up
// with the actual day the test runs on). Returns the reference "now" used
// to seed events, and the exact "today"/"yesterday"/"fiveDaysAgo" instants.
func seedStatsDB(t *testing.T) (db *DB, now, yesterday, fiveDaysAgo time.Time) {
	t.Helper()
	db = newTestDB(t)

	// Entities: userA has more of everything except areas; userB has fewer
	// tasks/no projects but more areas. One deleted task for userA must be
	// excluded from every count.
	insertStatsTask(t, db, "task-a1", "userA", 0)
	insertStatsTask(t, db, "task-a2", "userA", 0)
	insertStatsTask(t, db, "task-a3", "userA", 0)
	insertStatsTask(t, db, "task-a4-deleted", "userA", 1)
	insertStatsTask(t, db, "task-b1", "userB", 0)

	insertStatsProject(t, db, "proj-a1", "userA")
	insertStatsProject(t, db, "proj-a2", "userA")

	insertStatsArea(t, db, "area-a1", "userA")
	insertStatsArea(t, db, "area-b1", "userB")
	insertStatsArea(t, db, "area-b2", "userB")

	insertStatsTag(t, db, "tag-a1", "userA", "Work")
	insertStatsTag(t, db, "tag-b1", "userB", "Work")

	// Events: id order below is insertion order == ascending autoincrement
	// id, which is what RecentEvents/RecentEventsByUser order by.
	now = time.Now()
	yesterday = now.AddDate(0, 0, -1)
	fiveDaysAgo = now.AddDate(0, 0, -5)

	// id 1: userB, 5 days ago
	insertStatsEvent(t, db, "area.created", "area", "area-b1", "userB", "userB", fiveDaysAgo)
	// id 2,3: userA, yesterday
	insertStatsEvent(t, db, "task.created", "task", "t-y1", "userA", "userA", yesterday)
	insertStatsEvent(t, db, "task.updated", "task", "t-y2", "userA", "userA", yesterday)
	// id 4,5,6: userA, today
	insertStatsEvent(t, db, "task.created", "task", "t-1", "userA", "userA", now)
	insertStatsEvent(t, db, "task.created", "task", "t-2", "userA", "userA", now)
	insertStatsEvent(t, db, "task.created", "task", "t-3", "userA", "userA", now)
	// id 7: userB, NULL actor_id + NULL timestamp (must scan safely and be
	// excluded from CreationGrowth's day buckets and UserActivity's
	// LastActiveAt).
	insertStatsEvent(t, db, "project.created", "project", "p-null", "", "userB", time.Time{})

	return db, now, yesterday, fiveDaysAgo
}

// --- tests --------------------------------------------------------------

func TestSystemStats(t *testing.T) {
	db, _, _, _ := seedStatsDB(t)
	got, err := SystemStats(context.Background(), db.DB)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tasks != 4 { // 3 userA + 1 userB, deleted excluded
		t.Errorf("Tasks = %d, want 4", got.Tasks)
	}
	if got.Projects != 2 {
		t.Errorf("Projects = %d, want 2", got.Projects)
	}
	if got.Areas != 3 {
		t.Errorf("Areas = %d, want 3", got.Areas)
	}
	if got.Tags != 2 {
		t.Errorf("Tags = %d, want 2", got.Tags)
	}
	if got.Events != 7 {
		t.Errorf("Events = %d, want 7", got.Events)
	}
}

func TestPerUserCounts(t *testing.T) {
	db, _, _, _ := seedStatsDB(t)
	got, err := PerUserCounts(context.Background(), db.DB)
	if err != nil {
		t.Fatal(err)
	}

	userA, ok := got["userA"]
	if !ok {
		t.Fatal("expected userA in PerUserCounts result")
	}
	userB, ok := got["userB"]
	if !ok {
		t.Fatal("expected userB in PerUserCounts result")
	}

	wantA := EntityCounts{Tasks: 3, Projects: 2, Areas: 1, Tags: 1}
	wantB := EntityCounts{Tasks: 1, Projects: 0, Areas: 2, Tags: 1}
	if userA != wantA {
		t.Errorf("userA counts = %+v, want %+v", userA, wantA)
	}
	if userB != wantB {
		t.Errorf("userB counts = %+v, want %+v", userB, wantB)
	}
	if userA == userB {
		t.Errorf("expected per-user isolation: userA counts %+v should differ from userB %+v", userA, userB)
	}
}

func TestUserActivity(t *testing.T) {
	db, now, _, fiveDaysAgo := seedStatsDB(t)
	got, err := UserActivity(context.Background(), db.DB)
	if err != nil {
		t.Fatal(err)
	}

	userA, ok := got["userA"]
	if !ok {
		t.Fatal("expected userA in UserActivity result")
	}
	if userA.EventCount != 5 { // 2 yesterday + 3 today
		t.Errorf("userA.EventCount = %d, want 5", userA.EventCount)
	}
	if userA.LastActiveAt == nil {
		t.Fatal("expected userA.LastActiveAt to be set")
	}
	if diff := userA.LastActiveAt.Sub(now); diff < -time.Second || diff > time.Second {
		t.Errorf("userA.LastActiveAt = %v, want ~%v", *userA.LastActiveAt, now)
	}

	userB, ok := got["userB"]
	if !ok {
		t.Fatal("expected userB in UserActivity result")
	}
	if userB.EventCount != 2 { // 5-days-ago event + the null-timestamp event
		t.Errorf("userB.EventCount = %d, want 2", userB.EventCount)
	}
	if userB.LastActiveAt == nil {
		t.Fatal("expected userB.LastActiveAt to be set (from the 5-days-ago event, not the NULL-timestamp one)")
	}
	if diff := userB.LastActiveAt.Sub(fiveDaysAgo); diff < -time.Second || diff > time.Second {
		t.Errorf("userB.LastActiveAt = %v, want ~%v (NULL-timestamp event must not win)", *userB.LastActiveAt, fiveDaysAgo)
	}
}

func TestRecentEvents(t *testing.T) {
	db, _, _, _ := seedStatsDB(t)
	got, err := RecentEvents(context.Background(), db.DB, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len(RecentEvents) = %d, want 3", len(got))
	}

	// Newest first by id: the NULL-actor/timestamp event (id 7) is the most
	// recent, followed by today's t-3 (id 6) and t-2 (id 5).
	if got[0].EntityID != "p-null" || got[0].UserID != "userB" {
		t.Errorf("got[0] = %+v, want EntityID=p-null UserID=userB", got[0])
	}
	if got[0].ActorID != "" {
		t.Errorf("got[0].ActorID = %q, want empty (NULL actor_id must scan safely)", got[0].ActorID)
	}
	if !got[0].Timestamp.IsZero() {
		t.Errorf("got[0].Timestamp = %v, want zero value (NULL timestamp must scan safely)", got[0].Timestamp)
	}
	if got[1].EntityID != "t-3" {
		t.Errorf("got[1].EntityID = %q, want t-3", got[1].EntityID)
	}
	if got[2].EntityID != "t-2" {
		t.Errorf("got[2].EntityID = %q, want t-2", got[2].EntityID)
	}
}

func TestRecentEventsByUser(t *testing.T) {
	db, _, _, _ := seedStatsDB(t)

	gotA, err := RecentEventsByUser(context.Background(), db.DB, "userA", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotA) != 5 {
		t.Fatalf("len(RecentEventsByUser userA) = %d, want 5", len(gotA))
	}
	for _, ev := range gotA {
		if ev.UserID != "userA" {
			t.Errorf("RecentEventsByUser(userA) leaked event for %q", ev.UserID)
		}
	}
	if gotA[0].EntityID != "t-3" {
		t.Errorf("gotA[0].EntityID = %q, want t-3 (newest first)", gotA[0].EntityID)
	}

	gotB, err := RecentEventsByUser(context.Background(), db.DB, "userB", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotB) != 2 {
		t.Fatalf("len(RecentEventsByUser userB) = %d, want 2", len(gotB))
	}
	for _, ev := range gotB {
		if ev.UserID != "userB" {
			t.Errorf("RecentEventsByUser(userB) leaked event for %q", ev.UserID)
		}
	}
}

func TestCreationGrowth(t *testing.T) {
	db, _, _, _ := seedStatsDB(t)
	got, err := CreationGrowth(context.Background(), db.DB, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 7 {
		t.Fatalf("len(CreationGrowth) = %d, want 7", len(got))
	}

	// Oldest -> newest: index 6 is today, index 5 is yesterday, index 1 is
	// five days ago. Indices 0, 2, 3, 4 must be zero-filled.
	today := got[6]
	wantToday := time.Now().Format("2006-01-02")
	if today.Date != wantToday {
		t.Errorf("got[6].Date = %q, want %q (today)", today.Date, wantToday)
	}
	if today.Count != 3 {
		t.Errorf("got[6].Count = %d, want 3 (today's events; NULL-timestamp event excluded)", today.Count)
	}

	yesterday := got[5]
	if yesterday.Count != 2 {
		t.Errorf("got[5].Count = %d, want 2 (yesterday's events)", yesterday.Count)
	}

	fiveDaysAgoBucket := got[1]
	if fiveDaysAgoBucket.Count != 1 {
		t.Errorf("got[1].Count = %d, want 1 (five-days-ago event)", fiveDaysAgoBucket.Count)
	}

	for _, idx := range []int{0, 2, 3, 4} {
		if got[idx].Count != 0 {
			t.Errorf("got[%d].Count = %d, want 0 (zero-filled empty day)", idx, got[idx].Count)
		}
		if len(got[idx].Date) != len("2006-01-02") {
			t.Errorf("got[%d].Date = %q, want YYYY-MM-DD formatted date even when empty", idx, got[idx].Date)
		}
	}

	// Sum across all buckets must equal the 6 events that had a non-NULL
	// timestamp (7 total events minus the 1 NULL-timestamp event) -- this is
	// the "CreationGrowth actually returns nonzero counts for seeded events"
	// check called out in the task brief.
	sum := 0
	for _, b := range got {
		sum += b.Count
	}
	if sum != 6 {
		t.Errorf("sum(CreationGrowth counts) = %d, want 6", sum)
	}
}

func TestCreationGrowth_DefaultsWhenDaysNonPositive(t *testing.T) {
	db, _, _, _ := seedStatsDB(t)
	got, err := CreationGrowth(context.Background(), db.DB, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 30 {
		t.Fatalf("len(CreationGrowth(days=0)) = %d, want 30 (default)", len(got))
	}
}
