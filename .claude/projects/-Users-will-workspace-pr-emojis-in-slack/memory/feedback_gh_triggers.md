---
name: gh CLI doesn't trigger workflows
description: Closing/reopening PRs via gh CLI does not trigger GitHub Actions workflows
type: feedback
---

Closing and reopening PRs via `gh pr close` / `gh pr reopen` does not trigger GitHub Actions workflows. The user needs to do it manually through the GitHub UI.

**Why:** GitHub Actions doesn't fire on events created by the default GITHUB_TOKEN or gh CLI to prevent recursive workflow triggers.
**How to apply:** Don't offer to close/reopen PRs via gh CLI when the goal is to trigger a workflow. Let the user do it manually.
