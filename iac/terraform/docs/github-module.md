# GitHub Module

Manages the `jae-labs` GitHub organization.

## Resources managed

| Resource | Key | Description |
|---|---|---|
| `github_organization_settings` | `org` | Org-level settings |
| `github_membership` | `members[username]` | Org members |
| `github_team` | `teams[name]` | Teams |
| `github_team_members` | `teams[name]` | Team membership |
| `github_repository` | `repos[name]` | Repositories |
| `github_team_repository` | `repos[repo:team]` | Team-repo access |
| `github_repository_environment` | `envs[repo:env]` | Deployment environments |
| `github_branch_protection` | `repos[name]` | Branch protection rules |

## Configuration

All metadata lives in split locals files under `github/`:

| File | Content |
|---|---|
| `locals_org.tf` | `org`, `org_settings` |
| `locals_members.tf` | `members`, `teams` |
| `locals_repos.tf` | `repos` |

The module accepts structured maps for members, teams, repos, and org settings.

### Adding a repository

```hcl
"my-repo" = {
  description    = "My new repo"
  visibility     = "public"
  has_issues     = true
  default_branch = "main"
  team_access    = { "Maintainers" = "admin" }
  branch_protection = null
}
```

### Adding branch protection

```hcl
branch_protection = {
  required_reviews                = 1
  dismiss_stale_reviews           = true
  require_linear_history          = true
  require_conversation_resolution = true
}
```

### Adding an environment

```hcl
environments = {
  "production" = {
    deployment_branch_policy = {
      protected_branches     = true
      custom_branch_policies = false
    }
  }
}
```

## Auth

The GitHub provider reads `GITHUB_TOKEN` automatically. No variable needed.

### Fine-grained PAT permissions

| Scope | Permission | Access |
|---|---|---|
| Organization | Administration | Read and write |
| Organization | Members | Read and write |
| Repository | Administration | Read and write |
| Repository | Environments | Read and write |
| Repository | Metadata | Read-only |
