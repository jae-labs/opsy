locals {
  members = {
    "luiz1361"        = { role = "admin", full_name = "Luiz F. C. Martins" }
    "abubakar-abiona" = { role = "member", full_name = "Abubakar Abiona" }
  }

  teams = {
    "Maintainers" = {
      description = "Maintainers"
      privacy     = "closed"
      members     = ["luiz1361"]
      maintainers = ["luiz1361"]
      org_roles = {
        all_repo_admin    = 8136
        all_repo_maintain = 8135
        all_repo_read     = 8132
        all_repo_triage   = 8133
        all_repo_write    = 8134
        app_manager       = 33679
        ci_cd_admin       = 26237
        security_manager  = 138
      }
    }
    "Collaborators" = {
      description = "Collaborators"
      privacy     = "closed"
      members     = ["abubakar-abiona"]
      maintainers = ["luiz1361"]
    }
  }
}
