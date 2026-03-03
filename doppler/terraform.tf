terraform {
  required_version = ">= 1.5"

  required_providers {
    doppler = {
      source  = "DopplerHQ/doppler"
      version = "~> 1.13"
    }
  }

  backend "gcs" {
    bucket = "gh-jae-labs-terraform"
    prefix = "doppler"
  }
}
