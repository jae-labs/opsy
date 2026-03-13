module "doppler" {
  source = "../modules/doppler"

  projects       = local.projects
  groups         = local.groups
  project_access = local.project_access
}
