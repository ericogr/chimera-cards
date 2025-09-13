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

## Versioning & Deployment

This project embeds build and VCS metadata into both backend and frontend
artifacts and publishes Docker images to Docker Hub. The strategy provides:

- A canonical, human-friendly semantic/tag version (from `git describe --tags`).
- VCS commit short hash and build timestamp for traceability.
- A `dirty` flag when working tree had uncommitted changes at build time.
- Version metadata exposed on the backend via `/api/version`; the frontend
  reads this endpoint at runtime to surface the current build/version.

CI (GitHub Actions) builds images and pushes them to Docker Hub with both
the tagged version and `latest` tags. See `.github/workflows/docker-publish.yml`.

Required GitHub repository secrets (set in Settings → Secrets):

- `DOCKERHUB_USERNAME` — Docker Hub username (used to authenticate push).
- `DOCKERHUB_TOKEN` — Docker Hub access token or password.
- `REACT_APP_GOOGLE_CLIENT_ID` — Google OAuth client id for frontend build (optional).
- `REACT_APP_API_BASE_URL` — API base URL for frontend runtime (optional).

How it works (summary):

1. CI computes `VERSION`, `COMMIT`, `BUILD_DATE`, `DIRTY` from the repo state.
2. The backend binary is built with `-ldflags` to inject these values and the
   final Docker image is labeled with the same metadata.
3. The frontend reads `/api/version` at runtime so the SPA can surface the
   current build/version provided by the backend.
4. The workflow pushes backend and frontend images to Docker Hub using the
   following canonical names:

   - `ericogr/chimera-cards-backend:<version>` and `ericogr/chimera-cards-backend:latest`
   - `ericogr/chimera-cards-frontend:<version>` and `ericogr/chimera-cards-frontend:latest`

Local build notes:

- Backend: `make -C backend build` will embed version metadata detected from
  the local git repository. You can override values by passing `VERSION=...`
  `COMMIT=...` `BUILD_DATE=...` `DIRTY=...` to `make`.
- Frontend: `make -C frontend build` will generate `public/version.json` and
  then run the production build.
