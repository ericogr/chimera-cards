# Backend (Go)

This folder contains the Go backend server (Gin + GORM) used by Quimera
Cards.

## Overview

- Main executable: [`cmd/quimera-cards`](./cmd/quimera-cards)
- Development DB (SQLite): [`data/quimera.db`](./data/quimera.db)

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

