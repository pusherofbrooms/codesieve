terraform {
  source = "../modules/app"
}

include "root" {
  path = find_in_parent_folders()
}

inputs = {
  aws_region = "us-east-1"
}
