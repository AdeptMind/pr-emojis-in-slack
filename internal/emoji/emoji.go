package emoji

import (
	"github.com/AdeptMind/pr-emojis-in-slack/internal/github"
)

// GetForReviews determines the appropriate review emoji based on the current
// state of PR reviews. It groups reviews by author, considers only the last
// review per author, and applies priority: changes_requested > approved > commented.
func GetForReviews(
	reviews []github.Review,
	emojiCommented string,
	emojiNeedsChange string,
	emojiApproved string,
	numberOfApprovalsRequired int,
) string {
	if len(reviews) == 0 {
		return ""
	}

	// Group by author, keep order so last review per author wins.
	lastByAuthor := make(map[string]github.Review)
	var authorOrder []string
	for _, r := range reviews {
		if _, seen := lastByAuthor[r.Username]; !seen {
			authorOrder = append(authorOrder, r.Username)
		}
		lastByAuthor[r.Username] = r
	}

	uniqueStates := make(map[string]struct{})
	for _, author := range authorOrder {
		uniqueStates[lastByAuthor[author].State] = struct{}{}
	}

	if _, ok := uniqueStates["changes_requested"]; ok {
		return emojiNeedsChange
	}

	if _, ok := uniqueStates["approved"]; ok {
		approvalCount := 0
		for _, r := range reviews {
			if r.State == "approved" {
				approvalCount++
			}
		}
		if approvalCount >= numberOfApprovalsRequired {
			return emojiApproved
		}
	}

	if _, ok := uniqueStates["commented"]; ok {
		return emojiCommented
	}

	return ""
}

// Diff computes which emojis need to be added and removed given the desired
// new set and the currently existing set.
func Diff(newEmojis, existingEmojis map[string]struct{}) (toAdd, toRemove map[string]struct{}) {
	toAdd = make(map[string]struct{})
	toRemove = make(map[string]struct{})

	for e := range newEmojis {
		if _, exists := existingEmojis[e]; !exists {
			toAdd[e] = struct{}{}
		}
	}
	for e := range existingEmojis {
		if _, exists := newEmojis[e]; !exists {
			toRemove[e] = struct{}{}
		}
	}
	return toAdd, toRemove
}
