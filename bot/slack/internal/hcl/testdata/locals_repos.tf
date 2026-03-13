locals {
  repos = {
    "terraform" = {
      description    = "Terraform IaC for managing this GitHub organization and a few other tools"
      visibility     = "public"
      has_issues     = true
      has_wiki       = false
      default_branch = "main"
      topics         = ["iac", "terraform"]
      team_access    = { "Maintainers" = "admin" }
      branch_protection = {
        required_reviews                = 1
        dismiss_stale_reviews           = true
        require_linear_history          = true
        require_conversation_resolution = true
      }
      environments = {
        "production" = {
          deployment_branch_policy = {
            protected_branches     = true
            custom_branch_policies = false
          }
        }
      }
    }
    ".github" = {
      description       = "Organization-wide shared templates"
      visibility        = "public"
      has_issues        = true
      has_wiki          = false
      default_branch    = "main"
      topics            = ["organization-templates"]
      team_access       = { "Maintainers" = "admin" }
      branch_protection = null
    }
    "catv" = {
      description            = "Transform notes into flashcards with local AI."
      visibility             = "public"
      has_issues             = true
      allow_auto_merge       = true
      allow_update_branch    = true
      delete_branch_on_merge = true
      default_branch         = "main"
      topics                 = ["ai", "cli", "flashcards", "golang", "ollama"]
      team_access            = { "Maintainers" = "admin" }
      branch_protection      = null
    }
    "scripts" = {
      description       = "Collection of utility scripts."
      visibility        = "private"
      has_issues        = true
      has_wiki          = false
      default_branch    = "main"
      topics            = ["automation", "bash", "python", "scripting", "shell"]
      team_access       = { "Maintainers" = "admin" }
      branch_protection = null
    }
    "community-hub" = {
      description            = "Community discussion and project showcase."
      visibility             = "public"
      has_issues             = true
      has_discussions         = true
      has_projects            = true
      homepage_url            = "https://community.justanother.engineer"
      allow_auto_merge       = true
      delete_branch_on_merge = true
      default_branch         = "main"
      topics                 = ["community", "showcase"]
      team_access            = { "Maintainers" = "admin" }
      branch_protection      = null
    }
  }
}
