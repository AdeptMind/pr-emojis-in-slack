package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// WebBackend implements Backend using the GitHub REST API and a local event file.
type WebBackend struct {
	eventPath  string
	repo       string
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewWebBackend creates a WebBackend.
// eventPath is the path to the GitHub event JSON file.
// repo is the "owner/repo" string.
// token is the GitHub API token.
func NewWebBackend(eventPath, repo, token string) *WebBackend {
	return &WebBackend{
		eventPath:  eventPath,
		repo:       repo,
		token:      token,
		httpClient: http.DefaultClient,
		baseURL:    "https://api.github.com",
	}
}

// ReadEvent reads and parses the GitHub event JSON from the configured file path.
func (w *WebBackend) ReadEvent() (map[string]interface{}, error) {
	data, err := os.ReadFile(w.eventPath)
	if err != nil {
		return nil, fmt.Errorf("reading event file: %w", err)
	}

	var event map[string]interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("parsing event JSON: %w", err)
	}

	return event, nil
}

// GetPR retrieves pull request details from the GitHub REST API.
func (w *WebBackend) GetPR(prNumber int) (PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", w.baseURL, w.repo, prNumber)

	body, err := w.doRequest(url)
	if err != nil {
		return PullRequest{}, fmt.Errorf("getting PR: %w", err)
	}

	var raw struct {
		State          string `json:"state"`
		Merged         bool   `json:"merged"`
		MergeableState string `json:"mergeable_state"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return PullRequest{}, fmt.Errorf("parsing PR response: %w", err)
	}

	return PullRequest{
		State:          raw.State,
		Merged:         raw.Merged,
		MergeableState: raw.MergeableState,
	}, nil
}

// GetPRReviews retrieves reviews for a pull request from the GitHub REST API.
// Review states are lowercased to match the Python implementation convention.
func (w *WebBackend) GetPRReviews(prNumber int) ([]Review, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d/reviews", w.baseURL, w.repo, prNumber)

	body, err := w.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("getting PR reviews: %w", err)
	}

	var raw []struct {
		State string `json:"state"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing reviews response: %w", err)
	}

	reviews := make([]Review, len(raw))
	for i, r := range raw {
		reviews[i] = Review{
			State:    strings.ToLower(r.State),
			Username: r.User.Login,
		}
	}

	return reviews, nil
}

// doRequest performs an authenticated GET request and returns the response body.
func (w *WebBackend) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
