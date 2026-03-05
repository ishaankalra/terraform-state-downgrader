terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.0"
    }
  }
}

provider "random" {}
provider "null" {}

# Root level resource
resource "random_string" "root" {
  length = 16
}

# Call a local module
module "database" {
  source = "./modules/database"

  db_name = "mydb"
}

# Call another module
module "network" {
  source = "./modules/network"

  vpc_cidr = "10.0.0.0/16"
}