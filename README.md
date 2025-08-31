# Quimera Cards

Quimera Cards is a web-based tactical turn-based combat game where each
player builds two hybrids (each composed of 2–3 animals) and battles in
simultaneous-planning rounds. The backend is written in Go (Gin + GORM)
and the frontend is a React + TypeScript single-page app.

This README documents the current project layout, how to run the app,
the public API (including image generation endpoints) and a concise
summary of the combat mechanics.

---

## Highlights
- Backend refactor: single executable at `backend/cmd/quimera-cards` and
  modularized `internal/` packages (api, service, engine, storage, hybrid
  image/name helpers, OpenAI client, etc.).
- OpenAI integration: AI-generated hybrid names and images (DALL·E-like
  image generation). Images are cached in the database.
- Session-based Google login: `/api/auth/google/oauth2callback` exchanges
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
  - `quimera.db` — runtime SQLite DB created in `backend/` (development)

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
`backend/quimera.db` (this behavior is intended for development only).

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

- `GET /animals/image?ids=1` or `ids=1,2`
  - Returns a PNG (256x256) for one animal or a merged hybrid image for
    2–3 animals. The `ids` query parameter is a comma-separated list of
    animal IDs. The server uses OpenAI to generate images when not present
    in the DB. Response Content-Type: `image/png`.

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

