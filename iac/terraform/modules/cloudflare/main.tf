locals {
  # flatten nested dns_records map to "zone:key" composite keys
  zone_dns_records = {
    for entry in flatten([
      for zone, records in var.dns_records : [
        for key, r in records : {
          composite_key = "${zone}:${key}"
          zone          = zone
          type          = r.type
          name          = r.name
          content       = r.content
          ttl           = r.ttl
          proxied       = r.proxied
          comment       = r.comment
          priority      = r.priority
        }
      ]
    ]) : entry.composite_key => entry
  }
}

resource "cloudflare_zone" "zones" {
  for_each = var.zones

  account = {
    id = var.account_id
  }
  name = each.key
  type = each.value.type

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloudflare_dns_record" "records" {
  for_each = local.zone_dns_records

  zone_id  = cloudflare_zone.zones[each.value.zone].id
  type     = each.value.type
  name     = each.value.name
  content  = each.value.content
  ttl      = each.value.ttl
  proxied  = each.value.proxied
  comment  = each.value.comment
  priority = each.value.priority
}

resource "cloudflare_account_member" "members" {
  for_each = var.members

  account_id = var.account_id
  email      = each.key
  roles      = each.value.roles
  status     = "accepted"
}
