variable "projects" {
  description = "map of Doppler project names to project config"
  type = map(object({
    description = string
    environments = map(object({
      name             = string
      personal_configs = optional(bool, false)
    }))
  }))
  default = {}
}

variable "groups" {
  description = "map of Doppler group names to group config"
  type = map(object({
    description = string
  }))
  default = {}
}

variable "project_access" {
  description = "list of group-to-project access grants"
  type = list(object({
    project      = string
    group        = string
    role         = string
    environments = optional(list(string))
  }))
  default = []

  validation {
    condition = length(var.project_access) == length(distinct([
      for e in var.project_access : "${e.project}:${e.group}"
    ]))
    error_message = "Duplicate project:group pairs in project_access."
  }
}
