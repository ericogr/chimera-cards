# Quimera Cards

Quimera Cards is a web-based tactical turn-based combat game where each
player builds two hybrids (each composed of 2–3 animals) and battles in
simultaneous-planning rounds. The backend is written in Go (Gin + GORM)
and the frontend is a React + TypeScript single-page app.

This README documents the current project layout, how to run the app,
the public API (including image generation endpoints) and a concise
summary of the combat mechanics.

---

## Production Deployment

This repository includes a `docker-compose.yml` that builds and runs the
frontend and backend as containers. The production-oriented flow is:

- Build the frontend static bundle (Node) and serve it with `nginx`.
- The frontend's `nginx` is configured to proxy `/api/*` requests to the
  backend service (`http://chimera-backend:8080`) inside the Docker Compose
  network. This allows the SPA to use same-origin API paths (`/api/...`)
  and preserves HttpOnly session cookies (no CORS required).

Key points:

- Frontend static assets are built inside `frontend/Dockerfile` and the
  final image uses `nginx` with a small `frontend/nginx.conf` that
  proxies `/api/` to the `chimera-backend` service.
- The SPA already uses `/api` as its API prefix. The Compose file sets
  `REACT_APP_API_BASE_URL` to `/api` by default so builds remain
  consistent, but the nginx proxy is what ensures same-origin routing.

Environment variables (example, set in a root `.env` file or in your
deployment system):

```
# Backend (required)
SESSION_SECRET=your-strong-session-secret
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
OPENAI_API_KEY=sk-...   # required for hybrid name/image generation
SESSION_SECURE_COOKIE=1 # set to 1 when serving over HTTPS

# Frontend build-time args (mapped in docker-compose)
REACT_APP_GOOGLE_CLIENT_ID=your-google-client-id
# REACT_APP_API_BASE_URL defaults to /api in docker-compose.yml
```

Build and run with Docker Compose:

```
docker compose build
docker compose up -d
```

Logs and healthchecks:

- Frontend healthcheck: nginx serves `/`, logs available via
  `docker compose logs -f chimera-frontend`.
- Chimera backend healthcheck: configured to call the internal `/healthcheck` binary.

Production recommendations & notes

- Do not expose the backend port publicly if you don't need to. To keep
  the backend internal to the Compose network change the backend service
  `ports:` entry to `expose:` so only other services (nginx) can reach it.
  Example change in `docker-compose.yml`:

  ```yaml
  services:
    chimera-backend:
      # ...
      # replace this:
      # ports:
      #   - "8080:8080"
      # with this to avoid publishing the port to the host:
      expose:
        - "8080"
  ```

- TLS / HTTPS: the `frontend` container's nginx listens on port 80. In
  production you should terminate TLS at a fronting reverse proxy (Traefik,
  Caddy, cloud LB) and route traffic to the `frontend` container. If you
  do this, make sure `SESSION_SECURE_COOKIE=1` so the backend will set
  the session cookie with the `Secure` flag.

-- Persistent storage / DB: the project currently uses SQLite
  (`backend/data/quimera.db`) which is intended for development. For a real
  production deployment migrate the storage layer to a production-ready
  database (Postgres/MySQL) and update `internal/storage` accordingly.

- Secrets: never commit `SESSION_SECRET`, `GOOGLE_CLIENT_SECRET`, or
  `OPENAI_API_KEY` to source control. Use your deployment platform's
  secrets management (or environment variables injected securely).

If you want, I can also:

- Update `docker-compose.yml` to make the `chimera-backend` internal-only (replace
  `ports` with `expose`).
- Add an example `traefik` or `nginx` reverse-proxy entry to the compose
  file to show TLS termination and routing.


## Highlights
- Backend refactor: single executable at `backend/cmd/quimera-cards` and
  modularized `internal/` packages (api, service, engine, storage, hybrid
  image/name helpers, OpenAI client, etc.).
- OpenAI integration: AI-generated hybrid names and images (DALL·E-like
  image generation). Images are cached in the database.
- Session-based Google login: `/auth/google/oauth2callback` exchanges
  the authorization code and sets an HttpOnly session cookie.

---

## Repo Layout

- `backend/` — Go server module
  - `cmd/quimera-cards/` — main executable entrypoint
  - `internal/api/` — HTTP handlers and route wiring
  - `internal/service/` — game lifecycle and domain logic
  - `internal/engine/` — combat resolution engine
  - `internal/storage/` — SQLite repository & DB migration/seeding
  - `internal/hybridname`, `hybridimage` — name/image caching and generation
  - `internal/openaiclient` — OpenAI API integration
  - `chimera_config.json` — example animal configuration (defaults to `./chimera_config.json`)
  - `data/quimera.db` — runtime SQLite DB created in `backend/data/` (development)

