package slack

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// slackResponse is the common envelope for Slack Web API responses.
type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// conversationsHistoryResponse represents the response from conversations.history.
type conversationsHistoryResponse struct {
	slackResponse
	Messages []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		TS   string `json:"ts"`
	} `json:"messages"`
}

// reactionsGetResponse represents the response from reactions.get.
type reactionsGetResponse struct {
	slackResponse
	Type    string `json:"type"`
	Message struct {
		Reactions []struct {
			Name  string   `json:"name"`
			Users []string `json:"users"`
		} `json:"reactions"`
	} `json:"message"`
}

// WebBackend implements Backend using the Slack Web API over HTTP.
type WebBackend struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewWebBackend creates a WebBackend with the given Slack API token.
func NewWebBackend(token string) *WebBackend {
	return &WebBackend{
		token:      token,
		httpClient: &http.Client{},
		baseURL:    "https://slack.com/api",
	}
}

// post sends a POST request to the Slack API and decodes the JSON response.
func (w *WebBackend) post(endpoint string, params url.Values, result interface{}) error {
	resp, err := w.doPost(w.baseURL+"/"+endpoint, params)
	if err != nil {
		return fmt.Errorf("slack %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("slack %s: decode response: %w", endpoint, err)
	}
	return nil
}

func (w *WebBackend) doPost(endpoint string, params url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = params.Encode()

	return w.httpClient.Do(req)
}

// GetLatestMessages retrieves recent messages from a Slack channel.
func (w *WebBackend) GetLatestMessages(channelID string) ([]Message, error) {
	params := url.Values{"channel": {channelID}}

	var resp conversationsHistoryResponse
	if err := w.post("conversations.history", params, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("slack conversations.history: %s", resp.Error)
	}

	var messages []Message
	for _, m := range resp.Messages {
		if m.Type == "message" {
			messages = append(messages, Message{Text: m.Text, Timestamp: m.TS})
		}
	}
	return messages, nil
}

// GetReactions retrieves reactions on a specific message.
func (w *WebBackend) GetReactions(timestamp, channelID string) ([]Reaction, error) {
	params := url.Values{
		"channel":   {channelID},
		"timestamp": {timestamp},
	}

	var resp reactionsGetResponse
	if err := w.post("reactions.get", params, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("slack reactions.get: %s", resp.Error)
	}

	if resp.Type != "message" {
		return nil, nil
	}

	var reactions []Reaction
	for _, r := range resp.Message.Reactions {
		reactions = append(reactions, Reaction{Emoji: r.Name, UserIDs: r.Users})
	}
	return reactions, nil
}

// AddReaction adds an emoji reaction to a message. If the reaction already
// exists, a warning is logged and no error is returned.
func (w *WebBackend) AddReaction(timestamp, emoji, channelID string) error {
	params := url.Values{
		"channel":   {channelID},
		"name":      {emoji},
		"timestamp": {timestamp},
	}

	var resp slackResponse
	if err := w.post("reactions.add", params, &resp); err != nil {
		return err
	}
	if !resp.OK {
		if resp.Error == "already_reacted" {
			log.Printf("Warning: Message %s already has emoji %s in channel %s", timestamp, emoji, channelID)
			return nil
		}
		return fmt.Errorf("slack reactions.add: %s", resp.Error)
	}
	return nil
}

// RemoveReaction removes an emoji reaction from a message.
func (w *WebBackend) RemoveReaction(timestamp, emoji, channelID string) error {
	params := url.Values{
		"channel":   {channelID},
		"name":      {emoji},
		"timestamp": {timestamp},
	}

	var resp slackResponse
	if err := w.post("reactions.remove", params, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("slack reactions.remove: %s", resp.Error)
	}
	return nil
}
