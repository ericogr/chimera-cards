#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

TS=$(date -u +%Y%m%d%H%M%S)
ADMIN_PROFILE="DEFAULT"
NEW_USERNAME="chimera-terraform-${TS}"
EMAIL="${NEW_USERNAME}@example.com"
NEW_GROUPNAME="chimera-terraform-group-${TS}"
POLICY_NAME="chimera-terraform-policy-${TS}"
OUTDIR="infrastructure/bootstrap/oci-creds-${TS}"
TARGET_COMP_OCID=""
TENANCY_OCID=""
REGION=""

print_usage() {
  cat <<'USAGE'
Usage: bootstrap-oci.sh [options]

This script bootstraps OCI credentials for Terraform. It will:
  - generate an RSA API keypair
  - create an OCI user
  - upload the public API key to the user
  - create a group and add the user to it
  - create a policy that grants the group permissions (either in a compartment or in the tenancy)

Requirements: `oci` CLI configured with an admin profile and `openssl` available in PATH.

Options:
  -p, --profile PROFILE         OCI CLI profile to use for admin actions (default: DEFAULT)
  -t, --tenancy TENANCY_OCID    Tenancy OCID (will try to read from profile if omitted)
  -c, --compartment COMPARTMENT_OCID  Target compartment OCID for policy (optional)
  -n, --username USERNAME       Username to create (default: chimera-terraform-<ts>)
  -e, --email EMAIL             Email address to use for the new user (default: <username>@example.com)
  -g, --group GROUPNAME         Group name (default: chimera-terraform-group-<ts>)
  -o, --outdir DIR              Output directory for keys and files (default: infrastructure/bootstrap/oci-creds-<ts>)
  -r, --region REGION           OCI region (will try to read from profile if omitted)
  -h, --help                    Show this help and exit

Example:
  ./bootstrap-oci.sh --profile DEFAULT --compartment ocid1.compartment.oc1..aaaaaaa --outdir ./oci-creds
USAGE
}

# Parse args
while [ $# -gt 0 ]; do
  case "$1" in
    -p|--profile)
      ADMIN_PROFILE="$2"; shift 2;;
    -t|--tenancy)
      TENANCY_OCID="$2"; shift 2;;
    -c|--compartment)
      TARGET_COMP_OCID="$2"; shift 2;;
    -n|--username)
      NEW_USERNAME="$2"; shift 2;;
    -e|--email)
      EMAIL="$2"; shift 2;;
    -g|--group)
      NEW_GROUPNAME="$2"; shift 2;;
    -o|--outdir)
      OUTDIR="$2"; shift 2;;
    -r|--region)
      REGION="$2"; shift 2;;
    -h|--help)
      print_usage; exit 0;;
    *)
      echo "Unknown option: $1"; print_usage; exit 1;;
  esac
done

# Preconditions
if ! command -v oci >/dev/null 2>&1; then
  echo "Error: OCI CLI ('oci') not found in PATH. Install and configure it first." >&2
  exit 1
fi

if ! command -v openssl >/dev/null 2>&1; then
  echo "Error: openssl not found in PATH. Install OpenSSL to generate keypairs." >&2
  exit 1
fi

CONFIG_FILE="${OCI_CONFIG_FILE:-$HOME/.oci/config}"
if [ -z "${TENANCY_OCID}" ]; then
  if [ -r "$CONFIG_FILE" ]; then
    TENANCY_OCID=$(awk -v prof="$ADMIN_PROFILE" 'BEGIN{inside=0} $0=="["prof"]"{inside=1;next} inside && /^\[/ {exit} inside && $0 ~ /^tenancy[[:space:]]*=/ {split($0,a,"="); gsub(/^[ \t]+|[ \t]+$/,"",a[2]); print a[2]; exit}' "$CONFIG_FILE" || true)
  fi
fi

if [ -z "${TENANCY_OCID}" ]; then
  echo "Tenancy OCID not provided and not found in profile '$ADMIN_PROFILE' (file: $CONFIG_FILE)." >&2
  echo "Provide --tenancy TENANCY_OCID or set tenancy in the OCI config profile." >&2
  exit 1
