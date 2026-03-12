package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the pr-emojis-in-slack action.
type Config struct {
	SlackChannelID string
	BotUserID      string

	NumberOfApprovalsRequired int

	EmojiReviewStarted string
	EmojiApproved      string
	EmojiNeedsChange   string
	EmojiMerged        string
	EmojiClosed        string
	EmojiCommented     string

	// These are used by the clients, not the orchestration logic directly.
	SlackAPIToken   string
	GithubToken     string
	GithubEventPath string
	GithubRepo      string
}

// EmojisByReviewStep returns a sort key for the given emoji based on the
// typical review lifecycle order.
func (c *Config) EmojisByReviewStep(emoji string) int {
	order := []string{
		c.EmojiReviewStarted,
		c.EmojiCommented,
		c.EmojiNeedsChange,
		c.EmojiApproved,
		c.EmojiClosed,
		c.EmojiMerged,
	}
	for i, e := range order {
		if e == emoji {
			return i
		}
	}
	return len(order)
}

// LoadFromEnv reads configuration from environment variables.
func LoadFromEnv() (Config, error) {
	approvals := 1
	if v := os.Getenv("PEIS_NUMBER_OF_APPROVALS_REQUIRED"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PEIS_NUMBER_OF_APPROVALS_REQUIRED: %w", err)
		}
		if n < 1 {
			n = 1
		}
		approvals = n
	}

	cfg := Config{
		SlackAPIToken:   os.Getenv("SLACK_API_TOKEN"),
		GithubToken:     os.Getenv("GITHUB_TOKEN"),
		GithubEventPath: os.Getenv("GITHUB_EVENT_PATH"),
		GithubRepo:      os.Getenv("GITHUB_REPOSITORY"),

		SlackChannelID: os.Getenv("SLACK_CHANNEL_ID"),
		BotUserID:      os.Getenv("PEIS_BOT_USER_ID"),

		NumberOfApprovalsRequired: approvals,

		EmojiReviewStarted: envOrDefault("PEIS_EMOJI_REVIEW_STARTED", "review_started"),
		EmojiApproved:      envOrDefault("PEIS_EMOJI_APPROVED", "approved"),
		EmojiNeedsChange:   envOrDefault("PEIS_EMOJI_CHANGES_REQUESTED", "changes_requested"),
		EmojiMerged:        envOrDefault("PEIS_EMOJI_MERGED", "merged"),
		EmojiClosed:        envOrDefault("PEIS_EMOJI_CLOSED", "closed"),
		EmojiCommented:     envOrDefault("PEIS_EMOJI_COMMENTED", "comment"),
	}

	return cfg, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
