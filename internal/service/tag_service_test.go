package service

import (
	"context"
	"testing"
)

// TestTagCreate_SameTitleDifferentUsers_Succeeds is the regression test for
// the P1 finding: idx_tags_title_unique was a global UNIQUE index on
// tags(title), so a second user creating a tag with the same title as an
// existing tag (owned by a different user) got a 500. Migration 007
// rebuilds the index as UNIQUE(user_id, title), so this must now succeed.
func TestTagCreate_SameTitleDifferentUsers_Succeeds(t *testing.T) {
	tags, _ := newTestTagService(t)
	ctx := context.Background()

	if _, err := tags.Create(ctx, "user-a", "work", "actor-a"); err != nil {
		t.Fatalf("user-a create tag %q: %v", "work", err)
	}

	if _, err := tags.Create(ctx, "user-b", "work", "actor-b"); err != nil {
		t.Fatalf("user-b create tag %q: expected success, got %v", "work", err)
	}
}

// TestTagCreate_SameTitleSameUser_Fails ensures the migration preserves
// per-user uniqueness: the same user creating two tags with the same title
// must still fail.
func TestTagCreate_SameTitleSameUser_Fails(t *testing.T) {
	tags, _ := newTestTagService(t)
	ctx := context.Background()

	if _, err := tags.Create(ctx, "user-a", "work", "actor-a"); err != nil {
		t.Fatalf("first create tag %q: %v", "work", err)
	}

	if _, err := tags.Create(ctx, "user-a", "work", "actor-a"); err == nil {
		t.Fatalf("second create tag %q by same user: expected error, got nil", "work")
	}
}
