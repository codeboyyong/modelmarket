# Model Market

Model Market is a Phase 1 foundation for a unified AI model marketplace and API gateway.

The current implementation is a local developer scaffold:

- Go backend API
- Node.js + TypeScript frontend
- explicit PostgreSQL schema initialization script
- optional Redis readiness hook for future cache/rate-limit/session work
- repo-owned SQL dev test data
- Docker Compose stack
- basic API key flow
- mock chat completion endpoint
- admin/metadata/workbench UI shell

The product direction is documented in:

- [system_requirement.md](system_requirement.md)
- [system_design.md](system_design.md)
- [implementation_plan.md](implementation_plan.md)
- [docs/demo-accounts.md](docs/demo-accounts.md)
- [docs/docker.md](docs/docker.md)
- [docs/postgresql.md](docs/postgresql.md)
- [docs/security.md](docs/security.md)

## Requirements

For local checks without Docker:

- Go 1.25+
- Node.js 24+
- npm

For full local stack:

- Docker
- Docker Compose plugin or legacy `docker-compose`

On macOS, Docker Desktop or Colima both work. See [docs/docker.md](docs/docker.md).

For manual PostgreSQL setup, see [docs/postgresql.md](docs/postgresql.md).

## Quick Check

Run backend tests and frontend typecheck/build:

```sh
scripts/validate-local.sh
```

This does not require PostgreSQL, Redis, or Docker.

Expected backend output includes `ok` lines for tested packages. The `cmd/server` package may report `[no test files]` because it is only the executable entrypoint; core behavior is tested under `backend/internal/...`.

## Start With Docker Compose

Start the normal local stack:

```sh
scripts/dev-up.sh
```

Reset local containers and database volumes:

```sh
scripts/dev-reset.sh
```

The app should be available at:

- Frontend: http://localhost:3000
- Backend health: http://localhost:8080/healthz
- Backend readiness: http://localhost:8080/readyz

The backend does not create or mutate schema on startup. Initialize schema and load dev test data explicitly:

```sh
scripts/init_db.sh dev
scripts/populate_test_data.sh dev
```

Redis is optional in Phase 1. The default Docker Compose stack runs with PostgreSQL only. To start Redis as well for future cache/rate-limit experiments:

```sh
docker compose --profile redis up redis
```

## Environment

Copy the example environment file if you want to run services manually:

```sh
cp .env.example .env
```

Phase 1 uses mocked provider data and does not require real provider, OAuth, or payment credentials.

Environment-specific database files live in:

```text
config/env/dev.env
config/env/qa.env
config/env/prod.env
```

PostgreSQL is the implemented database driver for the application in Phase 1. The portable SQL files under `db/` avoid PostgreSQL-specific types and functions so the schema/test-data baseline is easier to adapt for another SQL database later.

## Initialize Database Schema and Test Data

Create tables:

```sh
scripts/init_db.sh dev
```

Populate deterministic test data:

```sh
scripts/populate_test_data.sh dev
```

The first argument is the environment name and maps to the matching env files.

Examples:

```sh
scripts/init_db.sh dev
scripts/populate_test_data.sh dev

scripts/init_db.sh qa
scripts/populate_test_data.sh qa
```

By default the scripts use:

- `db/init_db.sql`
- `db/populate_test_data.sql`

For `dev`, `test`, and `local`, `scripts/init_db.sh` resets the schema before recreating it. The test-data script clears existing dev test rows before inserting, so it can be rerun in a development database.

## Demo Accounts

After running `scripts/populate_test_data.sh dev`, these local demo accounts are available:

```text
System admin
Username: admin@example.com
Password: dev-password

Individual consumer
Username: developer@example.com
Password: dev-password

Corporate admin
Username: corp-admin@example.com
Password: dev-password
Company: Acme Creative Studio
```

Corporate admins can open the `Company Admin` view after login to review company members, shared credit usage, and model distribution.

For signup testing, choose `Corporate user` and enter this company name to attach the new user to the seeded company:

```text
Acme Creative Studio
```

## Manual Backend Run

If PostgreSQL is already running locally:

```sh
MM_APP_ENV=dev
cd backend
GOCACHE="$(pwd)/../.cache/go-build" \
GOMODCACHE="$(pwd)/../.cache/go-mod" \
go run ./cmd/server
```

