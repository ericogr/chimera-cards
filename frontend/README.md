# Frontend (React + TypeScript)

The frontend is a Create React App (CRA) TypeScript single-page application
located in the `frontend/` folder. The dev server proxies API requests to
the backend so local development preserves same-origin cookies.

## Quick overview

- Dev server: `http://localhost:3000` (proxies `/api` to backend)
- Build: produces static assets served by `nginx` in Docker Compose

## Prerequisites

- Node.js & npm
- `make`

## Development

```bash
make frontend-install   # install npm deps
make frontend-start     # start CRA dev server
make frontend-build     # build production bundle
make frontend-test      # run tests
```

## Environment

Create `frontend/.env` with at least the Google client ID required by the
SPA:

```env
REACT_APP_GOOGLE_CLIENT_ID="..."
```

## Notes

- The CRA dev server proxies `/api` to the backend as configured in the
  project. If running frontend and backend separately, ensure the backend
  is reachable and env vars are set appropriately.

