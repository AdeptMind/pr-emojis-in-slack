package slack

import (
	"errors"
	"testing"
)

// mockBackend implements Backend for testing.
type mockBackend struct {
	messages       []Message
	messagesErr    error
	reactions      []Reaction
	reactionsErr   error
	addReactionErr error
	rmReactionErr  error

	// capture calls
	addedReactions   []reactionCall
	removedReactions []reactionCall
}

type reactionCall struct {
	timestamp, emoji, channelID string
}

func (m *mockBackend) GetLatestMessages(channelID string) ([]Message, error) {
	return m.messages, m.messagesErr
}

func (m *mockBackend) GetReactions(timestamp, channelID string) ([]Reaction, error) {
	return m.reactions, m.reactionsErr
}

func (m *mockBackend) AddReaction(timestamp, emoji, channelID string) error {
	m.addedReactions = append(m.addedReactions, reactionCall{timestamp, emoji, channelID})
	return m.addReactionErr
}

func (m *mockBackend) RemoveReaction(timestamp, emoji, channelID string) error {
	m.removedReactions = append(m.removedReactions, reactionCall{timestamp, emoji, channelID})
	return m.rmReactionErr
}

func TestFindTimestampOfReviewRequestedMessage(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		prURL    string
		wantTS   string
		wantErr  bool
	}{
		{
			name: "finds matching message by exact PR URL",
			messages: []Message{
				{Text: "some other text", Timestamp: "111.111"},
				{Text: "Review requested: <https://github.com/owner/repo/pull/42>", Timestamp: "222.222"},
			},
			prURL:  "https://github.com/owner/repo/pull/42",
			wantTS: "222.222",
		},
		{
			name: "matches URL with /files suffix",
			messages: []Message{
				{Text: "PR link: <https://github.com/owner/repo/pull/42/files>", Timestamp: "333.333"},
			},
			prURL:  "https://github.com/owner/repo/pull/42",
			wantTS: "333.333",
		},
		{
			name: "matches URL with arbitrary suffix",
			messages: []Message{
				{Text: "<https://github.com/owner/repo/pull/42/s>", Timestamp: "444.444"},
			},
			prURL:  "https://github.com/owner/repo/pull/42",
			wantTS: "444.444",
		},
		{
			name: "returns empty string when no matching message",
			messages: []Message{
				{Text: "unrelated <https://github.com/owner/repo/pull/99>", Timestamp: "555.555"},
				{Text: "no URL here", Timestamp: "666.666"},
			},
			prURL:  "https://github.com/owner/repo/pull/42",
			wantTS: "",
		},
		{
			name:     "returns empty string with no messages",
			messages: nil,
			prURL:    "https://github.com/owner/repo/pull/42",
			wantTS:   "",
		},
		{
			name: "does not match partial PR number",
			messages: []Message{
				{Text: "<https://github.com/owner/repo/pull/4>", Timestamp: "777.777"},
			},
			prURL:  "https://github.com/owner/repo/pull/42",
			wantTS: "",
		},
		{
			name: "returns first matching message",
			messages: []Message{
				{Text: "<https://github.com/owner/repo/pull/42>", Timestamp: "100.100"},
				{Text: "<https://github.com/owner/repo/pull/42/files>", Timestamp: "200.200"},
			},
			prURL:  "https://github.com/owner/repo/pull/42",
			wantTS: "100.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &mockBackend{messages: tt.messages}
			client := NewClient(backend)

			ts, err := client.FindTimestampOfReviewRequestedMessage(tt.prURL, "C123")
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ts != tt.wantTS {
				t.Errorf("got timestamp %q, want %q", ts, tt.wantTS)
			}
		})
	}
}

func TestFindTimestampOfReviewRequestedMessage_BackendError(t *testing.T) {
	backend := &mockBackend{messagesErr: errors.New("api failure")}
	client := NewClient(backend)

	_, err := client.FindTimestampOfReviewRequestedMessage("https://github.com/owner/repo/pull/1", "C123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetEmojisForUser(t *testing.T) {
	tests := []struct {
		name      string
		reactions []Reaction
		userID    string
		want      map[string]struct{}
	}{
		{
			name: "filters reactions by user ID",
			reactions: []Reaction{
				{Emoji: "thumbsup", UserIDs: []string{"U1", "U2"}},
				{Emoji: "eyes", UserIDs: []string{"U2", "U3"}},
				{Emoji: "rocket", UserIDs: []string{"U1"}},
			},
			userID: "U1",
			want:   map[string]struct{}{"thumbsup": {}, "rocket": {}},
		},
		{
			name: "returns empty set when user has no reactions",
			reactions: []Reaction{
				{Emoji: "thumbsup", UserIDs: []string{"U2"}},
			},
			userID: "U1",
			want:   map[string]struct{}{},
		},
		{
			name:      "returns empty set with no reactions",
			reactions: nil,
			userID:    "U1",
			want:      map[string]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &mockBackend{reactions: tt.reactions}
			client := NewClient(backend)

			got, err := client.GetEmojisForUser("1234.5678", "C123", tt.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d emojis, want %d", len(got), len(tt.want))
			}
			for emoji := range tt.want {
				if _, ok := got[emoji]; !ok {
					t.Errorf("expected emoji %q not found", emoji)
				}
			}
		})
	}
}

func TestGetEmojisForUser_BackendError(t *testing.T) {
	backend := &mockBackend{reactionsErr: errors.New("api failure")}
	client := NewClient(backend)

	_, err := client.GetEmojisForUser("1234.5678", "C123", "U1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddReaction_DelegatesToBackend(t *testing.T) {
	backend := &mockBackend{}
	client := NewClient(backend)

	err := client.AddReaction("1234.5678", "thumbsup", "C123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backend.addedReactions) != 1 {
		t.Fatalf("expected 1 call, got %d", len(backend.addedReactions))
	}
	call := backend.addedReactions[0]
	if call.timestamp != "1234.5678" || call.emoji != "thumbsup" || call.channelID != "C123" {
		t.Errorf("unexpected call args: %+v", call)
	}
}

func TestAddReaction_PropagatesError(t *testing.T) {
	backend := &mockBackend{addReactionErr: errors.New("api failure")}
	client := NewClient(backend)

	err := client.AddReaction("1234.5678", "thumbsup", "C123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveReaction_DelegatesToBackend(t *testing.T) {
	backend := &mockBackend{}
	client := NewClient(backend)

	err := client.RemoveReaction("1234.5678", "thumbsup", "C123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backend.removedReactions) != 1 {
		t.Fatalf("expected 1 call, got %d", len(backend.removedReactions))
	}
	call := backend.removedReactions[0]
	if call.timestamp != "1234.5678" || call.emoji != "thumbsup" || call.channelID != "C123" {
		t.Errorf("unexpected call args: %+v", call)
	}
}

func TestRemoveReaction_PropagatesError(t *testing.T) {
	backend := &mockBackend{rmReactionErr: errors.New("api failure")}
	client := NewClient(backend)

	err := client.RemoveReaction("1234.5678", "thumbsup", "C123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
