# Backend (Go)

This folder contains the Go backend server (Gin + GORM) used by Chimera
Cards.

## Overview

- Main executable: [`cmd/chimera-cards`](./cmd/chimera-cards)
- Development DB (SQLite): [`data/chimera.db`](./data/chimera.db)

## Prerequisites

- Go 1.20+ installed
- `make` available

## Development commands

Run these from the repository root:

```bash
make backend-build   # build the backend binary
make backend-run     # run the backend (development)
make backend-test    # run backend tests
```

## Environment variables

Set these variables for local development (example `backend/.env`):

| Variable | Required | Description |
|---|:---:|---|
| `GOOGLE_CLIENT_ID` | yes | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | yes | Google OAuth client secret |
| `SESSION_SECRET` | yes | Long random string for sessions |
| `SESSION_SECURE_COOKIE` | no | `0` for local HTTP, `1` for HTTPS |
| `OPENAI_API_KEY` | yes | OpenAI API key for name/image generation |
| `CHIMERA_CONFIG` | no | Path to `chimera_config.json` (defaults to `./chimera_config.json`) |

Example `.env` snippet:

```env
GOOGLE_CLIENT_ID="..."
GOOGLE_CLIENT_SECRET="..."
SESSION_SECRET="A_LONG_RANDOM_STRING"
SESSION_SECURE_COOKIE="0"
OPENAI_API_KEY="sk-..."
CHIMERA_CONFIG="./chimera_config.json"
```

## Notes

- The backend validates that required env vars are present on startup and
  will exit if any are missing.
- The development DB is SQLite and intended only for local development; for
  production use, migrate to a managed database and update `internal/storage`.

## Configuration file (`chimera_config.json`)

The server reads `chimera_config.json` (path may be set via the `CHIMERA_CONFIG`
env var). This file defines the entities and several templates used at runtime.

- `entity_list`: array of entity objects with fields such as `name`,
  `hit_points`, `attack`, `defense`, `agility`, `energy`, `vigor_cost`,
  `skill_name`, `skill_cost`, `skill_description`, `skill_key` and a
  `skill_effect` object describing the mechanical behaviour of the ability.
- `single_image_prompt`: prompt template used when generating a single-entity
  portrait (used at startup and by the entity asset endpoint). Use the token
  `{{entities}}` where the entity name should be substituted.
- `hybrid_image_prompt`: prompt template used when generating hybrid images
  (used by hybrid image generation). Use `{{entities}}` to inject the comma-
  separated entity names in the prompt.

Example: the single-entity prompt is used when seeding/creating entity
portraits at startup; the hybrid prompt is used when generating final hybrid
images. Keep the prompts in `chimera_config.json` to make image styling
adjustable without code changes.
