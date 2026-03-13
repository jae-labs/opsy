# Agent Instructions — Opsy Slack Bot

## Architecture overview

Opsy is a self-service GitOps platform: Slack modal -> Bot manipulates HCL -> GitHub PR -> CI/CD applies Terraform. The bot reads live Terraform files from GitHub to populate dropdowns and validate input, then writes modified HCL back via PR. This creates a tight coupling between `bot/slack/` and `iac/terraform/` — changes to either side must consider the other.

## Module and commands

- Go module root: `bot/slack/` (go.mod lives here, NOT at repo root)
- Build: `pushd bot/slack && go build ./...`
- Test: `pushd bot/slack && go test ./...`
- MUST run from `bot/slack/` directory — `cd` alone may not work in all shells, use `pushd`

## Package map

| Package | Role |
|---|---|
| `internal/config` | Env var loading and validation |
| `internal/conversation` | Thread-keyed state machine (`State` struct); one state per Slack thread keyed by thread timestamp |
| `internal/github` | GitHub API wrapper: `GetFileContent`, `CreateBranchFromMain`, `UpdateFile`, `CreatePR` |
| `internal/hcl` | HCL editors for terraform locals files; template-based rendering for new entries |
| `internal/slack` | Socket Mode event loop, interaction routing, Block Kit modal definitions, validation |

## Bot <-> Terraform coupling

The bot fetches Terraform files at runtime via GitHub API and parses them to populate Slack modals. This means:

### Path constants (`handler.go`)
```
pathGitHubRepos   = "iac/terraform/github/locals_repos.tf"
pathGitHubMembers = "iac/terraform/github/locals_members.tf"
pathGitHubOrg     = "iac/terraform/github/locals_org.tf"
pathCloudflareDNS = "iac/terraform/cloudflare/locals_dns.tf"
```

### What the bot reads from Terraform
| Terraform file | Bot reads | Used for |
|---|---|---|
| `locals_members.tf` | `teams` map keys | Team dropdown options in repo modals |
| `locals_repos.tf` | repo names, full repo configs | Duplicate detection, edit pre-population, delete targets |
| `locals_org.tf` | org settings | Org settings edit pre-population |
| `locals_dns.tf` | DNS record keys and configs | DNS record dropdowns, edit pre-population |

### What the bot writes to Terraform
| Action | Terraform file | HCL function |
|---|---|---|
| Add repo | `locals_repos.tf` | `hcleditor.AddRepo()` |
| Delete repo | `locals_repos.tf` | `hcleditor.RemoveRepo()` |
| Edit repo | `locals_repos.tf` | `hcleditor.UpdateRepo()` |
| Add DNS | `locals_dns.tf` | `hcleditor.AddDnsRecord()` |
| Delete DNS | `locals_dns.tf` | `hcleditor.RemoveDnsRecord()` |
| Edit DNS | `locals_dns.tf` | `hcleditor.UpdateDnsRecord()` |
| Edit org | `locals_org.tf` | `hcleditor.UpdateOrgSettings()` |

### Impact
- Renaming/restructuring Terraform files breaks the bot at runtime
- Adding a new field to a Terraform resource requires changes in: HCL editor template, Block Kit modal, handler parsing, conversation state struct, confirmation blocks
- Removing a team from `locals_members.tf` affects team dropdown options in the bot

## Data fetching and fallbacks

| Function | Source file | Parses with | Fallback on error |
|---|---|---|---|
| `fetchTeamNames()` | `locals_members.tf` | `hcleditor.ExtractTeamNames()` | `["Maintainers"]` (hardcoded) |
| `fetchRepoNames()` | `locals_repos.tf` | `hcleditor.ExistingRepoNames()` | `nil` (no fallback) |

If the bot logs show only "Maintainers" in team dropdowns, check whether `fetchTeamNames()` is hitting an API error and falling back.

## Parallel flows: create vs edit repos

Create and edit share the same `RepoConfig` struct and similar 3-step modals, but differ in critical ways. When modifying one flow, check if the other needs the same change.

| Aspect | Create flow | Edit flow |
|---|---|---|
| Callbacks | `CallbackRepoStep1/2/3` | `CallbackSelectRepo`, `CallbackSettingsStep1/2/3` |
| Modal builders | `RepoStep1Modal`, `RepoStep2Modal`, `RepoStep3Modal` | `SelectRepoModal`, `SettingsStep1Modal`, `SettingsStep2Modal`, `SettingsStep3Modal` |
| Step 1 | Collects name + desc + visibility + justification | Collects desc + visibility + justification (no name) |
| Step 2 team access | Multi-select, all default to `"admin"` | Multi-select, preserves existing permission levels |
| Step 2 pre-population | Empty | Pre-populated from existing config |
| Step 3 pre-population | Defaults (protection off, reviews=1) | Pre-populated from existing config |
| Confirmation | `ConfirmationBlocks()` — shows summary | `SettingsConfirmationBlocks()` — shows old vs new diff |
| PR creation | `createPR()` -> `hcleditor.AddRepo()` | `createSettingsPR()` -> `hcleditor.UpdateRepo()` |
| Repo validation | `checkRepoAlreadyExists()` | `checkRepoStillExists()` |

### Common mistake: create/edit parity drift

The create and edit flows are implemented as separate code paths in both `blocks.go` and `handler.go`. When changing behavior (e.g., switching single-select to multi-select for team access), both flows must be updated:
- Modal builder function in `blocks.go`
- Submission parser in `handler.go` (`.SelectedOption.Value` vs `.SelectedOptions`)
- Confirmation display in `blocks.go`

## Block Kit constants (`blocks.go`)

All Block/Elem IDs are paired constants at the top of `blocks.go`. `Block*` is the container ID, `Elem*` is the form control ID. Both are needed when reading submission values: `values[BlockFoo][ElemFoo]`.

## HCL editing (`internal/hcl/`)

- **Read**: Uses `hcl/v2` AST parsing (safe, structured)
- **Write**: Uses Go templates + string insertion (preserves formatting)
- HCL template for repos is in `editor.go` (`renderRepoEntry`); uses dynamic padding for alignment
- `team_access` is rendered sorted by team name for deterministic output
- All editors double-validate: parse input HCL, modify, parse output HCL
- Test fixtures in `internal/hcl/testdata/` mirror production Terraform structure

## Conversation state (`internal/conversation/`)

- `State` struct holds: Phase, Category, ResourceType, ActionType, RepoConfig, DnsConfig, OrgConfig
- `RepoConfig.TeamAccess` is `map[string]string` (key=team name, value=permission level)
- Valid permission levels: `admin`, `maintain`, `push`, `triage`, `pull`
- `Store` is concurrency-safe in-memory map keyed by thread timestamp
- State is deleted after PR creation or cancel

## Key constraints

**HCL field names**: changing a field name requires updating the HCL editor template, Block Kit modal, handler parser, and confirmation blocks. Missing any will silently produce malformed Terraform or broken UI.

**Test data**: `internal/hcl/testdata/` mirrors production Terraform files. Update fixtures when adding new HCL editor features.

**Confirmation blocks**: create (`ConfirmationBlocks`) takes ~18 positional parameters. Edit (`SettingsConfirmationBlocks`) takes old/new config and diffs them. Adding a new field to `RepoConfig` requires updating both.
