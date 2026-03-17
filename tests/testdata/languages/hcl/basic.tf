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
}

module "network" {
  source     = "./modules/network"
  cidr_block = "10.0.0.0/16"
}

resource "aws_s3_bucket" "app" {
  bucket = "${local.name_prefix}-app"
}

data "aws_caller_identity" "current" {}

output "bucket_name" {
  value = resource.aws_s3_bucket.app.bucket
}
