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

Assumptions
- The VM runs Ubuntu (tested on 24.04).
- You have the VM public IP and can SSH as `ubuntu@<IP>` (the default cloud image user).
- You have built the Docker images locally (see the repo Makefiles) and have SSH access from your machine to the VM.

1) Connect and create a deployment user
- SSH into the VM:
  ```bash
  ssh ubuntu@<PUBLIC_IP>
  ```
- Create a new user (interactive):
  ```bash
  sudo adduser deployuser
  ```
- Give the new user sudo privileges:
  ```bash
  sudo usermod -aG sudo deployuser
  ```
  (Alternatively, to allow passwordless sudo — use with caution):
  ```bash
  echo 'deployuser ALL=(ALL) NOPASSWD:ALL' | sudo EDITOR='tee -a' visudo
  ```
- Copy SSH authorized keys (recommended; do not copy private keys):
  ```bash
  sudo mkdir -p /home/deployuser/.ssh
  sudo cp /home/ubuntu/.ssh/authorized_keys /home/deployuser/.ssh/authorized_keys
  sudo chown -R deployuser:deployuser /home/deployuser/.ssh
  sudo chmod 700 /home/deployuser/.ssh
  sudo chmod 600 /home/deployuser/.ssh/authorized_keys
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
  docker save -o /tmp/chimera-backend.tar chimera-backend:latest
  docker save -o /tmp/chimera-frontend.tar chimera-frontend:latest
  ```
- Copy the tar files to the VM (from your workstation):
  ```bash
  scp /tmp/chimera-backend.tar ubuntu@<PUBLIC_IP>:/tmp
  scp /tmp/chimera-frontend.tar ubuntu@<PUBLIC_IP>:/tmp
  ```

7) Load Docker images on the VM
```bash
ssh ubuntu@<PUBLIC_IP>
sudo docker load -i /tmp/chimera-backend.tar
sudo docker load -i /tmp/chimera-frontend.tar
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

13) (Optional) Basic firewall (ufw)
```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

Notes & security recommendations
- Do NOT copy private keys between users; copy only `authorized_keys`.
- Avoid using `NOPASSWD:ALL` in sudo unless you understand the risk.
- Keep `.env` and any private keys off source control.
- If you prefer CI/CD rather than manual image copies, consider pushing images to a private registry and pulling from the VM.