- `frontend/` — React + TypeScript app (CRA)
  - `src/` — components, views, types
  - `.env` / `.env.development.local` — frontend env variables

- Root files
  - `Makefile` — commands for building/running backend & frontend
  - `go.work` — development workspace

---

## Prerequisites

- Go 1.20+ (or as required by `backend/go.mod`)
- Node.js + npm (for the frontend)
- `make`

## Environment variables

Required for the backend (set in your shell or in a `.env` file at `backend/`):

```
GOOGLE_CLIENT_ID="..."
GOOGLE_CLIENT_SECRET="..."
SESSION_SECRET="A_LONG_RANDOM_STRING"
SESSION_SECURE_COOKIE="0" # 0 for local HTTP; 1 for HTTPS
OPENAI_API_KEY="sk-..."   # required for hybrid names/images
CHIMERA_CONFIG="./chimera_config.json" # optional, defaults to this path
```

Frontend (create `frontend/.env`):

```
REACT_APP_GOOGLE_CLIENT_ID="..."
```

Notes:
- Do not commit secrets. Keep `OPENAI_API_KEY`, `SESSION_SECRET`, and Google
  credentials out of source control.
- The backend process validates that the required env vars are present on
  startup and will exit if any required vars are missing.

---

## Installation & Running

Install frontend deps and run both services (recommended):

```bash
git clone <repo>
cd quimera-cards
make frontend-install
 # set the backend env vars in your shell
make run
```

Or run components individually:

- Backend only:
  - `make backend-run` (reads `backend/chimera_config.json` by default)
- Frontend only:
  - `make frontend-start`

The frontend runs on `http://localhost:3000` (CRA) and proxies API calls to
the backend which binds to the address in `chimera_config.json` (default
`:8080`). API root: `http://localhost:8080/api`.

The development SQLite database is created/seeded automatically at
`backend/data/quimera.db` (this behavior is intended for development only).

---

## API Reference (summary)

Base: `http://localhost:8080/api`

### Public endpoints

- `GET /animals`
  - Returns the base animal list and their stats.

- `GET /public-games`
  - List recent public games.

- `GET /leaderboard`
  - Top players by wins.

- `POST /auth/google/oauth2callback`
  - Body: `{ "code": "<google_auth_code>" }` — exchanges the code for
    user info and sets the session cookie (`q_session`). Returns user JSON.

> NOTE: image generation and asset endpoints require an authenticated
> session (see Protected endpoints). They are not public.

### Protected endpoints (require session cookie)

- `GET /assets/animals/<file>`
  - Serves the stored animal image (generates via OpenAI if missing).

- `GET /assets/hybrids/<key>.png`
  - Serves or generates a hybrid image. `key` is the canonical animal
    key (lowercase names joined with `_`, e.g. `lion_raven.png`).

