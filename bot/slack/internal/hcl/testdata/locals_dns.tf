locals {
  dns_records = {
    "justanother.engineer" = {
      "ha-a" = {
        type    = "A"
        name    = "ha"
        content = "46.7.7.84"
        proxied = true
        comment = "Home Assistant"
      }
      "ha-cname" = {
        type    = "CNAME"
        name    = "homeassistant"
        content = "ha.justanother.engineer"
        proxied = true
        comment = "Home Assistant alias"
      }
      "grafana-a" = {
        type    = "A"
        name    = "grafana"
        content = "46.7.7.84"
        proxied = true
        comment = "Grafana"
      }
      "vpn-a" = {
        type    = "A"
        name    = "vpn"
        content = "46.7.7.84"
        proxied = false
      }
      "root-a" = {
        type    = "A"
        name    = "@"
        content = "76.76.21.21"
        proxied = true
      }
      "www-cname" = {
        type    = "CNAME"
        name    = "www"
        content = "justanother.engineer"
        proxied = true
      }
      "mx-zoho-1" = {
        type     = "MX"
        name     = "@"
        content  = "mx.zoho.eu"
        priority = 10
        comment  = "Zoho Mail primary"
      }
      "mx-zoho-2" = {
        type     = "MX"
        name     = "@"
        content  = "mx2.zoho.eu"
        priority = 20
        comment  = "Zoho Mail secondary"
      }
      "mx-zoho-3" = {
        type     = "MX"
        name     = "@"
        content  = "mx3.zoho.eu"
        priority = 50
      }
      "txt-spf" = {
        type    = "TXT"
        name    = "@"
        content = "v=spf1 include:zoho.eu ~all"
        comment = "SPF record"
      }
      "txt-dkim" = {
        type    = "TXT"
        name    = "zmail._domainkey"
        content = "v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA"
        comment = "DKIM for Zoho"
      }
      "txt-verify" = {
        type    = "TXT"
        name    = "@"
        content = "zoho-verification=zb123456.zmverify.zoho.eu"
      }
      "api-cname" = {
        type    = "CNAME"
        name    = "api"
        content = "workers.justanother.engineer"
        proxied = true
        comment = "API gateway"
      }
      "blog-cname" = {
        type    = "CNAME"
        name    = "blog"
        content = "justanother.engineer"
        proxied = true
      }
      "mail-cname" = {
        type    = "CNAME"
        name    = "mail"
        content = "business.zoho.eu"
        proxied = false
        comment = "Zoho webmail"
      }
      "aaaa-test" = {
        type    = "AAAA"
        name    = "ipv6"
        content = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
        proxied = true
        comment = "IPv6 test record"
      }
    }
  }
}
