locals {
  org = "jae-labs"

  org_settings = {
    name                                                     = "JAE Labs"
    billing_email                                            = "luiz@justanother.engineer"
    blog                                                     = "https://justanother.engineer"
    description                                              = "Just Another Engineer playing with code."
    location                                                 = "Ireland"
    members_can_create_repositories                          = false
    default_repository_permission                            = "read"
    web_commit_signoff_required                              = false
    dependabot_alerts_enabled_for_new_repositories           = true
    dependabot_security_updates_enabled_for_new_repositories = true
    dependency_graph_enabled_for_new_repositories            = true
  }
}
