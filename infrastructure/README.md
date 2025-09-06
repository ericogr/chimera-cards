# Infrastructure

This folder contains infrastructure-as-code and helper scripts used to
provision cloud resources for Chimera Cards.

## Structure

- [`bootstrap/`](./bootstrap/) — helper script to create OCI credentials and a `chimera-cards` compartment (see [`infrastructure/bootstrap/README.md`](./bootstrap/README.md)).
- [`terraform/`](./terraform/) — Terraform code to provision VCN, compute instances and a load balancer (see [`infrastructure/terraform/README.md`](./terraform/README.md)).

## Quick commands

Run the bootstrap helper (creates user / API key and optional compartment):

```bash
make infra-bootstrap BOOTSTRAP_ARGS="--profile DEFAULT --outdir ./infrastructure/bootstrap/oci-creds --email you@example.com"
```

Terraform workflow (from repository root):

```bash
make terraform-init
make terraform-plan
make terraform-apply
```

## Notes

- The bootstrap script writes a private key to the chosen output directory
  with `0600` permissions — keep that file safe and never commit it to
  source control.