fi

if [ -z "${REGION}" ]; then
  if [ -r "$CONFIG_FILE" ]; then
    REGION=$(awk -v prof="$ADMIN_PROFILE" 'BEGIN{inside=0} $0=="["prof"]"{inside=1;next} inside && /^\[/ {exit} inside && $0 ~ /^region[[:space:]]*=/ {split($0,a,"="); gsub(/^[ \t]+|[ \t]+$/,"",a[2]); print a[2]; exit}' "$CONFIG_FILE" || true)
  fi
fi

if [ -z "${REGION}" ]; then
  echo "Warning: OCI region not found in profile '$ADMIN_PROFILE'. You can pass --region REGION or set it in your OCI config." >&2
fi

# Prepare output folder
mkdir -p "$OUTDIR"
PRIVATE_KEY="$OUTDIR/oci_api_key.pem"
PUBLIC_KEY="$OUTDIR/oci_api_key_public.pem"

echo "Generating RSA keypair..."
umask 077
openssl genpkey -algorithm RSA -out "$PRIVATE_KEY" -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in "$PRIVATE_KEY" -out "$PUBLIC_KEY"
chmod 600 "$PRIVATE_KEY"

# Create the OCI user
echo "Creating user '$NEW_USERNAME' in tenancy: $TENANCY_OCID (profile: $ADMIN_PROFILE)"
if ! printf '%s' "$EMAIL" | grep -qE '^[^@]+@[^@]+\.[^@]+'; then
  echo "Provided email ($EMAIL) doesn't look like a valid email address. Use --email user@example.com" >&2
  exit 1
fi

USER_OCID=$(oci --profile "$ADMIN_PROFILE" iam user create --compartment-id "$TENANCY_OCID" --name "$NEW_USERNAME" --auth security_token \
  --description "Terraform user for Chimera (created by bootstrap script)" --email "$EMAIL" --query 'data.id' --raw-output)

if [ -z "$USER_OCID" ]; then
  echo "Failed to create OCI user." >&2
  exit 1
fi

echo "Created user: $USER_OCID"

# Upload API key for the user
echo "Uploading API public key to user..."
FINGERPRINT=$(oci --profile "$ADMIN_PROFILE" iam user api-key upload --user-id "$USER_OCID" --key-file "$PUBLIC_KEY" --query 'data.fingerprint' --raw-output --auth security_token)
if [ -z "$FINGERPRINT" ]; then
  echo "Failed to upload API key for user $USER_OCID" >&2
  exit 1
fi

# Create a group and add the user
echo "Creating group '$NEW_GROUPNAME'"
GROUP_OCID=$(oci --profile "$ADMIN_PROFILE" iam group create --compartment-id "$TENANCY_OCID" --name "$NEW_GROUPNAME" \
  --description "Group for Terraform (created by bootstrap script)" --query 'data.id' --raw-output --auth security_token)
if [ -z "$GROUP_OCID" ]; then
  echo "Failed to create group" >&2
  exit 1
fi

echo "Adding user to group"
oci --profile "$ADMIN_PROFILE" iam group add-user --group-id "$GROUP_OCID" --user-id "$USER_OCID" --auth security_token>/dev/null

# Ensure a compartment named 'chimera-cards' exists (if no compartment OCID provided)
if [ -z "$TARGET_COMP_OCID" ]; then
  echo "No compartment OCID provided. Looking for compartment named 'chimera-cards' under tenancy..."
  EXISTING_COMP_OCID=$(oci --profile "$ADMIN_PROFILE" iam compartment list --compartment-id "$TENANCY_OCID" --all --query 'data[?name==`chimera-cards`].id | [0]' --raw-output --auth security_token 2>/dev/null || true)
  if [ -n "$EXISTING_COMP_OCID" ] && [ "$EXISTING_COMP_OCID" != "None" ]; then
    TARGET_COMP_OCID="$EXISTING_COMP_OCID"
    echo "Found existing compartment 'chimera-cards': $TARGET_COMP_OCID"
  else
    echo "Compartment 'chimera-cards' not found. Creating a new compartment under the tenancy..."
    CREATED_COMP_OCID=$(oci --profile "$ADMIN_PROFILE" iam compartment create --compartment-id "$TENANCY_OCID" --name "chimera-cards" \
      --description "Compartment for Chimera resources (created by bootstrap script)" --query 'data.id' --raw-output --auth security_token 2>/dev/null || true)
    if [ -n "$CREATED_COMP_OCID" ] && [ "$CREATED_COMP_OCID" != "None" ]; then
      TARGET_COMP_OCID="$CREATED_COMP_OCID"
      echo "Created compartment 'chimera-cards': $TARGET_COMP_OCID"
    else
      echo "Failed to create compartment 'chimera-cards'. Continuing without compartment (policy will apply to tenancy)." >&2
      TARGET_COMP_OCID=""
    fi
  fi
