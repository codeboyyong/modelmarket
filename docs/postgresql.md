# PostgreSQL Setup

This project uses PostgreSQL as the implemented Phase 1 database.

Default dev configuration:

```sh
MM_APP_ENV=dev
MM_DB_DRIVER=postgres
MM_DB_HOST=localhost
MM_DB_PORT=5432
MM_DB_NAME=model_market
MM_DB_USER=model_market
MM_DB_PASSWORD=model_market
MM_DB_SSL_MODE=disable
```

The backend builds this connection URL from the `MM_DB_*` values:

```text
postgres://model_market:model_market@localhost:5432/model_market?sslmode=disable
```

## Option 1: Use Docker Compose

This is the simplest local path if Docker is available.

```sh
scripts/dev-up.sh
```

Docker Compose starts PostgreSQL with:

```sh
POSTGRES_DB=model_market
POSTGRES_USER=model_market
POSTGRES_PASSWORD=model_market
```

Then open:

```text
Frontend: http://localhost:3000
Backend:  http://localhost:8080
```

## Option 2: Install on macOS With Homebrew

Install PostgreSQL:

```sh
brew install postgresql@16
```

Start PostgreSQL:

```sh
brew services start postgresql@16
```

If `psql` is not on your path, add it:

```sh
echo 'export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

For Intel Macs, Homebrew may use `/usr/local` instead:

```sh
echo 'export PATH="/usr/local/opt/postgresql@16/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Verify:

```sh
psql --version
```

Create the dev user and database:

```sh
createuser model_market
createdb model_market -O model_market
psql -d postgres -c "alter user model_market with password 'model_market';"
```

Initialize schema and test data:

```sh
scripts/init_db.sh dev
scripts/populate_test_data.sh dev
```

## Option 3: Install on Ubuntu/Debian

Install PostgreSQL:

```sh
sudo apt update
sudo apt install -y postgresql postgresql-client
```

Start and enable service:

```sh
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

Create the dev user and database:

```sh
sudo -u postgres psql -c "create user model_market with password 'model_market';"
sudo -u postgres psql -c "create database model_market owner model_market;"
```

Initialize schema and test data:

```sh
scripts/init_db.sh dev
scripts/populate_test_data.sh dev
```

## Environment Files

The DB scripts load environment values from these files when present:

```text
.env
.env.<environment>
<!-- deploy/env/<environment>.env -->
```

Example dev env file:

```sh
MM_APP_ENV=dev
DEV_MODE=true
MM_DB_DRIVER=postgres
MM_DB_HOST=localhost
MM_DB_PORT=5432
MM_DB_NAME=model_market
MM_DB_USER=model_market
MM_DB_PASSWORD=model_market
MM_DB_SSL_MODE=disable
REDIS_ENABLED=false
REDIS_ADDR=localhost:6379
```

The repo includes:

```text
deploy/env/dev.env
deploy/env/qa.env.example
deploy/env/prod.env.example
```

## SSL Modes

For local development:

```sh
MM_DB_SSL_MODE=disable
```

For QA/staging/production:

```sh
MM_DB_SSL_MODE=require
```

For strict production verification when certificates and hostnames are configured:

```sh
MM_DB_SSL_MODE=verify-full
```

## Manual Connection Test

Connect with `psql`:

```sh
PGPASSWORD=model_market psql \
  -h localhost \
  -p 5432 \
  -U model_market \
  -d model_market
```

List tables:

```sql
\dt
```

Check seeded data:

```sql
select email, name from users;
select slug, name from models;
select paid_credits, promotional_credits from wallets;
```

## Run Backend Manually

If PostgreSQL is already running:

```sh
cd backend
GOCACHE="$(pwd)/../.cache/go-build" \
GOMODCACHE="$(pwd)/../.cache/go-mod" \
go run ./cmd/server
```

Backend health:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

## Common Issues

### `psql: command not found`

Install PostgreSQL client tools, or add PostgreSQL binaries to your `PATH`.

### `password authentication failed`

Reset the dev password:

```sh
psql -d postgres -c "alter user model_market with password 'model_market';"
```

On Linux:

```sh
sudo -u postgres psql -c "alter user model_market with password 'model_market';"
```

### `database "model_market" does not exist`

Create it:

```sh
createdb model_market -O model_market
```

Or on Linux:

```sh
sudo -u postgres psql -c "create database model_market owner model_market;"
```

### `connection refused`

PostgreSQL is not running or is listening on a different host/port.

Check service:

```sh
brew services list
```

Or on Linux:

```sh
sudo systemctl status postgresql
```

### SSL Error in Local Dev

Use:

```sh
MM_DB_SSL_MODE=disable
```

Local PostgreSQL usually does not require SSL.
