terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
  }
}

variable "vpc_id" {
  type = string
}

resource "random_string" "subnet_suffix" {
  length  = 8
  special = false
}

output "subnet_id" {
  value = "${var.vpc_id}-${random_string.subnet_suffix.result}"
}