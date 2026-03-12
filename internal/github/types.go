package github

// Review represents a single PR review.
type Review struct {
	State    string
	Username string
}

// PullRequest represents the relevant state of a GitHub pull request.
type PullRequest struct {
	State          string
	Merged         bool
	MergeableState string
}
