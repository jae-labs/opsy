locals {
  projects = {
    "example-project" = {
      description = "An example project with some sample secrets."
      environments = {
        "dev" = { name = "Development", personal_configs = true }
        "stg" = { name = "Staging" }
        "prd" = { name = "Production" }
      }
    }
  }
}
