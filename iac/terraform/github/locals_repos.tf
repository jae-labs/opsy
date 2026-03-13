locals {
  repos = {
    "opsy" = {
      description    = "A Slack Bot written in GoLang that provisions resources, manages access, and automates workflows across various platforms via Terraform."
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
        force_push_bypassers            = ["/luiz1361"]
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
    "dotfiles" = {
      description       = "dotfiles for macOS."
      visibility        = "public"
      has_issues        = true
      default_branch    = "main"
      topics            = ["dotfiles", "macos", "mise", "neovim", "tmux", "zsh"]
      team_access       = { "Maintainers" = "admin" }
      branch_protection = null
      security_and_analysis = {
        secret_scanning                 = { status = "enabled" }
        secret_scanning_push_protection = { status = "enabled" }
      }
    }
    "flashcards" = {
      description            = "Transform notes into flashcards with local AI. Study offline, stay private, learn smarter—for free!"
      visibility             = "public"
      has_issues             = true
      allow_auto_merge       = true
      allow_update_branch    = true
      delete_branch_on_merge = true
      default_branch         = "main"
      topics                 = ["ai", "cli", "flashcards", "flashcards-cli", "golang", "llama", "ollama", "sqlite"]
      team_access            = { "Maintainers" = "admin" }
      branch_protection      = null
    }
    "pages" = {
      description       = "Just Another Engineer website. Built with Docusaurus and deployed on GitHub Pages."
      visibility        = "public"
      has_issues        = true
      has_discussions   = true
      has_wiki          = true
      has_projects      = true
      homepage_url      = "http://justanother.engineer/"
      default_branch    = "main"
      topics            = ["cheatsheets", "docs", "documentation", "docusaurus", "markdown"]
      team_access       = { "Maintainers" = "admin" }
      branch_protection = null
      pages = {
        build_type = "workflow"
        cname      = "justanother.engineer"
      }
      security_and_analysis = {
        secret_scanning                 = { status = "enabled" }
        secret_scanning_push_protection = { status = "enabled" }
      }
    }
    "sandbox" = {
      description       = "Exploring ideas, testing concepts, and prototyping solutions."
      visibility        = "public"
      has_issues        = true
      default_branch    = "main"
      team_access       = { "Maintainers" = "admin" }
      branch_protection = null
    }
    "homebrew-formulae" = {
      description       = "A Homebrew tap that provides formulae for installing my projects."
      visibility        = "public"
      has_issues        = true
      default_branch    = "main"
      topics            = ["brew", "homebrew-formulae", "homebrew-tap"]
      team_access       = { "Maintainers" = "admin" }
      branch_protection = null
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
    "test" = {
      description      = "test"
      visibility       = "private"
      has_issues       = true
      default_branch   = "main"
      topics           = ["test"]
      team_access      = { "Collaborators" = "admin" }
      branch_protection = null
    }
  }
}
