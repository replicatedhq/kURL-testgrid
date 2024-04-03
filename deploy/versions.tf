terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.5.0"
    }
    equinix = {
      source = "equinix/equinix"
      version = "1.34.0"
    }
  }
  required_version = ">= 1.0.0"
}
