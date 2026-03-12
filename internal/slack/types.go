package slack

// Message represents a Slack message.
type Message struct {
	Text      string
	Timestamp string
}

// Reaction represents a Slack emoji reaction with the users who reacted.
type Reaction struct {
	Emoji   string
	UserIDs []string
}
