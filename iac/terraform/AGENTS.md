# Terraform IaC — jae-labs GitHub Org

Root modules: `github/`, `cloudflare/`, `doppler/`. Reusable modules: `modules/github/`, `modules/doppler/`. State in GCS bucket `gh-jae-labs-terraform`; each root module has its own state. Terraform >= 1.5. CI auto-applies on merge to main, path-filtered per module. Action SHAs are ratchet-pinned in workflows.

## Structure

- `github/` — org settings, members, teams, repos, branch protection, environments
- `cloudflare/` — DNS records
- `doppler/` — secrets management
- `modules/` — reusable modules
- `scripts/` — GCS backend bootstrap
- `docs/` — ci-cd.md, bootstrap.md, github-module.md

## Bot-Consumed Files

The Opsy bot reads and writes these files via the GitHub API:

| File | Bot Operation |
|---|---|
| `github/locals_repos.tf` | Add/delete/update repos |
| `github/locals_members.tf` | Extract team names for dropdowns |
| `github/locals_org.tf` | Read/update org settings |
| `cloudflare/locals_dns.tf` | Add/delete/update DNS records |

## Critical Constraints

- MUST NOT rename or restructure any bot-consumed locals file without updating path constants in `bot/slack/internal/slack/handler.go`
- MUST NOT change HCL key names or nesting in those files without updating the HCL editor at `bot/slack/internal/hcl/`
- These files are the contract between IaC and the bot; treat them as a public API
