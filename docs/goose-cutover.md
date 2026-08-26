# Cutover: golang-migrate → goose

This document covers the one-time migration of an existing database that has
migrations applied by golang-migrate (the old runner) to the goose-based
runner.

## Background

- **Before:** golang-migrate tracked applied migrations in a `schema_migrations`
  table (one row per applied version).
- **After:** goose tracks applied migrations in the table configured by
  `DB_POSTGRES_MIGRATION_TABLE` (default `oil_migrations`, schema:
  `id`, `version_id`, `is_applied`, `tstamp`).

If you skip the cutover and deploy the goose version against a database that
only has `schema_migrations`, goose will try to re-run migrations that were
already applied (they are not `IF NOT EXISTS`-safe for columns in all cases),
which fails or duplicates work.

## Procedure

Run the following **once, per environment, before deploying the goose-based
version of the binary** (dev, staging, production — in that order).

### 1. Back up the database

```sql
-- e.g. with pg_dump
pg_dump -h <host> -U <user> <db> > backup_$(date +%F).sql
```

### 2. Create the goose version table

```sql
CREATE TABLE IF NOT EXISTS oil_migrations (
    id serial PRIMARY KEY,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp DEFAULT now()
);
```

Replace `oil_migrations` with your `DB_POSTGRES_MIGRATION_TABLE` value if
overridden.

### 3. Seed it from golang-migrate's state

golang-migrate's `schema_migrations` stores one row per applied version.
Copy **every** applied version (not just the latest — goose validates that all
versions up to the current one are present):

```sql
INSERT INTO oil_migrations (version_id, is_applied)
SELECT version, TRUE
FROM schema_migrations
ORDER BY version;
```

### 4. Verify

```bash
# should print the same version as the last schema_migrations row, e.g. 3
go run cmd/migrate/main.go up
# goose should report: no migrations to run. current version: 3
```

You can also confirm with:

```sql
SELECT version_id, is_applied FROM oil_migrations ORDER BY version_id;
```

### 5. Deploy the goose-based binary

Deploy the new binary and run `make migrate.up` (or the equivalent deploy
step). New migrations after this point apply normally on top of the seeded
version.

## Notes

- The old `schema_migrations` table can be dropped once the cutover is
  confirmed working: `DROP TABLE schema_migrations;`
- Do **not** run the seed twice — it will insert duplicate `version_id` rows.
  If you need to redo it, `TRUNCATE oil_migrations;` and re-seed.
- Test the cutover against a **copy** of a production-like database first
  (e.g. a fresh restore of the staging dump) before running it against a live
  environment.
