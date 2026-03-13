output "project_names" {
  description = "map of managed Doppler project names"
  value       = { for k, v in doppler_project.projects : k => v.name }
}

output "group_slugs" {
  description = "map of group names to their Doppler slugs"
  value       = { for k, v in doppler_group.groups : k => v.slug }
}
