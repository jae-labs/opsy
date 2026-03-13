output "team_ids" {
  description = "map of team names to their GitHub IDs"
  value       = { for k, v in github_team.teams : k => v.id }
}

output "repo_names" {
  description = "map of managed repository names"
  value       = { for k, v in github_repository.repos : k => v.name }
}

output "repo_node_ids" {
  description = "map of repository names to node IDs"
  value       = { for k, v in github_repository.repos : k => v.node_id }
}
