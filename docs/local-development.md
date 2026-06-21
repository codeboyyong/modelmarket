# Local Development From Source

This workflow runs PostgreSQL in one standalone Docker container and runs the Go backend and Node.js frontend directly from source. It does not use Docker Compose.

## Requirements

- Docker Desktop or another Docker runtime
- Go 1.25+
- Node.js 24+
- npm
- PostgreSQL client tools (`psql` and `pg_isready`)

Run all commands from the repository root unless a step says otherwise.

## 1. Start PostgreSQL

Create the standalone database container the first time:

```sh
docker run -d \
  --name model-market-dev-postgres \
  -e POSTGRES_DB=model_market \
  -e POSTGRES_USER=model_market \
  -e POSTGRES_PASSWORD=model_market \
  -p 5432:5432 \
  -v yong-zhao-model-market_postgres-data:/var/lib/postgresql/data \
  postgres:16-alpine
```

The volume name above reuses the database volume created by this repository's Compose project. On later runs, start the existing container instead:

```sh
docker start model-market-dev-postgres
```

Verify that PostgreSQL is ready:

```sh
pg_isready -h localhost -p 5432 -U model_market -d model_market
```

## 2. Initialize Development Data

For a new or disposable development database, initialize the schema and load the seeded demo data:

```sh
scripts/init_db.sh dev
scripts/populate_test_data.sh dev
```

`scripts/init_db.sh dev` drops and recreates the development schema. Do not run it when you need to preserve current local data.
It also resets every seeded account password to `dev-password`, including passwords changed through the UI.

## 3. Start the Backend

Open a new terminal:

```sh
cd backend
set -a
[ ! -f ../.env ] || . ../.env
set +a
export MM_APP_ENV=dev DEV_MODE=true REDIS_ENABLED=false
export GOCACHE="$(pwd)/../.cache/go-build"
export GOMODCACHE="$(pwd)/../.cache/go-mod"
go run ./cmd/server
```

The backend listens on http://localhost:8080.

## 4. Start the Frontend

Open another terminal. Install dependencies and build once before starting the server:

```sh
cd frontend
npm install
npm run build
npm run dev
```

The frontend listens on http://localhost:3000.

## 5. Rebuild TypeScript While Editing

The frontend server serves `frontend/dist`; `npm run dev` does not compile TypeScript. Open another terminal to rebuild `src/main.ts` when it changes:

```sh
cd frontend
npm exec tsc -- --watch --preserveWatchOutput
```

Refresh the browser after a rebuild. Changes to `public/index.html` or `public/styles.css` are not copied by the TypeScript watcher; run `npm run build` again after editing those files.

Go recompiles when `go run` starts, but it does not automatically restart after source changes. Stop the backend with `Ctrl+C` and rerun the backend command after editing Go code.

## 6. Verify the Environment

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl -I http://localhost:3000
```

Then open http://localhost:3000. Seeded accounts are listed in [demo-accounts.md](demo-accounts.md).

To run the backend smoke path:

```sh
scripts/demo-smoke.sh dev
```

## Stop and Restart

Stop the backend, frontend, and TypeScript watcher with `Ctrl+C` in their terminals. Stop PostgreSQL without deleting its data:

```sh
docker stop model-market-dev-postgres
```

Restart PostgreSQL later with:

```sh
docker start model-market-dev-postgres
```

To permanently remove the standalone container, stop it first and then run:

```sh
docker rm model-market-dev-postgres
```

Removing the container does not remove the named PostgreSQL volume.
