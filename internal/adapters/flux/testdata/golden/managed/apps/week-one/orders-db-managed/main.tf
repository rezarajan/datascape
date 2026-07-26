terraform {
  required_providers {
    neon = {
      source  = "kislerdm/neon"
      version = "0.14.0"
    }
  }
}

# The Neon API key is never a literal value here (golden rule 51): the
# kislerdm/neon provider reads it from the NEON_API_KEY environment
# variable, which the wrapping Terraform CR injects from the
# "neon-api-key" Kubernetes Secret at runner-pod runtime.
provider "neon" {}

# The Neon project is an environment prerequisite (week-two plan
# Revision 4) - d7s never provisions or destroys it, and its id is an
# environment binding that must never appear here as a literal (golden
# rules 22/45: baking it in would make compiled output depend on which
# environment compiled it). The wrapping Terraform CR's spec.varsFrom
# supplies this variable from the same Secret the API key comes from.
variable "project_id" {
  type = string
}

data "neon_project" "orders-db" {
  id = var.project_id
}

resource "neon_branch" "orders-db" {
  project_id = var.project_id
  parent_id  = data.neon_project.orders-db.default_branch_id
  name       = "orders-db"
}

resource "neon_endpoint" "orders-db" {
  project_id = var.project_id
  branch_id  = neon_branch.orders-db.id
  type       = "read_write"
}

# depends_on is required, not stylistic (found live, 2026-07-26): neon_role
# and neon_endpoint have no attribute reference between them, so without
# an explicit edge OpenTofu creates/destroys them in parallel - which
# reproducibly broke both directions against the real API. On create,
# neon_role's password generation raced neon_endpoint's own creation
# ("no read-write endpoint for branch"). On destroy, deleting the role
# concurrently with the endpoint reproducibly broke the provider's SDK
# ("unexpected end of JSON input" - the SDK decodes an empty response
# body it never checks for, a real upstream fragility, but only reachable
# because of the missing dependency edge). Confirmed by removing this
# line and reproducing both failures twice, then adding it back and
# succeeding twice - not a guess.
resource "neon_role" "orders-db" {
  project_id = var.project_id
  branch_id  = neon_branch.orders-db.id
  name       = "orders-db"
  depends_on = [neon_endpoint.orders-db]
}

resource "neon_database" "orders-db" {
  project_id = var.project_id
  branch_id  = neon_branch.orders-db.id
  name       = "orders-db"
  owner_name = neon_role.orders-db.name
}

# Connection outputs: tofu-controller's spec.writeOutputsToSecret (see
# TerraformWriteOutputsToSecret) writes each of these into the Kubernetes
# Secret named by the component's declared credentials.secretRef.name —
# the output name becomes the Secret's data key, verbatim, so these five
# names ARE the consumer-facing contract (mirrors CNPG's own "username"/
# "password" convention for the self-hosted placement, plus host/port/
# database for a placement where there is no in-cluster Service to
# default those from).
output "host" {
  value = neon_endpoint.orders-db.host
}

output "port" {
  # Neon's connection proxy always listens on 5432 - the provider
  # exposes no separate port attribute because there is only ever one.
  value = "5432"
}

output "database" {
  value = neon_database.orders-db.name
}

output "username" {
  value = neon_role.orders-db.name
}

output "password" {
  value     = neon_role.orders-db.password
  sensitive = true
}
