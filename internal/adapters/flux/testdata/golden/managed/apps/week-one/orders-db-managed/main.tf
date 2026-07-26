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

resource "neon_project" "orders-db" {
  name = "orders-db"
}

resource "neon_role" "orders-db" {
  project_id = neon_project.orders-db.id
  branch_id  = neon_project.orders-db.default_branch_id
  name       = "orders-db"
}

resource "neon_database" "orders-db" {
  project_id = neon_project.orders-db.id
  branch_id  = neon_project.orders-db.default_branch_id
  name       = "orders-db"
  owner_name = neon_role.orders-db.name
}
