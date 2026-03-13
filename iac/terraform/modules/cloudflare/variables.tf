variable "account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "zones" {
  description = "map of domain names to zone config"
  type = map(object({
    type = optional(string, "full")
  }))
  default = {}
}

variable "dns_records" {
  description = "map of zone names to a map of DNS record configs; outer key = zone, inner key = logical name"
  type = map(map(object({
    type     = string
    name     = string
    content  = string
    ttl      = optional(number, 1) # 1 = automatic (required for proxied records)
    proxied  = optional(bool, false)
    comment  = optional(string)
    priority = optional(number)
  })))
  default = {}
}

variable "members" {
  description = "map of email addresses to account member config with role IDs"
  type = map(object({
    roles = list(string)
  }))
  default = {}
}
