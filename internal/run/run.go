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

	// Check if PR is from a fork.
	isFork, err := extractBool(event, "pull_request", "head", "repo", "fork")
	if err != nil {
		return fmt.Errorf("extracting fork status: %w", err)
	}
	if isFork {
		fmt.Println("Fork PRs are not supported.")
		return nil
	}

	// Get PR number and details.
	prNumber, err := extractInt(event, "pull_request", "number")
	if err != nil {
		return fmt.Errorf("extracting PR number: %w", err)
	}

	pr, err := gh.GetPR(prNumber)
	if err != nil {
		return fmt.Errorf("getting PR: %w", err)
	}

	reviews, err := gh.GetPRReviews(prNumber)
	if err != nil {
		return fmt.Errorf("getting PR reviews: %w", err)
	}

	reviewEmoji := emoji.GetForReviews(
		reviews,
		cfg.EmojiCommented,
		cfg.EmojiNeedsChange,
		cfg.EmojiApproved,
		cfg.NumberOfApprovalsRequired,
	)

	prURL, err := extractString(event, "pull_request", "html_url")
	if err != nil {
		return fmt.Errorf("extracting PR URL: %w", err)
	}
	fmt.Printf("Event PR: %s\n", prURL)

	timestamp, err := sl.FindTimestampOfReviewRequestedMessage(prURL, cfg.SlackChannelID)
	if err != nil {
		return fmt.Errorf("finding slack message: %w", err)
	}
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
		cfg.EmojiReviewStarted: {},
	}
	if reviewEmoji != "" {
		newEmojis[reviewEmoji] = struct{}{}
	}

	fmt.Printf("Is merged: %v\n", pr.Merged)
	fmt.Printf("Mergeable state: %s\n", pr.MergeableState)

	if pr.Merged {
		newEmojis[cfg.EmojiMerged] = struct{}{}
	} else if pr.State == "closed" {
		newEmojis[cfg.EmojiClosed] = struct{}{}
	}

	// Compute diff.
	toAdd, toRemove := emoji.Diff(newEmojis, existingEmojis)

	// Sort emojis to add by review step order.
	sortedToAdd := setToSlice(toAdd)
	sort.Slice(sortedToAdd, func(i, j int) bool {
		return cfg.EmojisByReviewStep(sortedToAdd[i]) < cfg.EmojisByReviewStep(sortedToAdd[j])
	})

	fmt.Printf("Emojis to add (ordered) : %s\n", strings.Join(sortedToAdd, ", "))
	fmt.Printf("Emojis to remove        : %s\n", joinSet(toRemove))

	for _, e := range sortedToAdd {
		if err := sl.AddReaction(timestamp, e, cfg.SlackChannelID); err != nil {
			return fmt.Errorf("adding reaction %s: %w", e, err)
		}
	}

	for e := range toRemove {
		if err := sl.RemoveReaction(timestamp, e, cfg.SlackChannelID); err != nil {
			return fmt.Errorf("removing reaction %s: %w", e, err)
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
