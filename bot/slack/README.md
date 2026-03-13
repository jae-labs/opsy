# Opsy Slack Bot

A Go Slack bot (Socket Mode) providing self-service infrastructure workflows via Slack modals.

## What it does

- Create, update, and delete GitHub repositories (HCL manipulation of terraform locals files)
- Add, update, and delete Cloudflare DNS records
- Update GitHub org settings and repo settings (visibility, features, branch protection, team access)

Each workflow creates a branch, updates the relevant terraform file, and opens a PR for review.

## Architecture

```
Socket Mode -> event/interaction handler -> thread-keyed state machine -> Block Kit modals
  -> HCL text manipulation -> GitHub API (branch + file update + PR)
```

Terraform files live in `iac/terraform/` within this monorepo. The bot targets the `opsy` repo by default.

## Packages

| Package | Description |
|---|---|
| `internal/config` | Loads and validates environment config |
| `internal/conversation` | Thread-keyed state machine for multi-step workflows |
| `internal/github` | GitHub API client (branch, file, PR operations) |
| `internal/hcl` | HCL text editors for reading and writing terraform locals files |
| `internal/slack` | Socket Mode handler, Block Kit modals, interaction routing |

## Environment variables

| Variable | Description |
|---|---|
| `SLACK_BOT_TOKEN` | Bot OAuth token (`xoxb-...`) |
| `SLACK_APP_TOKEN` | App-level token for Socket Mode (`xapp-...`) |
| `GITHUB_TOKEN` | Personal access token or app token |
| `GITHUB_ORG` | GitHub organisation name |
| `GITHUB_REPO` | Terraform repo (default: `opsy`) |
| `GITHUB_REPO_BASE_BRANCH` | Base branch for PRs |
| `LOG_LEVEL` | Log level (default: `info`) |

Copy `.env.example` to `.env` and populate before running.

## Build and run

```sh
go build ./cmd/opsy/
./opsy
```

## Test

```sh
go test ./...
```
