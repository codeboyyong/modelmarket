---
name: database-check
description: Check the Model Market database from repo env files. Use when asked to inspect database status, count tables/users/models, or list model pricing from the configured PostgreSQL database.
---

# Database Check

Use this skill when the user asks to inspect the local/dev/qa/prod Model Market database, especially for:

- How many tables are in the database.
- How many users exist.
- How many models exist.
- What model prices are configured.

## Run

From the project root:

```sh
.codex/skills/database-check/scripts/check_database.sh dev
```

Replace `dev` with `qa`, `staging`, `prod`, or another environment name when needed.

## Environment Loading

The script reads environment values in this order, with later files overriding earlier files when variables are repeated:

1. `.env`
2. `.env.<env>`
3. `deploy/env/<env>.env`
4. `config/env/<env>.env`

It uses only the `MM_*` database variables:

- `MM_DATABASE_URL`
- `MM_DB_DRIVER`
- `MM_DB_HOST`
- `MM_DB_PORT`
- `MM_DB_NAME`
- `MM_DB_USER`
- `MM_DB_PASSWORD`
- `MM_DB_SSL_MODE`

`MM_DATABASE_URL` takes priority. If it is not set, the script builds the PostgreSQL connection from the individual `MM_DB_*` variables.

## Safety

The script runs read-only `SELECT` queries and does not print the database password. It requires `psql` and a reachable PostgreSQL database. PostgreSQL is the Phase 1 runtime database even though the schema avoids unnecessary PostgreSQL-only SQL where practical.
