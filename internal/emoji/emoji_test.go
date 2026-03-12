package emoji

import (
	"testing"

	"github.com/AdeptMind/pr-emojis-in-slack/internal/github"
)

const (
	emojiCommented   = ":eyes:"
	emojiNeedsChange = ":warning:"
	emojiApproved    = ":white_check_mark:"
)

func TestGetForReviews_SingleApproval(t *testing.T) {
	reviews := []github.Review{
		{State: "approved", Username: "alice"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != emojiApproved {
		t.Errorf("expected %q, got %q", emojiApproved, got)
	}
}

func TestGetForReviews_ChangesRequested(t *testing.T) {
	reviews := []github.Review{
		{State: "changes_requested", Username: "alice"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != emojiNeedsChange {
		t.Errorf("expected %q, got %q", emojiNeedsChange, got)
	}
}

func TestGetForReviews_Commented(t *testing.T) {
	reviews := []github.Review{
		{State: "commented", Username: "alice"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != emojiCommented {
		t.Errorf("expected %q, got %q", emojiCommented, got)
	}
}

func TestGetForReviews_ChangesRequestedTakesPriorityOverApproval(t *testing.T) {
	reviews := []github.Review{
		{State: "changes_requested", Username: "alice"},
		{State: "approved", Username: "bob"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != emojiNeedsChange {
		t.Errorf("expected %q, got %q", emojiNeedsChange, got)
	}
}

func TestGetForReviews_SameAuthorChangesRequestedThenApproved(t *testing.T) {
	reviews := []github.Review{
		{State: "changes_requested", Username: "alice"},
		{State: "approved", Username: "alice"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != emojiApproved {
		t.Errorf("expected %q, got %q", emojiApproved, got)
	}
}

func TestGetForReviews_SameAuthorCommentedThenApproved(t *testing.T) {
	reviews := []github.Review{
		{State: "commented", Username: "alice"},
		{State: "approved", Username: "alice"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != emojiApproved {
		t.Errorf("expected %q, got %q", emojiApproved, got)
	}
}

func TestGetForReviews_NoReviews(t *testing.T) {
	got := GetForReviews(nil, emojiCommented, emojiNeedsChange, emojiApproved, 1)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGetForReviews_OneApprovalWhenTwoRequired(t *testing.T) {
	reviews := []github.Review{
		{State: "approved", Username: "alice"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 2)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGetForReviews_TwoApprovalsWhenTwoRequired(t *testing.T) {
	reviews := []github.Review{
		{State: "approved", Username: "alice"},
		{State: "approved", Username: "bob"},
	}
	got := GetForReviews(reviews, emojiCommented, emojiNeedsChange, emojiApproved, 2)
	if got != emojiApproved {
		t.Errorf("expected %q, got %q", emojiApproved, got)
	}
}

func TestDiff_NewEmojiMinusExisting(t *testing.T) {
	newEmojis := map[string]struct{}{
		":a:": {},
		":b:": {},
	}
	existing := map[string]struct{}{
		":a:": {},
	}
	toAdd, toRemove := Diff(newEmojis, existing)

	if _, ok := toAdd[":b:"]; !ok {
		t.Error("expected :b: in toAdd")
	}
	if len(toAdd) != 1 {
		t.Errorf("expected 1 item in toAdd, got %d", len(toAdd))
	}
	if len(toRemove) != 0 {
		t.Errorf("expected 0 items in toRemove, got %d", len(toRemove))
	}
}

func TestDiff_ExistingMinusNew(t *testing.T) {
	newEmojis := map[string]struct{}{
		":a:": {},
	}
	existing := map[string]struct{}{
		":a:": {},
		":b:": {},
	}
	toAdd, toRemove := Diff(newEmojis, existing)

	if len(toAdd) != 0 {
		t.Errorf("expected 0 items in toAdd, got %d", len(toAdd))
	}
	if _, ok := toRemove[":b:"]; !ok {
		t.Error("expected :b: in toRemove")
	}
	if len(toRemove) != 1 {
		t.Errorf("expected 1 item in toRemove, got %d", len(toRemove))
	}
}

func TestDiff_OverlappingSets(t *testing.T) {
	newEmojis := map[string]struct{}{
		":a:": {},
		":b:": {},
	}
	existing := map[string]struct{}{
		":b:": {},
		":c:": {},
	}
	toAdd, toRemove := Diff(newEmojis, existing)

	if _, ok := toAdd[":a:"]; !ok {
		t.Error("expected :a: in toAdd")
	}
	if len(toAdd) != 1 {
		t.Errorf("expected 1 item in toAdd, got %d", len(toAdd))
	}
	if _, ok := toRemove[":c:"]; !ok {
		t.Error("expected :c: in toRemove")
	}
	if len(toRemove) != 1 {
		t.Errorf("expected 1 item in toRemove, got %d", len(toRemove))
	}
	// Shared :b: should not appear in either set.
	if _, ok := toAdd[":b:"]; ok {
		t.Error(":b: should not be in toAdd")
	}
	if _, ok := toRemove[":b:"]; ok {
		t.Error(":b: should not be in toRemove")
	}
}

func TestDiff_EmptySets(t *testing.T) {
	newEmojis := map[string]struct{}{}
	existing := map[string]struct{}{}
	toAdd, toRemove := Diff(newEmojis, existing)

	if len(toAdd) != 0 {
		t.Errorf("expected 0 items in toAdd, got %d", len(toAdd))
	}
	if len(toRemove) != 0 {
		t.Errorf("expected 0 items in toRemove, got %d", len(toRemove))
	}
}
