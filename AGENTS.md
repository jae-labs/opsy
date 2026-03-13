# AGENTS.md — jae-labs/opsy

## Overview

Monorepo for two systems with a hard file-path contract between them:

1. **Terraform IaC** (`iac/terraform/`) — manages GitHub org, Cloudflare DNS, Doppler secrets.
2. **Opsy Slack Bot** (`bot/slack/`) — Go bot that opens PRs mutating terraform locals files.

## Cross-system contract

The bot reads and writes terraform locals files directly via the GitHub API. Any rename or restructure of those files breaks the bot unless path constants in `bot/slack/internal/slack/handler.go` are updated in the same change.

| Bot operation        | Terraform file                                    |
|----------------------|---------------------------------------------------|
| Add/remove repo      | `iac/terraform/github/locals_repos.tf`            |
| Add/remove member    | `iac/terraform/github/locals_members.tf`          |
| Update org settings  | `iac/terraform/github/locals_org.tf`              |
| Add/remove DNS record| `iac/terraform/cloudflare/locals_dns.tf`          |

## Component guidelines

- `iac/terraform/AGENTS.md` — terraform module conventions, variable naming, state backend.
- `bot/slack/AGENTS.md` — bot architecture, HCL parsing, PR creation flow, test patterns.

## CI

Workflows live in `.github/workflows/`. Triggering is path-based:

- Changes under `iac/terraform/` trigger terraform plan/apply CI.
- Changes under `bot/slack/` trigger Go build, lint, and test CI.

## Agent rules

- MUST update the four path constants in `bot/slack/internal/slack/handler.go` whenever a terraform locals file is renamed or moved.
- MUST run `go test ./...` from `bot/slack/` after any bot changes.
- MUST NOT modify terraform files and bot files in the same PR — they have different review concerns and CI pipelines.
- Test data in `bot/slack/internal/hcl/testdata/` mirrors the structure of the terraform locals files; keep it in sync when terraform file structure changes.
