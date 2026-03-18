# pr-emojis-in-slack

A GitHub Action that adds emoji reactions to Slack messages when PRs are reviewed, approved, merged, or closed.

## Setup

1. Create a Slack app Oauth Bot Token scopes: `reactions:read`, `reactions:write`, `channels:history`, `groups:history`
2. Install the app to your workspace and note the **Bot User ID** and **OAuth Token**
3. Add the Bot User OAuth Token (`xoxb-...`) as a repository secret named `SLACK_BOT_TOKEN`
4. Add `SLACK_CHANNEL_ID` and `SLACK_BOT_USER_ID` as repository variables (Settings > Secrets and variables > Actions > Variables)
5. Create `.github/workflows/pr-emojis.yml`:

```yaml
name: PR Emojis in Slack

on:
  pull_request_review:
    types: [submitted]
  pull_request:
    types: [opened, synchronize, closed]
  issue_comment:
    types: [created]

jobs:
  pr-emojis:
    runs-on: ubuntu-latest
    steps:
      - name: Run pr-emojis-in-slack
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SLACK_CHANNEL_ID: ${{ vars.SLACK_CHANNEL_ID }}
          SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
          SLACK_BOT_USER_ID: ${{ vars.SLACK_BOT_USER_ID }}
        run: |
          curl -fsSL https://github.com/AdeptMind/pr-emojis-in-slack/releases/download/v1.0.0/pr-emojis-in-slack -o pr-emojis-in-slack
          chmod +x pr-emojis-in-slack
          ./pr-emojis-in-slack
```

> **Pinning a version**: Replace `v1.0.0` in the URL above with the desired
> [release tag](https://github.com/AdeptMind/pr-emojis-in-slack/releases).
> Using `latest` is also supported but may introduce breaking changes.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITHUB_TOKEN` | Yes | | GitHub token for API access — provided automatically by GitHub Actions |
| `SLACK_BOT_TOKEN` | Yes | | Slack Bot User OAuth Token (`xoxb-...`) |
| `SLACK_CHANNEL_ID` | Yes | | Slack channel to monitor |
| `SLACK_BOT_USER_ID` | Yes | | Slack bot user ID |
| `EMOJI_MONITORING` | No | `sparkles` | Emoji added when PR is being monitored |
| `EMOJI_REVIEW_STARTED` | No | `eyes` | Emoji for review started |
| `EMOJI_APPROVED` | No | `white_check_mark` | Emoji for approval |
| `EMOJI_CHANGES_REQUESTED` | No | `warning` | Emoji for changes requested |
| `EMOJI_COMMENTED` | No | `speech_balloon` | Emoji for comments |
| `EMOJI_MERGED` | No | `rocket` | Emoji for merged PR |
| `EMOJI_CLOSED` | No | `no_entry_sign` | Emoji for closed PR |
| `NUMBER_OF_APPROVALS_REQUIRED` | No | `1` | Approvals needed for approved emoji |
