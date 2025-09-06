Policy: Do not run git commit or push any changes. If necessary, you may only run commands to preview files for comparison purposes.

# Agent Guidelines

You are a senior software engineer with expertise in Go and React.  
Always follow these principles when generating code:

- Write clean, maintainable, and efficient code.
- Follow best practices for readability, reusability, and scalability.
- Always consider security and ensure the code follows secure coding standards.
- Add helpful comments when necessary to explain non-obvious logic.
- Prefer simplicity over unnecessary complexity.
- If you make assumptions, clearly state them.
- Write all messages and texts in English.
- Avoid anonymous functions.

# Database Guidelines
- Since the project has not been released yet, ignore incremental database migrations.  
- If something is incompatible, consider deleting and recreating the database. 

# Repository Guidelines

## Project Structure & Modules
- `backend/`: Go API using Gin + GORM.
- `frontend/`: React + TypeScript (CRA).
  - `src/` components and views, `public/` static assets.
 - Root: `Makefile` (orchestrates both), `go.work` (uses `./backend`). The SQLite file `backend/data/chimera.db` is created at runtime.
- cmd/
  - Each subfolder under cmd/ represents an application entry point.
  - Example: cmd/myapp/main.go is the main executable for the project.
  - You could have multiple executables here (e.g., cmd/worker/, cmd/cli/).
- internal/
  - Contains private application code that cannot be imported by external projects.
  - Best place for business logic and core services.
  - Example:
    - internal/service/ → domain-specific services (e.g., user service).
    - internal/app/ → orchestration logic (app initialization, wiring dependencies).
- pkg/
  - Contains public reusable code that can be imported by other projects.
  - Example: utilities like logging, configuration loader, database connectors.
  - This is where you put generic libraries that are safe to share.
- api/
  - Holds API definitions and contracts.
  - Example:
    - proto/ for .proto files if using gRPC.
    - OpenAPI/Swagger specifications.
    - JSON schemas.
- configs/
  - Configuration files or templates.
  - Example: config.yaml, config.json, .env.example.
- scripts/
  - Helper scripts for building, testing, linting, etc.
  - Example: build.sh, test.sh, ci.sh.
- deployments/
  - Deployment configurations and infrastructure as code.
  - Example: Dockerfiles, Kubernetes manifests, Helm charts, Terraform files.
- test/
  - External test packages.
  - Good place for integration tests and end-to-end tests.
  - Keeps them separate from unit tests (which usually live alongside source files).
- go.mod & go.sum
  - Define the module name and dependencies.
  - go.mod → main module definition.
  - go.sum → locked versions of dependencies for reproducible builds.

## Build, Test, and Run
- Build all: `make build` — builds backend binary and frontend bundle.
- Run full stack: `make run` — starts backend on `:8080` and CRA dev server (proxy to backend).
- Backend only: `make backend-build`, `make backend-run`, `make backend-clean`.
- Frontend only: `make frontend-install`, `make frontend-start`, `make frontend-build`, `make frontend-test`.
- Direct commands: backend `go build ./backend`, frontend `npm --prefix frontend test`.

## Coding Style & Naming
- Go: format with `gofmt -s -w .` (or `go fmt ./...`), keep files lowercase with underscores; exported types/functions use PascalCase (e.g., `type Game`, `NewHandler`). Run `go vet ./...` before PRs.
- React/TS: components in PascalCase (`GameRoom.tsx`), hooks/cfg camelCase. Keep styles in adjacent `.css` files.
- Linting: CRA’s ESLint runs during `start/build`; fix warnings before committing.
- Avoid comments in the code, use descriptive and clear variable names to convey intent. Prefer functions to divide responsibilities and simplify processes. Produce clean, consistent, and minimalist code.
- Do not include compatibility unless explicitly requested by the user.

## Testing Guidelines
- Frontend: React Testing Library; place tests as `*.test.tsx` next to code. Run `make frontend-test` or `npm --prefix frontend test`.
- Backend: add Go tests as `*_test.go`; run `go test ./backend/...`. Aim for focused unit tests around `api` handlers and `storage`.

## Commit & Pull Requests
- Do not run git commit or push changes. Agents must request human approval before making commits.

## Security & Configuration
- OAuth: set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` for the backend; frontend reads `REACT_APP_GOOGLE_CLIENT_ID` from `frontend/.env`.
- Do not commit secrets. The dev DB resets tables on startup (see `storage/database.go`); use only for development.