fi

# Create a policy for the group
if [ -n "$TARGET_COMP_OCID" ]; then
  echo "Fetching compartment name for $TARGET_COMP_OCID"
  COMP_NAME=$(oci --profile "$ADMIN_PROFILE" iam compartment get --compartment-id "$TARGET_COMP_OCID" --query 'data.name' --raw-output --auth security_token 2>/dev/null || true)
  if [ -z "$COMP_NAME" ]; then
    echo "Could not determine compartment name for $TARGET_COMP_OCID. Falling back to tenancy-level policy." >&2
    STATEMENT="Allow group $NEW_GROUPNAME to manage all-resources in tenancy"
  else
    STATEMENT="Allow group $NEW_GROUPNAME to manage all-resources in compartment $COMP_NAME"
  fi
else
  STATEMENT="Allow group $NEW_GROUPNAME to manage all-resources in tenancy"
fi

STATEMENTS_JSON=$(printf '["%s"]' "$STATEMENT")

echo "Creating policy for group (statement: $STATEMENT)"
POLICY_OCID=$(oci --profile "$ADMIN_PROFILE" iam policy create --compartment-id "$TENANCY_OCID" --name "$POLICY_NAME" \
  --description "Policy for Terraform (created by bootstrap script)" --statements "$STATEMENTS_JSON" --query 'data.id' --raw-output --auth security_token 2>/dev/null || true)

if [ -z "$POLICY_OCID" ]; then
  echo "Warning: failed to create policy automatically. You may need to create the policy manually with the following statement:" >&2
  echo "  $STATEMENT"
else
  echo "Created policy: $POLICY_OCID"
fi

# Write a small provider env file and a config profile snippet
cat > "$OUTDIR/oci-provider.env" <<EOF
# Export these when running Terraform or your local shell session
export OCI_TENANCY="$TENANCY_OCID"
export OCI_USER="$USER_OCID"
export OCI_FINGERPRINT="$FINGERPRINT"
export OCI_KEY_FILE="$PRIVATE_KEY"
export OCI_REGION="$REGION"
EOF

cat > "$OUTDIR/oci-config-profile.txt" <<EOF
# Add this profile to your ~/.oci/config if you want a named profile for Terraform
[terraform]
user=$USER_OCID
fingerprint=$FINGERPRINT
key_file=$PRIVATE_KEY
tenancy=$TENANCY_OCID
region=$REGION
EOF

# Summary
cat <<SUMMARY
Bootstrap complete — credentials and files have been written to: $OUTDIR

Files created:
  - Private key: $PRIVATE_KEY (permission 600)
  - Public key:  $PUBLIC_KEY
  - Env file:    $OUTDIR/oci-provider.env  (use: 'source oci-provider.env')
  - OCI config snippet: $OUTDIR/oci-config-profile.txt (append to ~/.oci/config as [terraform])

To use these credentials with Terraform you can either:
  - export variables from the env file:  'source $OUTDIR/oci-provider.env' and then run 'make terraform-init && make terraform-apply'
  - or append the config snippet to your '~/.oci/config' and run Terraform with '--profile terraform'

Notes & security:
  - Keep the private key safe. Do not commit it to source control.
  - The script attempts to create a policy automatically; if that fails, create the policy manually using the printed statement above.
SUMMARY

exit 0
