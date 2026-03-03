locals {
  # flatten project/environment pairs for for_each
  project_environments = {
    for entry in flatten([
      for project_name, project in var.projects : [
        for env_slug, env in project.environments : {
          key          = "${project_name}:${env_slug}"
          project_name = project_name
          env_slug     = env_slug
          env_name     = env.name
          env          = env
        }
      ]
    ]) : entry.key => entry
  }

  # keyed by "project:group" for for_each; duplicates fail fast
  project_access_map = {
    for entry in var.project_access :
    "${entry.project}:${entry.group}" => entry
  }
}

resource "doppler_project" "projects" {
  for_each = var.projects

  name        = each.key
  description = each.value.description
}

resource "doppler_environment" "envs" {
  for_each = local.project_environments

  project          = doppler_project.projects[each.value.project_name].name
  slug             = each.value.env_slug
  name             = each.value.env_name
  personal_configs = each.value.env.personal_configs
}

resource "doppler_group" "groups" {
  for_each = var.groups

  name = each.key
}

resource "doppler_project_member_group" "access" {
  for_each = local.project_access_map

  project      = doppler_project.projects[each.value.project].name
  group_slug   = doppler_group.groups[each.value.group].slug
  role         = each.value.role
  environments = each.value.environments
}