- `GET /player-stats?email=<email>`
  - Returns aggregated stats for a player (or uses logged-in user's email).

- `POST /games`
  - Create a new game. Body: `{ "player_name": string, "player_uuid": string?, "player_email": string?, "name": string, "description": string, "private": boolean }`
  - Response: `{ "game_id": number, "join_code": string, "creator_uuid": string }`

- `POST /games/join`
  - Join by code. Body: `{ "join_code": string, "player_name": string, "player_uuid": string? }`
  - Response: `{ "game_id": number, "player_uuid": string }`

- `GET /games/:gameID`
  - Get game state.

- `POST /games/:gameID/create-hybrids`
  - Create the player's two hybrids. Body example:
    ```json
    {
      "player_uuid": "...",
      "hybrid1": { "animal_ids": [1,2], "selected_animal_id": 1 },
      "hybrid2": { "animal_ids": [3,4], "selected_animal_id": 4 }
    }
    ```
  - Rules: 2–3 animals per hybrid; selected animal must belong to the hybrid; animals cannot be reused across both hybrids for the same player.

- `POST /games/:gameID/start`
  - Starts the match. The server will generate AI hybrid names and images
    (background worker). This endpoint returns quickly with HTTP 202 while
    heavy work runs asynchronously.

- `POST /games/:gameID/action`
  - Submit a player's chosen action for the current round.
  - Body: `{ "player_uuid": string, "action_type": "basic_attack" | "defend" | "ability" | "rest", "animal_id"?: number }`
  - When both players submitted actions, the server resolves the round
    immediately and updates the game state.

- `POST /games/:gameID/leave` and `POST /games/:gameID/end`
  - Leave a waiting room or force-end a game (server enforces transitions).

See the handler source in `backend/internal/api/` for exact validation and
error responses.

---

## Image & Name Generation

- Hybrid names are generated using the OpenAI client and cached in the DB
  (`internal/hybridname`) to avoid duplicate calls.
- Images are generated via OpenAI (or returned from DB if already stored).
  Generation requests are deduplicated (singleflight) and may take up to
  ~90 seconds; the server returns `500` on OpenAI errors or timeouts.

---

## Combat Mechanics (concise)

- Planning & resolution
  - Each round both players choose actions simultaneously. When both
    actions are submitted the server resolves the round deterministically
    (priority by Agility, ties broken randomly).

- Available actions
  - `basic_attack`: costs 1 VIG (if VIG=0 damage is halved; minimum 1).
  - `defend`: costs 1 VIG and grants +50% Defense this round if VIG>0.
  - `ability`: each hybrid has exactly one selected animal ability chosen
    at creation; abilities consume ENE (energy) and a VIG cost derived from
    the base animal's config. If VIG is insufficient, the ability still
    happens but VIG is set to 0 and the hybrid becomes Vulnerable
    (receives +25% damage this round).
  - `rest`: no cost; restores +2 VIG (capped at base VIG) and +2 ENE.

- Energy (ENE)
  - Base ENE is the sum of animals' `energy` values and is clamped to
    the range [1,3] on hybrid creation. Each round active hybrids gain
    +1 ENE at the start of the round. Some abilities (e.g., Wolf) can add
    additional ENE.

- Vigor (VIG)
  - Each hybrid has a `base_vig` (initialized to 3 if not set) and a
    current VIG. Actions spend VIG as described above.

- Priority & special rules
  - Higher Agility acts earlier; Eagle's ability grants `priority_next_round`.
  - Attack vs Defense: damage = max(1, attack - defense) with modifiers
    (buffs/debuffs, VIG penalties, Vulnerable multiplier, and some
    abilities that ignore defense or add recoil).

- Fatigue
  - From Round 3 on, active hybrid Defense is reduced: R3 −1, R4 −2,
    R5+ −3 (defense clamped to >= 0).

- Defeat & substitution
  - When an active hybrid reaches 0 HP, the reserve (if any) becomes
    active with base stats. Win by defeating both opponent hybrids.

- Resignation / desertion
  - If a player leaves during a match (or explicitly ends the match), the
    server marks them as resigned and the opponent is recorded as the
    winner.

---

## Development notes & troubleshooting

- If the server fails on startup, check that `GOOGLE_CLIENT_ID`,
  `GOOGLE_CLIENT_SECRET`, `SESSION_SECRET` and `OPENAI_API_KEY` are set.
- The chimera configuration file `chimera_config.json` must include an
  `animal_list` array. The repo includes an example at
  `backend/chimera_config.json`.
- The backend logs informative messages to the console; watch for
  OpenAI errors (image/name generation) and missing env var fatal logs.

---

If you want, I can also:
- add a short example client script that demonstrates the auth + game
  creation flow, or
- generate a small OpenAPI spec from the route handlers.

---

## Docker Compose with Caddy (Let's Encrypt via Cloudflare)

This project includes a `docker-compose.yml` that can run the backend,
frontend and a fronting Caddy reverse-proxy that performs TLS termination
using Let's Encrypt via the Cloudflare DNS challenge.

Below are the practical details and requirements to run this compose
stack in a development or production-like environment.

### Prerequisites

- Docker and Docker Compose (or the `docker compose` plugin) installed.
- A domain managed in Cloudflare (the domain must exist in your Cloudflare
  account).
- A Cloudflare API token with permission to edit DNS records for the
  target zone (recommended: a token scoped to `Zone.DNS` with **Edit**
  permission, or an equivalent token for the zone). Do not use the
  global API key unless necessary.
- Port availability on the host: by default the compose maps host ports
  `8880 -> 80` and `8443 -> 443`. If you want to expose standard HTTP/HTTPS
  change the ports to `80:80` and `443:443` (requires root/admin on the host).

### Important files & paths

- `docker-compose.yml` — the compose file that runs `chimera-backend`,
  `chimera-frontend` and `caddy`.
- `infrastructure/caddy/Caddyfile` — the Caddy configuration used by the
  `caddy` service (proxy rules for `/api*` and `/auth*`, plus TLS config).
- Host bind mounts for Caddy data/config (where certificates live):
  - `./infrastructure/caddy/data` -> `/data` inside the container
  - `./infrastructure/caddy/config` -> `/config` inside the container

  These folders persist certificates, ACME account data and runtime
  configuration and are intentionally host-mounted so you can inspect
  or copy certificates.

### Environment variables (.env)

Create or update the repository root `.env` with the following values
(example):

```
# Domain and Cloudflare API token for ACME DNS challenge
DOMAIN=your.domain.example
CLOUDFLARE_API_TOKEN=<cloudflare-token-with-dns-edit>
CADDY_EMAIL=admin@your.domain.example

# App secrets/config
SESSION_SECRET=change-this-to-a-long-random-string
REACT_APP_GOOGLE_CLIENT_ID=...
REACT_APP_API_BASE_URL=/api
SESSION_SECURE_COOKIE=1

# Optional: if you run backend as non-root and want host UID mapping
LOCAL_UID=1000
LOCAL_GID=1000
```

Notes:
- `CLOUDFLARE_API_TOKEN` must have DNS edit rights for the zone. A token
  scoped only to the zone is best practice.
- `SESSION_SECURE_COOKIE=1` is recommended when serving over HTTPS so the
  backend sets secure cookies.

### Running the stack

1. Ensure `.env` is populated as above.
2. (Optional) Pull the Caddy image that already includes the Cloudflare
   DNS provider plugin:

   ```bash
   docker compose pull caddy
   ```

3. Start the services (builds backend/frontend images if needed):

   ```bash
   docker compose up -d --build
   ```

4. Watch Caddy logs to confirm ACME activity and certificate issuance:

   ```bash
   docker compose logs -f caddy
   ```

### Testing TLS locally (SNI)

Because certificates are issued for your real domain, testing via
`https://localhost` will fail. Use `curl --resolve` to force DNS while
keeping the correct SNI header. Replace `your.domain` below with
`$DOMAIN` from your `.env`:

```bash
curl -vk --resolve 'your.domain:8443:127.0.0.1' https://your.domain:8443/
```

If you map the service to `443:443` then omit the custom port:

```bash
curl -vk --resolve 'your.domain:443:127.0.0.1' https://your.domain/
```

If the Caddyfile uses the Let's Encrypt staging CA for testing, the
certificate will be untrusted by browsers — this is expected. Remove the
`ca https://acme-staging-v02.api.letsencrypt.org/directory` line from
`infrastructure/caddy/Caddyfile` to request production certificates.

### File locations for issued certificates

On the host (when using the default bind mounts) Caddy stores certs and
keys under:

```
infrastructure/caddy/data/caddy/certificates/<issuer>/<domain>/
```

Examples:

```
infrastructure/caddy/data/caddy/certificates/acme-staging-v02.api.letsencrypt.org-directory/chimera.ericogr.com.br/chimera.ericogr.com.br.key
infrastructure/caddy/data/caddy/certificates/acme-staging-v02.api.letsencrypt.org-directory/chimera.ericogr.com.br/chimera.ericogr.com.br.crt
```

Use `openssl x509 -in <certfile> -text -noout` to inspect certificate
contents.

### Permissions & common issues

- If Caddy logs show `permission denied` when writing under `/data`,
  ensure the host bind mount directories are owned by the same UID/GID
  that the Caddy process uses inside the container. Determine the UID/GID
  with:

  ```bash
  docker run --rm --entrypoint sh ghcr.io/caddybuilds/caddy-cloudflare:latest -c 'id -u && id -g'
  ```

  Then chown the host folders (example for UID/GID `1000:1000`):

  ```bash
  sudo chown -R 1000:1000 infrastructure/caddy/data infrastructure/caddy/config
  sudo chmod -R 750 infrastructure/caddy/data infrastructure/caddy/config
  ```

- If your host enforces SELinux, either add `:Z` to the bind mounts in
  `docker-compose.yml` or run `chcon` to give Docker the right labels.

### Troubleshooting notes

- Error `module not registered: dns.providers.cloudflare`: the running
  Caddy binary must include the Cloudflare DNS module. The compose in
  this repo uses an image that already bundles the provider. If you
  build a custom Caddy binary, include the Cloudflare module.
- ACME errors related to Cloudflare typically indicate an invalid token
  or insufficient token scope (must be able to edit DNS for the zone).
- If certificates are not issued, check `docker compose logs caddy` for
  the detailed ACME messages.

---

If you want, I can add a short `docs/` page with quick copy-paste
commands (for permission fixes, migration, and curl tests). Otherwise
this section should provide the necessary details to run the compose
stack with Caddy and Cloudflare DNS-based certificate issuance.
