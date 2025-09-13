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

## Steps to prepare the VM (Ubuntu 24.04 — tested)

**Assumptions**
- The VM runs Ubuntu (tested on 24.04).
- You have the VM public IP and can SSH as `ubuntu@<IP>` (the default cloud image user).
- You have built the Docker images locally (see the repo Makefiles) and have SSH access from your machine to the VM.

1) Connect and create a deployment user
- SSH into the VM:
  ```bash
  ssh ubuntu@<PUBLIC_IP>
  ```
2) Install essential utilities
```bash
sudo apt update
sudo apt install -y nano curl wget git
```

3) Install Docker (official repository)
Follow the official Docker instructions for Ubuntu (short version):
```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg lsb-release
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
```
(Or follow the official doc: https://docs.docker.com/engine/install/ubuntu/)

4) Allow the deploy user to run Docker without sudo
```bash
sudo usermod -aG docker deployuser
# Log out / back in, or use `newgrp docker` to apply immediately in the current session.
```

5) Enable Docker to start on boot
```bash
sudo systemctl enable --now docker
```

6) Build, save and upload images (local machine)
- Build the images locally (from repo root):
  ```bash
  make docker-build
  ```
- Save the images to tar files (specify tag if necessary):
  ```bash
  docker save -o /tmp/chimera-cards-backend.tar ericogr/chimera-cards-backend:latest
  docker save -o /tmp/chimera-cards-frontend.tar ericogr/chimera-cards-frontend:latest
  ```
- Copy the tar files to the VM (from your workstation):
  ```bash
  scp /tmp/chimera-cards-backend.tar ubuntu@<PUBLIC_IP>:/tmp
  scp /tmp/chimera-cards-frontend.tar ubuntu@<PUBLIC_IP>:/tmp
  ```

7) Load Docker images on the VM
```bash
ssh ubuntu@<PUBLIC_IP>
sudo docker load -i /tmp/chimera-cards-backend.tar
sudo docker load -i /tmp/chimera-cards-frontend.tar
```

8) Prepare directory layout and configuration on VM
```bash
# on remote VM
mkdir -p ~/chimera-cards/backend/data
mkdir -p ~/chimera-cards/infrastructure/caddy
mkdir -p ~/chimera-cards/frontend
```
Copy application files from your workstation to the VM:
```bash
scp docker-compose.yml ubuntu@<PUBLIC_IP>:~/chimera-cards/docker-compose.yml
scp backend/chimera_config.json ubuntu@<PUBLIC_IP>:~/chimera-cards/backend/chimera_config.json
scp -r infrastructure/caddy ubuntu@<PUBLIC_IP>:~/chimera-cards/infrastructure/
```
(You can also use `rsync -av` for sync.)

9) Configure Caddy (TLS)
- Edit `~/chimera-cards/infrastructure/caddy/Caddyfile` and set your domain(s).
- For production Let's Encrypt, set a real contact email and ensure the production TLS lines are enabled (and staging commented out). Example:
  ```text
  example.com {
    reverse_proxy /api/* localhost:8080
    file_server
    tls you@example.com
  }
  ```

10) Create the environment file `.env` in `~/chimera-cards` (DO NOT commit)
Example `.env` (adjust values):
```bash
#Backend
GOOGLE_CLIENT_SECRET=xxx
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
OPENAI_API_KEY=sk-xxx

#Frontend
SESSION_SECRET=production
REACT_APP_GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
SESSION_SECURE_COOKIE=1

DOMAIN=chimera.xxx.com.br
CLOUDFLARE_API_TOKEN=xxxx
CADDY_EMAIL=xxx@gmail.com

LOCAL_UID=1000
LOCAL_GID=1000
```
- Keep this file safe and do not commit it to source control.

11) Start the application
```bash
cd ~/chimera-cards
docker compose up -d
```

12) Verify and troubleshoot
- Check containers:
  ```bash
  docker compose ps
  docker compose logs -f
  ```
- Confirm HTTP(S) is working:
  ```bash
  curl -I http://localhost
  ```
- If Caddy manages TLS for your domain, check DNS and firewall rules to allow ports 80 and 443.

### Local smoke test (HTTPS)

After the stack is up (`docker compose up -d`) you can perform a quick HTTPS smoke test that forces the configured domain to resolve to the local host. This is useful when DNS is not yet pointing to the server but you want to verify Caddy, TLS routing and the frontend/backend are serving requests.

1. Confirm the domain configured for Caddy: open the deployment `.env` file in your deployment directory (for example `~/chimera-cards/.env`) and note the `DOMAIN` value. Example:

```bash
grep -E '^DOMAIN=' .env || echo 'DOMAIN is not set in .env'
```

2. Run `curl` on the same host where `docker compose` is running (the VM). Replace `example.com` with the domain from your `.env` file:

```bash
curl -vk --resolve 'example.com:8443:127.0.0.1' https://example.com:8443/
```

Notes:
- `--resolve 'example.com:8443:127.0.0.1'` tells `curl` to resolve DNS for `example.com:8443` to `127.0.0.1` (localhost).
- `https://example.com:8443/` requests HTTPS to that host and port. In this compose setup the host port `8443` maps to container port `443` (see `docker-compose.yml`).
- `-k` disables TLS certificate verification for testing; remove it if you expect a valid certificate.

3. Run `curl` from your workstation (different host)

If you run the command from a different machine (for example your workstation), replace `127.0.0.1` with the VM public IP address:

```bash
curl -vk --resolve 'example.com:8443:198.51.100.23' https://example.com:8443/
```

Replace `198.51.100.23` with the VM's actual public IP.

4. Interpreting results
- A successful test usually returns HTTP `200` and the frontend HTML (for `/`) or the backend response for API paths.
- Use `curl -I` or `curl -v` to inspect headers and TLS details.

5. Troubleshooting
- If the connection is refused, check that Caddy is running and listening on host port `8443`: `docker compose ps` and `docker compose logs caddy`.
- Ensure the `DOMAIN` value in `.env` matches the domain used in the `--resolve` argument.
- If testing from a remote host, ensure firewalls/security groups allow access to port `8443` on the VM.

### Notes & security recommendations
- Do NOT copy private keys between users; copy only `authorized_keys`.
- Avoid using `NOPASSWD:ALL` in sudo unless you understand the risk.
- Keep `.env` and any private keys off source control.
- If you prefer CI/CD rather than manual image copies, consider pushing images to a private registry and pulling from the VM.