locals {
  # merge maintainers into members per team (maintainer role takes precedence)
  team_members_by_role = {
    for team_name, team in var.teams : team_name => merge(
      { for u in team.members : u => "member" },
      { for u in team.maintainers : u => "maintainer" }
    )
  }

  # flatten team org_roles to a single map keyed "team:role"
  team_org_roles = {
    for entry in flatten([
      for team_name, team in var.teams : [
        for role_name, role_id in team.org_roles : {
          key       = "${team_name}:${role_name}"
          team_name = team_name
          role_id   = role_id
        }
      ]
    ]) : entry.key => entry
  }

  # flatten repo team_access to a single map keyed "repo:team"
  repo_team_access = {
    for entry in flatten([
      for repo_name, repo in var.repos : [
        for team_name, permission in repo.team_access : {
          key        = "${repo_name}:${team_name}"
          repo_name  = repo_name
          team_name  = team_name
          permission = permission
        }
      ]
    ]) : entry.key => entry
  }

  # flatten repo environments to a single map keyed "repo:env"
  repo_environments = {
    for entry in flatten([
      for repo_name, repo in var.repos : [
        for env_name, env in repo.environments : {
          key       = "${repo_name}:${env_name}"
          repo_name = repo_name
          env_name  = env_name
          env       = env
        }
      ]
    ]) : entry.key => entry
  }
}

resource "github_organization_settings" "org" {
  name                                                         = var.org_settings.name
  billing_email                                                = var.org_settings.billing_email
  blog                                                         = var.org_settings.blog
  email                                                        = var.org_settings.email
  description                                                  = var.org_settings.description
  location                                                     = var.org_settings.location
  members_can_create_repositories                              = var.org_settings.members_can_create_repositories
  members_can_create_public_repositories                       = var.org_settings.members_can_create_public_repositories
  members_can_create_private_repositories                      = var.org_settings.members_can_create_private_repositories
  members_can_create_private_pages                             = var.org_settings.members_can_create_private_pages
  default_repository_permission                                = var.org_settings.default_repository_permission
  web_commit_signoff_required                                  = var.org_settings.web_commit_signoff_required
  dependabot_alerts_enabled_for_new_repositories               = var.org_settings.dependabot_alerts_enabled_for_new_repositories
  dependabot_security_updates_enabled_for_new_repositories     = var.org_settings.dependabot_security_updates_enabled_for_new_repositories
  dependency_graph_enabled_for_new_repositories                = var.org_settings.dependency_graph_enabled_for_new_repositories
  advanced_security_enabled_for_new_repositories               = var.org_settings.advanced_security_enabled_for_new_repositories
  secret_scanning_enabled_for_new_repositories                 = var.org_settings.secret_scanning_enabled_for_new_repositories
  secret_scanning_push_protection_enabled_for_new_repositories = var.org_settings.secret_scanning_push_protection_enabled_for_new_repositories
}

resource "github_membership" "members" {
  for_each = var.members

  username = each.key
  role     = each.value.role
}

resource "github_team" "teams" {
  for_each = var.teams

  name        = each.key
  description = each.value.description
  privacy     = each.value.privacy
}

resource "github_team_members" "teams" {
  for_each = var.teams

  team_id = github_team.teams[each.key].id

  dynamic "members" {
    for_each = local.team_members_by_role[each.key]
    content {
      username = members.key
      role     = members.value
    }
  }

  depends_on = [github_membership.members]
}

resource "github_organization_role_team" "team_roles" {
  for_each = local.team_org_roles

  team_slug = github_team.teams[each.value.team_name].slug
  role_id   = each.value.role_id
}

resource "github_repository" "repos" {
  for_each = var.repos

  name                        = each.key
  description                 = each.value.description
  visibility                  = each.value.visibility
  has_issues                  = each.value.has_issues
  has_wiki                    = each.value.has_wiki
  has_projects                = each.value.has_projects
  has_discussions             = each.value.has_discussions
  homepage_url                = each.value.homepage_url
  allow_auto_merge            = each.value.allow_auto_merge
  allow_merge_commit          = each.value.allow_merge_commit
  allow_rebase_merge          = each.value.allow_rebase_merge
  allow_squash_merge          = each.value.allow_squash_merge
  allow_update_branch         = each.value.allow_update_branch
  archived                    = each.value.archived
  delete_branch_on_merge      = each.value.delete_branch_on_merge
  is_template                 = each.value.is_template
  merge_commit_title          = each.value.merge_commit_title
  merge_commit_message        = each.value.merge_commit_message
  squash_merge_commit_title   = each.value.squash_merge_commit_title
  squash_merge_commit_message = each.value.squash_merge_commit_message
  vulnerability_alerts        = each.value.vulnerability_alerts
  web_commit_signoff_required = each.value.web_commit_signoff_required
  topics                      = each.value.topics

  dynamic "pages" {
    for_each = each.value.pages != null ? [each.value.pages] : []
    content {
      build_type = pages.value.build_type
      cname      = pages.value.cname

      dynamic "source" {
        for_each = pages.value.source != null ? [pages.value.source] : []
        content {
          branch = source.value.branch
          path   = source.value.path
        }
      }
    }
  }

  dynamic "security_and_analysis" {
    for_each = each.value.security_and_analysis != null ? [each.value.security_and_analysis] : []
    content {
      dynamic "secret_scanning" {
        for_each = security_and_analysis.value.secret_scanning != null ? [security_and_analysis.value.secret_scanning] : []
        content {
          status = secret_scanning.value.status
        }
      }
      dynamic "secret_scanning_push_protection" {
        for_each = security_and_analysis.value.secret_scanning_push_protection != null ? [security_and_analysis.value.secret_scanning_push_protection] : []
        content {
          status = secret_scanning_push_protection.value.status
        }
      }
    }
  }

  lifecycle {
    prevent_destroy = false #TODO: Review
  }
}

resource "github_team_repository" "repos" {
  for_each = local.repo_team_access

  team_id    = github_team.teams[each.value.team_name].id
  repository = github_repository.repos[each.value.repo_name].name
  permission = each.value.permission
}

resource "github_repository_environment" "envs" {
  for_each = local.repo_environments

  repository          = github_repository.repos[each.value.repo_name].name
  environment         = each.value.env_name
  wait_timer          = each.value.env.wait_timer
  prevent_self_review = each.value.env.prevent_self_review
  can_admins_bypass   = each.value.env.can_admins_bypass

  dynamic "reviewers" {
    for_each = each.value.env.reviewers != null ? [each.value.env.reviewers] : []
    content {
      users = reviewers.value.users
      teams = reviewers.value.teams
    }
  }

  dynamic "deployment_branch_policy" {
    for_each = each.value.env.deployment_branch_policy != null ? [each.value.env.deployment_branch_policy] : []
    content {
      protected_branches     = deployment_branch_policy.value.protected_branches
      custom_branch_policies = deployment_branch_policy.value.custom_branch_policies
    }
  }
}

resource "github_branch_protection" "repos" {
  for_each = {
    for k, v in var.repos : k => v if v.branch_protection != null
  }

  repository_id = github_repository.repos[each.key].node_id
  pattern       = each.value.default_branch

  required_linear_history         = each.value.branch_protection.require_linear_history
  require_conversation_resolution = each.value.branch_protection.require_conversation_resolution
  force_push_bypassers            = each.value.branch_protection.force_push_bypassers

  required_pull_request_reviews {
    required_approving_review_count = each.value.branch_protection.required_reviews
    dismiss_stale_reviews           = each.value.branch_protection.dismiss_stale_reviews
  }
}
