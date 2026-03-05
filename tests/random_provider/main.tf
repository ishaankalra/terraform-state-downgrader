provider "random" {}

# Random string resource - schema_version = 2 in v3.5.1
resource "random_string" "suffix" {
  length  = 16
  special = false
}

# Random password resource - schema_version = 3 in v3.5.1
resource "random_password" "db_password" {
  length  = 16
  special = true
}
