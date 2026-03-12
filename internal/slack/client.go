package slack

import (
	"regexp"
	"strings"
)

var prURLPattern = regexp.MustCompile(`<(?P<url>[^>]+)>`)

// Backend defines the interface for Slack data access.
type Backend interface {
	GetLatestMessages(channelID string) ([]Message, error)
	GetReactions(timestamp, channelID string) ([]Reaction, error)
	AddReaction(timestamp, emoji, channelID string) error
	RemoveReaction(timestamp, emoji, channelID string) error
}

// Client wraps a Backend to provide Slack operations.
type Client struct {
	backend Backend
}

// NewClient creates a Client with the given backend.
func NewClient(backend Backend) *Client {
	return &Client{backend: backend}
}

// FindTimestampOfReviewRequestedMessage searches recent messages for one
// containing the given PR URL and returns its timestamp.
func (c *Client) FindTimestampOfReviewRequestedMessage(prURL, channelID string) (string, error) {
	messages, err := c.backend.GetLatestMessages(channelID)
	if err != nil {
		return "", err
	}

	for _, msg := range messages {
		matches := prURLPattern.FindAllStringSubmatch(msg.Text, -1)
		for _, match := range matches {
			// Slack formats links as <url> or <url|label> — strip the label.
			url := match[1]
			if i := strings.Index(url, "|"); i >= 0 {
				url = url[:i]
			}
			if strings.HasPrefix(url, prURL) {
				return msg.Timestamp, nil
			}
		}
	}

	return "", nil
}

// GetEmojisForUser returns the set of emoji names the given user has reacted
// with on the specified message.
func (c *Client) GetEmojisForUser(timestamp, channelID, userID string) (map[string]struct{}, error) {
	reactions, err := c.backend.GetReactions(timestamp, channelID)
	if err != nil {
		return nil, err
	}

	emojis := make(map[string]struct{})
	for _, r := range reactions {
		for _, uid := range r.UserIDs {
			if uid == userID {
				emojis[r.Emoji] = struct{}{}
				break
			}
		}
	}
	return emojis, nil
}

// AddReaction adds an emoji reaction to a message.
func (c *Client) AddReaction(timestamp, emoji, channelID string) error {
	return c.backend.AddReaction(timestamp, emoji, channelID)
}

// RemoveReaction removes an emoji reaction from a message.
func (c *Client) RemoveReaction(timestamp, emoji, channelID string) error {
	return c.backend.RemoveReaction(timestamp, emoji, channelID)
}
