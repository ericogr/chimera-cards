# Chimera Cards

Chimera Cards is a web-based tactical turn-based combat game where each
player builds two hybrids (each composed of 2–3 entities) and battles in
simultaneous-planning rounds. The backend is written in Go (Gin + GORM)
and the frontend is a React + TypeScript single-page app.

This README contains common, cross-cutting information and links to the
module-level documentation for backend, frontend and infrastructure.

## Quick links

- Backend details: [`backend/README.md`](backend/README.md)
- Frontend details: [`frontend/README.md`](frontend/README.md)
- Infrastructure (bootstrap + Terraform): [`infrastructure/README.md`](infrastructure/README.md)
- Game mechanics: [`game.md`](game.md)

## Quick start (development)

1. Install prerequisites: Go, Node.js + npm, and `make`.
2. Install frontend deps: `make frontend-install` (see [`frontend/README.md`](frontend/README.md)).
3. Configure backend env vars (see [`backend/README.md`](backend/README.md)).
4. Run both services:

```bash
make run
```

## Production (Docker Compose)

Build and run with Docker Compose (example):

```bash
docker compose build
docker compose up -d
```

## Further details

See the module READMEs for detailed, focused documentation:

- [`backend/README.md`](backend/README.md) — backend setup, env vars, tests and build instructions.
- [`frontend/README.md`](frontend/README.md) — frontend dev server, build and test commands.
- [`infrastructure/README.md`](infrastructure/README.md) — bootstrap helper and Terraform workflow.
- [`game.md`](game.md) — full game mechanics documentation.
