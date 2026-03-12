package run

import (
	"fmt"
	"testing"

	"github.com/AdeptMind/pr-emojis-in-slack/internal/config"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/github"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/slack"
)

// --- Mock GitHub Backend ---

type mockGithubBackend struct {
	event   map[string]interface{}
	pr      github.PullRequest
	reviews []github.Review
}

func (m *mockGithubBackend) ReadEvent() (map[string]interface{}, error) {
	return m.event, nil
}

func (m *mockGithubBackend) GetPR(prNumber int) (github.PullRequest, error) {
	return m.pr, nil
}

func (m *mockGithubBackend) GetPRReviews(prNumber int) ([]github.Review, error) {
	return m.reviews, nil
}

// --- Mock Slack Backend ---

type mockSlackBackend struct {
	messages  []slack.Message
	reactions []slack.Reaction
	botUserID string

	// Track the ordered list of emojis on the message (like the Python test).
	emojis []string
}

func (m *mockSlackBackend) GetLatestMessages(channelID string) ([]slack.Message, error) {
	return m.messages, nil
}

func (m *mockSlackBackend) GetReactions(timestamp, channelID string) ([]slack.Reaction, error) {
	return m.reactions, nil
}

func (m *mockSlackBackend) AddReaction(timestamp, emoji, channelID string) error {
	for _, e := range m.emojis {
		if e == emoji {
			return fmt.Errorf("emoji already present: %s", emoji)
		}
	}
	m.emojis = append(m.emojis, emoji)
	return nil
}

func (m *mockSlackBackend) RemoveReaction(timestamp, emoji, channelID string) error {
	for i, e := range m.emojis {
		if e == emoji {
			m.emojis = append(m.emojis[:i], m.emojis[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- Test Helpers ---

var mockEvent = map[string]interface{}{
	"pull_request": map[string]interface{}{
		"number":   float64(42),
		"html_url": "https://github.com/example/repo/pull/42",
		"head": map[string]interface{}{
			"repo": map[string]interface{}{
				"fork": false,
			},
		},
	},
}

var forkEvent = map[string]interface{}{
	"pull_request": map[string]interface{}{
		"number":   float64(42),
		"html_url": "https://github.com/example/repo/pull/42",
		"head": map[string]interface{}{
			"repo": map[string]interface{}{
				"fork": true,
			},
		},
	},
}

func testConfig() *config.Config {
	return &config.Config{
		SlackChannelID:            "C1234",
		BotUserID:                 "U1234",
		NumberOfApprovalsRequired: 1,
		EmojiReviewStarted:       "test_review_started",
		EmojiApproved:            "test_approved",
		EmojiNeedsChange:         "test_needs_change",
		EmojiMerged:              "test_merged",
		EmojiClosed:              "test_closed",
		EmojiCommented:           "test_commented",
	}
}

func openPR() github.PullRequest {
	return github.PullRequest{State: "open", Merged: false, MergeableState: "clean"}
}

func slackMsg() slack.Message {
	return slack.Message{
		Text:      "Need review <https://github.com/example/repo/pull/42>",
		Timestamp: "yyyy-mm-dd",
	}
}

func assertEmojis(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("emoji count: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("emoji[%d]: got %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// --- Tests ---

func TestApproval(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		emojis:   []string{},
	}
	gb := &mockGithubBackend{
		event:   mockEvent,
		pr:      openPR(),
		reviews: []github.Review{{State: "approved", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_approved"})
}

func TestChangesRequested(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		emojis:   []string{},
	}
	gb := &mockGithubBackend{
		event:   mockEvent,
		pr:      openPR(),
		reviews: []github.Review{{State: "changes_requested", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_needs_change"})
}

func TestCommented(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		emojis:   []string{},
	}
	gb := &mockGithubBackend{
		event:   mockEvent,
		pr:      openPR(),
		reviews: []github.Review{{State: "commented", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_commented"})
}

func TestApprovedFromChangesRequested(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		reactions: []slack.Reaction{
			{Emoji: "test_needs_change", UserIDs: []string{"U1234"}},
		},
		emojis: []string{"test_needs_change"},
	}
	gb := &mockGithubBackend{
		event: mockEvent,
		pr:    openPR(),
		reviews: []github.Review{
			{State: "changes_requested", Username: "alice"},
			{State: "approved", Username: "alice"},
		},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_approved"})
}

func TestApprovedFromCommented(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		reactions: []slack.Reaction{
			{Emoji: "test_commented", UserIDs: []string{"U1234"}},
		},
		emojis: []string{"test_commented"},
	}
	gb := &mockGithubBackend{
		event: mockEvent,
		pr:    openPR(),
		reviews: []github.Review{
			{State: "commented", Username: "alice"},
			{State: "approved", Username: "alice"},
		},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_approved"})
}

func TestCommentedIgnoredWhenChangesRequested(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		emojis:   []string{},
	}
	gb := &mockGithubBackend{
		event: mockEvent,
		pr:    openPR(),
		reviews: []github.Review{
			{State: "changes_requested", Username: "alice"},
			{State: "commented", Username: "bob"},
		},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_needs_change"})
}

func TestApprovedIgnoredWhenChangesRequested(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		emojis:   []string{},
	}
	gb := &mockGithubBackend{
		event: mockEvent,
		pr:    openPR(),
		reviews: []github.Review{
			{State: "changes_requested", Username: "alice"},
			{State: "approved", Username: "bob"},
		},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_needs_change"})
}

func TestNoMessageFound(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{
			{Text: "Need :eyes: but I've got no PR URL", Timestamp: "yyyy-mm-dd"},
		},
		emojis: []string{},
	}
	gb := &mockGithubBackend{
		event:   mockEvent,
		pr:      openPR(),
		reviews: []github.Review{{State: "approved", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{})
}

func TestMergedPR(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		reactions: []slack.Reaction{
			{Emoji: "test_review_started", UserIDs: []string{"U1234"}},
			{Emoji: "test_approved", UserIDs: []string{"U1234"}},
		},
		emojis: []string{"test_review_started", "test_approved"},
	}
	gb := &mockGithubBackend{
		event:   mockEvent,
		pr:      github.PullRequest{State: "closed", Merged: true, MergeableState: "clean"},
		reviews: []github.Review{{State: "approved", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_approved", "test_merged"})
}

func TestClosedPR(t *testing.T) {
	sb := &mockSlackBackend{
		messages: []slack.Message{slackMsg()},
		reactions: []slack.Reaction{
			{Emoji: "test_review_started", UserIDs: []string{"U1234"}},
			{Emoji: "test_approved", UserIDs: []string{"U1234"}},
		},
		emojis: []string{"test_review_started", "test_approved"},
	}
	gb := &mockGithubBackend{
		event:   mockEvent,
		pr:      github.PullRequest{State: "closed", Merged: false, MergeableState: "clean"},
		reviews: []github.Review{{State: "approved", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{"test_review_started", "test_approved", "test_closed"})
}

func TestForkPR(t *testing.T) {
	sb := &mockSlackBackend{
		emojis: []string{},
	}
	gb := &mockGithubBackend{
		event:   forkEvent,
		pr:      openPR(),
		reviews: []github.Review{{State: "approved", Username: "alice"}},
	}

	err := Run(testConfig(), github.NewClient(gb), slack.NewClient(sb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmojis(t, sb.emojis, []string{})
}
