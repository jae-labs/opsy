module "cloudflare" {
  source = "../modules/cloudflare"

  account_id  = local.account_id
  zones       = local.zones
  dns_records = local.dns_records
  members     = local.members
}

