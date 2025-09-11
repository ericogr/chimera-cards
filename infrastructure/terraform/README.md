# Chimera - Terraform (OCI)

This directory contains Terraform automation used to provision Oracle Cloud resources (VCN, subnets, compute instances, and a Load Balancer).

## Contents

- `main.tf` — VCN, Subnet, Internet Gateway, Route Table, Security List, Compute Instance, Load Balancer
- `variables.tf` — configurable variables
- `outputs.tf` — public IP addresses (instance and LB)
- `versions.tf` — provider and Terraform versions
- `terraform.tfvars.example` — example variables

## Quick start

1. Copy the example variables and edit at least `compartment_ocid` and `ssh_public_key`:

```bash
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars
```

2. If you have an `image_ocid`, set it; otherwise the code will attempt to locate an image by `image_display_name` (this may fail depending on tenancy/region).

3. Initialize and apply:

```bash
terraform init
terraform plan
terraform apply
```

## Notes / assumptions

- `ssh_public_key` should contain your `~/.ssh/id_rsa.pub` content (OpenSSH format).
- The automatic image lookup uses `image_display_name` as a fallback; prefer using `image_ocid` when possible.
- To configure OCPUs/memory, use a flex shape (e.g., `VM.Standard.E3.Flex`) and set `use_flex_shape = true`.
- `storage_size_gb` sets the boot volume size (overrides the image default).
- The Load Balancer uses TCP and forwards connections from the `lb_listen_*` ports to the backend ports on the instance.

## Optional: Cloudflare DNS

- Terraform can optionally create or update a DNS record in Cloudflare pointing to the load balancer. See the example variables in `terraform.tfvars.example`.
- The Cloudflare provider reads the API token from the environment variable `CLOUDFLARE_API_TOKEN`. You can also set `cloudflare_api_token` in `terraform.tfvars`, but storing tokens in environment variables is recommended.
- Enable the feature by setting `cloudflare_enabled = true` and providing either `cloudflare_zone` (zone name) or `cloudflare_zone_id`.

## Environment variables

- `CLOUDFLARE_API_TOKEN` — Cloudflare API token. Export this in your environment when using `cloudflare_enabled = true` (recommended). Alternatively set `cloudflare_api_token` in `terraform.tfvars`.
- OCI credentials — the bootstrap script creates an `oci-provider.env` file in the output directory. After running the bootstrap, load the variables into your shell with `source <OUTDIR>/oci-provider.env` before running Terraform, or configure the provider variables manually.

## Bootstrap dependency

This Terraform configuration depends on the bootstrap helper in `../bootstrap` (see `infrastructure/bootstrap/README.md`). Run the bootstrap first to generate OCI credentials and the provider variables required by Terraform.

