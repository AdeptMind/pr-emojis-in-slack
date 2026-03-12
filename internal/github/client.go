package github

// Backend defines the interface for GitHub data access.
type Backend interface {
	ReadEvent() (map[string]interface{}, error)
	GetPR(prNumber int) (PullRequest, error)
	GetPRReviews(prNumber int) ([]Review, error)
}

// Client wraps a Backend to provide GitHub operations.
type Client struct {
	backend Backend
}

// NewClient creates a Client with the given backend.
func NewClient(backend Backend) *Client {
	return &Client{backend: backend}
}

// ReadEvent reads the GitHub event payload.
func (c *Client) ReadEvent() (map[string]interface{}, error) {
	return c.backend.ReadEvent()
}

// GetPR retrieves pull request details.
func (c *Client) GetPR(prNumber int) (PullRequest, error) {
	return c.backend.GetPR(prNumber)
}

// GetPRReviews retrieves reviews for a pull request.
func (c *Client) GetPRReviews(prNumber int) ([]Review, error) {
	return c.backend.GetPRReviews(prNumber)
}
