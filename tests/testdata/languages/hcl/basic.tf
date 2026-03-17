terraform {
  required_version = ">= 1.6.0"
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

locals {
  name_prefix = "codesieve"
  tags = {
    service = "codesieve"
  }
}

module "network" {
  source     = "./modules/network"
  cidr_block = "10.0.0.0/16"
}

resource "aws_security_group" "web" {
  name = "web"

  ingress {
    from_port = 80
    to_port   = 80
    protocol  = "tcp"
  }

  ingress {
    from_port = 443
    to_port   = 443
    protocol  = "tcp"
  }

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
  }
}

resource "aws_s3_bucket" "app" {
  bucket = "${local.name_prefix}-app"

  dynamic "lifecycle_rule" {
    for_each = ["enabled"]
    content {
      enabled = true
    }
  }
}

data "aws_caller_identity" "current" {}

output "bucket_name" {
  value = resource.aws_s3_bucket.app.bucket
}
