# pr-emojis-in-slack

A GitHub Action that adds emoji reactions to Slack messages when PRs are reviewed, approved, merged, or closed.

## Setup

1. Create a Slack app Oauth Bot Token scopes: `reactions:read`, `reactions:write`, `channels:history`, `groups:history`
2. Install the app to your workspace and note the **Bot User ID** and **OAuth Token**
3. Add the OAuth token as a repository secret named `SLACK_TOKEN`
4. Create `.github/workflows/pr-emojis.yml`:

```yaml
name: PR Emojis in Slack

on:
  pull_request_review:
    types: [submitted]
  pull_request:
    types: [closed]

permissions: {}

jobs:
  pr-emojis:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: AdeptMind/pr-emojis-in-slack@main
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SLACK_CHANNEL_ID: "<your-channel-id>"
          SLACK_API_TOKEN: ${{ secrets.SLACK_TOKEN }}
          BOT_USER_ID: "<your-bot-user-id>"
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITHUB_TOKEN` | Yes | | GitHub token for API access |
| `SLACK_API_TOKEN` | Yes | | Slack bot OAuth token |
| `SLACK_CHANNEL_ID` | Yes | | Slack channel to monitor |
| `BOT_USER_ID` | Yes | | Slack bot user ID |
| `EMOJI_REVIEW_STARTED` | No | `review_started` | Emoji for review started |
| `EMOJI_APPROVED` | No | `approved` | Emoji for approval |
| `EMOJI_CHANGES_REQUESTED` | No | `changes_requested` | Emoji for changes requested |
| `EMOJI_COMMENTED` | No | `comment` | Emoji for comments |
| `EMOJI_MERGED` | No | `merged` | Emoji for merged PR |
| `EMOJI_CLOSED` | No | `closed` | Emoji for closed PR |
| `NUMBER_OF_APPROVALS_REQUIRED` | No | `1` | Approvals needed for approved emoji |