Default backend config:

- `MM_APP_ENV=dev`
- `MM_DB_DRIVER=postgres`
- `MM_DB_HOST=localhost`
- `MM_DB_PORT=5432`
- `MM_DB_NAME=model_market`
- `MM_DB_USER=model_market`
- `MM_DB_PASSWORD=model_market`
- `MM_DB_SSL_MODE=disable`
- `REDIS_ENABLED=false`
- `REDIS_ADDR=localhost:6379`
- `DEV_MODE=true`

You can also set `MM_DATABASE_URL` directly. For qa/prod, use SSL, for example:

```sh
MM_APP_ENV=prod
MM_DATABASE_URL='postgres://user:password@db.example.com:5432/model_market?sslmode=require'
```

If `MM_DATABASE_URL` is not set, the backend builds a PostgreSQL connection string from the `MM_DB_*` fields. `MM_DB_SSL_MODE` defaults to `disable` for `dev/test/local` and `require` for `qa/staging/prod`.

## Database Initialization

The generic SQL schema is in:

```text
db/init_db.sql
```

The test/dev data SQL is in:

```text
db/populate_test_data.sql
```

Initialize a database for an environment:

```sh
scripts/init_db.sh dev
```

Populate test data:

```sh
scripts/populate_test_data.sh dev
```

The scripts accept an environment name as the first argument:

```sh
scripts/init_db.sh qa
scripts/populate_test_data.sh qa
```

They load environment variables from these files when present:

```text
.env
.env.<environment>
deploy/env/<environment>.env
```

PostgreSQL is the implemented runtime driver in Phase 1. The SQL files avoid PostgreSQL-only column types so the schema can be adapted to other databases later, but the shell scripts currently execute through `psql` for `MM_DB_DRIVER=postgres`.

## Manual Frontend Run

```sh
cd frontend
npm install
npm run build
npm start
```

Open:

```text
http://localhost:3000
```

## API Smoke Test

Health:

```sh
curl http://localhost:8080/healthz
```

Dev summary:

```sh
curl http://localhost:8080/api/v1/dev/summary
```

List models:

```sh
curl http://localhost:8080/api/v1/models
```

Get projects:

```sh
curl http://localhost:8080/api/v1/projects
```

Dev test data includes this API key for local smoke tests:

```text
mk_dev_test_key
```

You can also create a new API key. Replace `PROJECT_ID` with a project ID from the previous response:

```sh
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"PROJECT_ID","name":"Local dev key"}'
```

Call mock chat with the dev test data key:

```sh
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer mk_dev_test_key' \
  -d '{
    "model": "mock-chat-default",
    "messages": [
      {
        "role": "user",
        "content": "Explain this platform in one sentence."
      }
    ]
  }'
```

## Repository Layout

```text
backend/              Go backend API
frontend/             Node.js + TypeScript frontend
db/                   generic SQL schema and dev test data
scripts/              local dev/check/build helpers
docs/                 developer documentation
deploy/               future deployment assets
docker-compose.yml    local full-stack development
```

## Docker Image Builds

Build local images for the current architecture:

```sh
scripts/docker-build-local.sh
```

Build multi-architecture images for `linux/amd64` and `linux/arm64` and push to a registry:

```sh
IMAGE_REGISTRY=ghcr.io \
IMAGE_NAMESPACE=your-org/model-market \
IMAGE_TAG=phase1 \
scripts/docker-build-multiarch.sh
```

More details are in [docs/docker.md](docs/docker.md).

## Current Phase

Phase 1 is implemented:

- runnable skeleton
- schema init script
- dev test data
- frontend shell
- backend health/readiness
- API key basics
- mock chat endpoint
- Docker/Compose configuration
- CI skeleton

Next phase:

- core product loop
- metadata manager editing
- model profile resolution
- real provider adapter
- credit reservation/capture
- persisted Workbench chat


### Example web site

* https://www.atlascloud.ai/
* https://openrouter.ai/
* https://siliconflow.cn/  



  price_type : input/output
  asset_type: token/image/video/audio
  unit_type : 1k_token/480P/720P/1080P/2k/4K/
  unit:
  unit_price: 
 