terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.4.0"
    }
  }
}

variable "db_name" {
  type = string
}

resource "random_password" "db_password" {
  length  = 32
  special = true
}

resource "local_file" "db_config" {
  filename = "${path.module}/db_config.txt"
  content  = "Database: ${var.db_name}, Password: ${random_password.db_password.result}"
}

output "db_password" {
  value     = random_password.db_password.result
  sensitive = true
}