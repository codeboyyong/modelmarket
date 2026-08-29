# PostgreSQL Backup and Restore

These scripts create and restore a complete logical backup of one Model Market
PostgreSQL database. They use PostgreSQL's native custom archive format, which
includes the database schema, tables, sequences, constraints, indexes, and all
table data.

The scripts load connection settings from the repository's existing environment
files through `scripts/load-db-env.sh`:

```text
config/env/dev.env
config/env/qa.env
config/env/prod.env
```

An explicitly configured `MM_DATABASE_URL` takes precedence over the individual
`MM_DB_HOST`, `MM_DB_PORT`, `MM_DB_NAME`, `MM_DB_USER`, `MM_DB_PASSWORD`, and
`MM_DB_SSL_MODE` settings.

## Requirements

Install PostgreSQL client tools containing compatible versions of `pg_dump`
and `pg_restore`.

Check that they are available:

```sh
pg_dump --version
pg_restore --version
```

The `pg_dump` client should generally be the same major version as the server
or newer. A client older than the PostgreSQL server may refuse to create the
backup.

## Create a Backup

From the repository root, create a development backup:

```sh
scripts/op/backup_db.sh dev
```

The default archive is written beneath `backups/` with a UTC timestamp:

```text
backups/model_market-dev-20260829T010000Z.dump
```

Choose an explicit destination when desired:

```sh
scripts/op/backup_db.sh prod /secure/backups/model-market-prod.dump
```

The backup script:

1. Loads the selected environment configuration.
2. Runs `pg_dump` in compressed custom format.
3. Excludes original ownership and ACL statements so the archive can be
   restored by the configured application/database user.
4. Writes to a temporary `.partial` file.
5. Validates the archive with `pg_restore --list`.
6. Renames it to the requested destination only after successful validation.

It refuses to overwrite an existing archive.

## Inspect a Backup

List an archive's contents without restoring it:

```sh
pg_restore --list backups/model_market-dev-20260829T010000Z.dump
```

## Restore a Backup

> **Warning:** Restore is destructive. It drops existing objects represented
> by the archive and replaces their schema and data inside the target database.

Restore into the development database:

```sh
scripts/op/restore_db.sh dev backups/model_market-dev-20260829T010000Z.dump
```

The script displays the resolved target and requires you to type the target
database name before it proceeds.

For a reviewed non-interactive operation, pass `--yes`:

```sh
scripts/op/restore_db.sh dev backups/model_market-dev-20260829T010000Z.dump --yes
```

Use `--yes` carefully, especially in automation. Always verify the environment
name, target host, target database, and archive path first.

The restore uses:

- `--clean --if-exists` to replace existing archived objects;
- `--single-transaction` so an error rolls back the database changes;
- `--exit-on-error` to stop immediately after a restore failure;
- `--no-owner --no-privileges` for portability between environments.

## Suggested Recovery Procedure

1. Stop application traffic or put the application into maintenance mode.
2. Confirm that the archive belongs to the intended environment.
3. Create a safety backup of the current target database.
4. Run the restore without `--yes` and verify the displayed target.
5. Start or restart the backend.
6. Check `/healthz` and `/readyz`.
7. Run application smoke tests and inspect important row counts.
8. Retain the pre-restore safety backup until recovery is accepted.

Example:

```sh
scripts/op/backup_db.sh prod /secure/backups/pre-restore-safety.dump
scripts/op/restore_db.sh prod /secure/backups/approved-production.dump
```

## Scope and Limitations

These are single-database logical backups. They do not include PostgreSQL
cluster-wide roles, tablespaces, server configuration, WAL files, or other
databases in the same PostgreSQL cluster. Production disaster recovery should
also include infrastructure-managed snapshots or physical backups and a tested
point-in-time-recovery strategy.

The local filesystem used by the MVP mock object-storage service is not stored
inside PostgreSQL and is therefore not included. Back up `OBJECT_STORAGE_DIR`
separately if those generated or uploaded demo files must be recovered.

## Security

- Do not commit backup archives; they may contain credentials, personal data,
  prompts, payment metadata, and other sensitive information.
- Store production archives encrypted and access-controlled.
- Do not print or copy database URLs containing passwords into logs.
- Apply an appropriate retention policy and securely delete expired backups.
- Regularly test restoration instead of assuming an archive is recoverable.
