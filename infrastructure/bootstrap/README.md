# OCI Bootstrap (infrastructure/bootstrap)

This folder contains a convenience script to bootstrap OCI credentials that
can be used by the Terraform configuration in [`infrastructure/terraform`](../terraform/README.md).

What the script does
- Generates an RSA API keypair (PEM private key + PEM public key).
- Creates an OCI user and uploads the public API key to the user.
- Creates a group and adds the new user to it.
- Attempts to create an IAM policy that grants the group permissions to
  manage resources (either in a specified compartment or in the tenancy).
- Writes a small env file and a `~/.oci/config` profile snippet to the
  chosen output directory.

Prerequisites & preflight checklist

Follow these steps before running the bootstrap script to avoid common
permission and configuration issues.

1) Install the OCI CLI

- **Official installer (Linux/macOS):**

  ```bash
  bash -c "$(curl -L https://raw.githubusercontent.com/oracle/oci-cli/master/scripts/install/install.sh)"
  ```

  - Ensure the installer adds `oci` to your `PATH` (you may need to add
    `~/.local/bin` to `PATH` if installed with `--user`).

- **Alternative (pip):**

  ```bash
  python3 -m pip install --user oci-cli
  export PATH="$HOME/.local/bin:$PATH"
  ```

- **Verify:**

  ```bash
  oci --version
  ```

2) Authenticate with OCI (configure `~/.oci/config`)

```bash
oci session authenticate
```

3) Verify connectivity and tenancy

- Check compartments as a sanity test (replace `<tenancy-ocid>`):

  ```bash
  oci iam compartment list --compartment-id <tenancy-ocid> --all
  ```

4) Install or verify OpenSSL

- **Verify:** `openssl version`
- **macOS (Homebrew):** `brew install openssl`
- **Linux:** install via your distro package manager (usually present).

Usage

Use the Makefile target and forward arguments via `BOOTSTRAP_ARGS`, or
call the script directly. Examples:

- **Let the script find/create `chimera-cards` (defaults):**

   ```bash
   make infra-bootstrap BOOTSTRAP_ARGS="--profile DEFAULT --outdir ./infrastructure/bootstrap/oci-creds --email you@example.com"
   ```

- **Explicit tenancy (recommended if tenancy is not in your profile):**

  ```bash
  make infra-bootstrap BOOTSTRAP_ARGS="--profile DEFAULT --tenancy ocid1.tenancy.oc1..XXXXXXXX --outdir ./infrastructure/bootstrap/oci-creds"
  ```

- **Or call the script directly:**

  ```bash
  bash infrastructure/bootstrap/bootstrap-oci.sh --profile DEFAULT --outdir ./infrastructure/bootstrap/oci-creds
  ```

- **Target an existing compartment by OCID:**

  ```bash
  bash infrastructure/bootstrap/bootstrap-oci.sh --profile DEFAULT --compartment ocid1.compartment.oc1..YYYYYYYYYYYY --outdir ./infrastructure/bootstrap/oci-creds
  ```

Post-run

- **Find created files:**

  ```bash
  OUTDIR=$(ls -dt infrastructure/bootstrap/oci-creds-* 2>/dev/null | head -1)
  ls -la "$OUTDIR"
  ```

- **Load provider env vars:**

  ```bash
  source "$OUTDIR/oci-provider.env"
  ```

- **Set Terraform variables:** copy [`infrastructure/terraform/terraform.tfvars.example`](../terraform/terraform.tfvars.example) to
  [`infrastructure/terraform/terraform.tfvars`](../terraform/terraform.tfvars) and set `compartment_ocid` to the created
  compartment OCID (the script prints the created compartment OCID to stdout).

- **Initialize and run Terraform:**

  ```bash
  make terraform-init
  make terraform-plan
  make terraform-apply
  ```

Security notes
- The private key is written to the output directory with `0600` file
  permissions; keep this file safe and do not commit it to source control.
- The script attempts to create a broad policy (manage all-resources) for
  convenience; review the created policy and narrow its scope for
  production environments.

If you prefer not to run the script, you can create the following items
manually: an OCI user, an API key for that user, a group containing the
user, and an IAM policy that allows the group to manage resources in the
target compartment.
