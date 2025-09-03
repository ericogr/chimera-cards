terraform {
  required_version = ">= 1.0.0"
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 4.60"
    }
  }
}

provider "oci" {
  region = var.region
}

