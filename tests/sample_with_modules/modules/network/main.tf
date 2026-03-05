terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.9.0"
    }
  }
}

variable "vpc_cidr" {
  type = string
}

resource "random_id" "vpc_id" {
  byte_length = 4
}

resource "time_static" "created_at" {}

# Nested module
module "subnet" {
  source = "./subnet"

  vpc_id = random_id.vpc_id.hex
}

output "vpc_id" {
  value = random_id.vpc_id.hex
}