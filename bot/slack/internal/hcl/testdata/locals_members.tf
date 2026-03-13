locals {
  members = {
    "luiz1361" = { role = "admin", full_name = "Luiz" }
  }

  teams = {
    "Maintainers" = {
      description = "Maintainers"
      privacy     = "closed"
      members     = ["luiz1361"]
      maintainers = ["luiz1361"]
    }
  }
}
