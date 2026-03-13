terraform {
  required_version = ">= 1.5"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5"
    }
  }

  backend "gcs" {
    bucket = "gh-jae-labs-terraform"
    prefix = "cloudflare"
  }
}
