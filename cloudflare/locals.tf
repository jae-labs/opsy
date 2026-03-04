locals {
  account_id = "9fac8f4aa513faa30a47e34439f9702c"

  zones = {
    "justanother.engineer" = {}
  }

  dns_records = {
    "justanother.engineer" = {
      "ha-a" = {
        type    = "A"
        name    = "ha"
        content = "46.7.7.84"
        proxied = true
        comment = "Home Assistant"
      }
      "root-a-1" = {
        type    = "A"
        name    = "justanother.engineer"
        content = "185.199.108.153"
        proxied = true
        comment = "GitHub Pages"
      }
      "root-a-2" = {
        type    = "A"
        name    = "justanother.engineer"
        content = "185.199.109.153"
        proxied = true
        comment = "GitHub Pages"
      }
      "root-a-3" = {
        type    = "A"
        name    = "justanother.engineer"
        content = "185.199.110.153"
        proxied = true
        comment = "GitHub Pages"
      }
      "root-a-4" = {
        type    = "A"
        name    = "justanother.engineer"
        content = "185.199.111.153"
        proxied = true
        comment = "GitHub Pages"
      }
      "www-cname" = {
        type    = "CNAME"
        name    = "www"
        content = "justanother.engineer"
        proxied = true
      }
      "mx-zoho-1" = {
        type     = "MX"
        name     = "justanother.engineer"
        content  = "mx.zoho.com"
        priority = 10
      }
      "mx-zoho-2" = {
        type     = "MX"
        name     = "justanother.engineer"
        content  = "mx2.zoho.com"
        priority = 20
      }
      "mx-zoho-3" = {
        type     = "MX"
        name     = "justanother.engineer"
        content  = "mx3.zoho.com"
        priority = 30
      }
      "txt-algolia" = {
        type    = "TXT"
        name    = "algolia-site-verification"
        content = "\"3B4606A0624E09DD\""
        comment = "Algolia DocSearch Domain Verification"
      }
      "txt-gh-org" = {
        type    = "TXT"
        name    = "_gh-jae-labs-o"
        content = "\"9063a009e9\""
        comment = "GitHub Org Domain Validation"
      }
      "txt-gh-pages-jae-labs" = {
        type    = "TXT"
        name    = "_github-pages-challenge-jae-labs"
        content = "\"d607d0093ff674e6c0c362437b7fc4\""
        comment = "jae-labs"
      }
      "txt-gh-pages-luiz1361" = {
        type    = "TXT"
        name    = "_github-pages-challenge-luiz1361"
        content = "\"9045fccfcaab3637dbf2615f1691b3\""
        comment = "luiz1361"
      }
      "txt-gitlab-pages" = {
        type    = "TXT"
        name    = "justanother.engineer"
        content = "_gitlab-pages-verification-code.justanother.engineer TXT gitlab-pages-verification-code=3b0dedc3c251e299750d35c9843989fe"
      }
      "txt-google-verification" = {
        type    = "TXT"
        name    = "justanother.engineer"
        content = "google-site-verification=X_3eoCG2CVQl0pCspNHhKplB2hkwJGKeqzVmqJknhzk"
      }
      "txt-spf" = {
        type    = "TXT"
        name    = "justanother.engineer"
        content = "v=spf1 include:zoho.com ~all"
      }
      "blabla" = {
        type    = "A"
        name    = "@"
        content = "bla"
        proxied = false
      }
    }
  }

  members = {}
}
