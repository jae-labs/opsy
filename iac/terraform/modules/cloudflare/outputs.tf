output "zone_ids" {
  description = "map of zone names to their Cloudflare zone IDs"
  value       = { for k, v in cloudflare_zone.zones : k => v.id }
}

output "dns_record_ids" {
  description = "map of composite keys to DNS record IDs"
  value       = { for k, v in cloudflare_dns_record.records : k => v.id }
}

output "member_ids" {
  description = "map of member emails to their Cloudflare member IDs"
  value       = { for k, v in cloudflare_account_member.members : k => v.id }
}
