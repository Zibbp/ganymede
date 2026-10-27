# Development database

Rebuild and reopen the devcontainer to start PostgreSQL alongside it. The
development container receives the database connection settings automatically:
host `db`, port `5432`, user and password `ganymede`, and database `ganymede`.
These settings take precedence over values in the local `.env` when using the
development commands.

PostgreSQL 18 stores its data in the Compose `postgres-18-data` named volume, which
survives container rebuilds and normal shutdowns. Removing that volume (for
example, with `docker compose down -v`) permanently deletes the development
database. PostgreSQL initialization settings apply only to a new volume.

If you used the previous PostgreSQL 14 configuration, its `postgres-data` volume
is retained but is not used by PostgreSQL 18. Migrating its contents requires a
PostgreSQL major-version upgrade, such as a dump and restore; rebuilding alone
starts with a new database.
