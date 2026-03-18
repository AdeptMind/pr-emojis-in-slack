package run

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AdeptMind/pr-emojis-in-slack/internal/config"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/emoji"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/github"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/slack"
)

// Run executes the main orchestration logic: reads a GitHub event, determines
// the appropriate emoji reactions, and applies them to the corresponding Slack
// message.
func Run(cfg *config.Config, gh *github.Client, sl *slack.Client) error {
	event, err := gh.ReadEvent()
	if err != nil {
		return fmt.Errorf("reading github event: %w", err)
	}

	// Determine if this is a pull_request or issue_comment event.
	var prNumber int
	var prURL string

	if _, ok := event["pull_request"]; ok {
		// pull_request or pull_request_review event.
		isFork, err := extractBool(event, "pull_request", "head", "repo", "fork")
		if err != nil {
			return fmt.Errorf("extracting fork status: %w", err)
		}
		if isFork {
			fmt.Println("Fork PRs are not supported.")
			return nil
		}
		prNumber, err = extractInt(event, "pull_request", "number")
		if err != nil {
			return fmt.Errorf("extracting PR number: %w", err)
		}
		prURL, err = extractString(event, "pull_request", "html_url")
		if err != nil {
			return fmt.Errorf("extracting PR URL: %w", err)
		}
	} else if _, ok := event["issue"]; ok {
		// issue_comment event — skip if not a PR.
		if _, err := navigate(event, []string{"issue", "pull_request"}); err != nil {
			fmt.Println("Not a pull request comment, skipping.")
			return nil
		}
		prNumber, err = extractInt(event, "issue", "number")
		if err != nil {
			return fmt.Errorf("extracting issue number: %w", err)
		}
		prURL, err = extractString(event, "issue", "pull_request", "html_url")
		if err != nil {
			return fmt.Errorf("extracting PR URL from issue: %w", err)
		}
	} else {
		return fmt.Errorf("unrecognized event payload")
	}
	fmt.Printf("Event PR: %s\n", prURL)

	// Fetch PR details, reviews, and Slack message timestamp in parallel.
	type prResult struct {
		pr  github.PullRequest
		err error
	}
	type reviewsResult struct {
		reviews []github.Review
		err     error
	}
	type tsResult struct {
		timestamp string
		err       error
	}

	prCh := make(chan prResult, 1)
	reviewsCh := make(chan reviewsResult, 1)
	tsCh := make(chan tsResult, 1)

	go func() {
		pr, err := gh.GetPR(prNumber)
		prCh <- prResult{pr, err}
	}()
	go func() {
		reviews, err := gh.GetPRReviews(prNumber)
		reviewsCh <- reviewsResult{reviews, err}
	}()
	go func() {
		ts, err := sl.FindTimestampOfReviewRequestedMessage(prURL, cfg.SlackChannelID)
		tsCh <- tsResult{ts, err}
	}()

	prRes := <-prCh
	if prRes.err != nil {
		return fmt.Errorf("getting PR: %w", prRes.err)
	}
	pr := prRes.pr

	reviewsRes := <-reviewsCh
	if reviewsRes.err != nil {
		return fmt.Errorf("getting PR reviews: %w", reviewsRes.err)
	}

	reviewEmoji := emoji.GetForReviews(
		reviewsRes.reviews,
		cfg.EmojiCommented,
		cfg.EmojiNeedsChange,
		cfg.EmojiApproved,
		cfg.NumberOfApprovalsRequired,
	)

	tsRes := <-tsCh
	if tsRes.err != nil {
		return fmt.Errorf("finding slack message: %w", tsRes.err)
	}
	timestamp := tsRes.timestamp
	fmt.Printf("Slack message timestamp: %s\n", timestamp)

	if timestamp == "" {
		fmt.Printf("No message found requesting review for PR: %s\n", prURL)
		return nil
	}

	existingEmojis, err := sl.GetEmojisForUser(timestamp, cfg.SlackChannelID, cfg.BotUserID)
	if err != nil {
		return fmt.Errorf("getting existing emojis: %w", err)
	}
	fmt.Printf("Existing emojis: %s\n", joinSet(existingEmojis))

	// Build the desired emoji set.
	newEmojis := map[string]struct{}{
		cfg.EmojiMonitoring: {},
	}
	if reviewEmoji != "" {
		newEmojis[cfg.EmojiReviewStarted] = struct{}{}
		newEmojis[reviewEmoji] = struct{}{}
	}

	fmt.Printf("Is merged: %v\n", pr.Merged)
	fmt.Printf("Mergeable state: %s\n", pr.MergeableState)

	if pr.Merged {
		newEmojis[cfg.EmojiMerged] = struct{}{}
	} else if pr.State == "closed" {
		newEmojis[cfg.EmojiClosed] = struct{}{}
	}

	// Sort desired emojis by review step order.
	sortedNew := setToSlice(newEmojis)
	sort.Slice(sortedNew, func(i, j int) bool {
		return cfg.EmojisByReviewStep(sortedNew[i]) < cfg.EmojisByReviewStep(sortedNew[j])
	})

	fmt.Printf("Desired emojis (ordered): %s\n", strings.Join(sortedNew, ", "))

	// Remove all existing emojis, then re-add in sorted order so Slack
	// displays them in the correct review-lifecycle sequence.
	for e := range existingEmojis {
		if err := sl.RemoveReaction(timestamp, e, cfg.SlackChannelID); err != nil {
			return fmt.Errorf("removing reaction %s: %w", e, err)
		}
	}

	for _, e := range sortedNew {
		if err := sl.AddReaction(timestamp, e, cfg.SlackChannelID); err != nil {
			return fmt.Errorf("adding reaction %s: %w", e, err)
		}
	}

	return nil
}

// extractBool navigates a nested map[string]interface{} by keys and returns
// the final value as a bool.
func extractBool(m map[string]interface{}, keys ...string) (bool, error) {
	val, err := navigate(m, keys)
	if err != nil {
		return false, err
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("expected bool at %s, got %T", strings.Join(keys, "."), val)
	}
	return b, nil
}

// extractInt navigates a nested map and returns the value as an int.
// JSON numbers are decoded as float64 by encoding/json.
func extractInt(m map[string]interface{}, keys ...string) (int, error) {
	val, err := navigate(m, keys)
	if err != nil {
		return 0, err
	}
	f, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("expected number at %s, got %T", strings.Join(keys, "."), val)
	}
	return int(f), nil
}

// extractString navigates a nested map and returns the value as a string.
func extractString(m map[string]interface{}, keys ...string) (string, error) {
	val, err := navigate(m, keys)
	if err != nil {
		return "", err
	}
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("expected string at %s, got %T", strings.Join(keys, "."), val)
	}
	return s, nil
}

// navigate walks a nested map by the given keys and returns the final value.
func navigate(m map[string]interface{}, keys []string) (interface{}, error) {
	var current interface{} = m
	for _, k := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map at key %q, got %T", k, current)
		}
		val, exists := cm[k]
		if !exists {
			return nil, fmt.Errorf("key %q not found", k)
		}
		current = val
	}
	return current, nil
}

// setToSlice converts a set (map[string]struct{}) to a slice.
func setToSlice(s map[string]struct{}) []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	return result
}

// joinSet converts a set to a comma-separated string.
func joinSet(s map[string]struct{}) string {
	return strings.Join(setToSlice(s), ", ")
}
