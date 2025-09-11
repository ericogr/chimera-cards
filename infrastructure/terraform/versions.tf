terraform {
  required_version = ">= 1.0.0"
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 7.16.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.9"
    }
  }
}

provider "oci" {
  tenancy_ocid     = var.tenancy_ocid
  region           = var.region
  user_ocid        = var.user_ocid
  fingerprint      = var.fingerprint
  private_key_path = var.private_key_path
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
