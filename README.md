# [**`GoSMig`**](https://github.com/ogg/gosmig) Examples

This directory contains three example implementations demonstrating how to use [**`GoSMig`**](https://github.com/ogg/gosmig) with different database drivers and concurrency patterns.

## Overview

Each example implements the same migration workflow but with different database drivers or connection guarantees:

- **`with_std_sql/`** - Uses Go's standard library [**`database/sql`**](https://pkg.go.dev/database/sql)
- **`with_sqlx/`** - Uses the [**`sqlx`**](https://github.com/jmoiron/sqlx) library
- **`with_pg_advisory_lock/`** - Uses the standard library but enforces single-run safety with a PostgreSQL [advisory lock](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS)

## Example Structure

Each example follows the same structure (the third example adds a small `LockedDB` wrapper that lives alongside the migrations):

```text
with_.../
├── go.mod              # Module dependencies
├── cmd/
│   └── main.go        # CLI entry point
└── migrations/
    ├── migrations.go  # Migration list and type definitions
    ├── 00001.go       # First migration (transactional)
    ├── 00002.go       # Second migration (non-transactional, uses PostgreSQL CREATE INDEX CONCURRENTLY)
    ├── 00003.go       # Third migration (transactional)
    └── locked_db.go   # (Advisory-lock example only) Lock-aware DB wrapper
```

## Example 1: Using Standard Library ([`with_std_sql/`](./with_std_sql/))

This example uses Go's built-in [**`database/sql`**](https://pkg.go.dev/database/sql) package for database interactions.

See [Running the Examples](#running-the-examples) for the shared command set.

## Example 2: Using sqlx ([`with_sqlx/`](./with_sqlx/))

This example uses the [**`sqlx`**](https://github.com/jmoiron/sqlx) library, which extends [**`database/sql`**](https://pkg.go.dev/database/sql) with additional features like named queries and struct scanning.

See [Running the Examples](#running-the-examples) for the shared command set.

## Example 3: Using a PostgreSQL Advisory Lock ([`with_pg_advisory_lock/`](./with_pg_advisory_lock/))

This example extends the standard-library approach with a lock-aware connection wrapper. By acquiring a PostgreSQL advisory lock before running, it prevents separate processes from applying migrations simultaneously.

### Why advisory locking?

- Ensures only one migration runner operates at a time, even across multiple hosts
- Uses `pg_try_advisory_lock(hashtext('gosmig_advisory_lock_example'))` under a configurable timeout
- Releases the lock when the CLI exits to avoid leaving the database in a locked state

See [Running the Examples](#running-the-examples) for the shared command set.

## Running the Examples

Pick the variant you want to try (`with_std_sql`, `with_sqlx`, or `with_pg_advisory_lock`) and run the CLI the same way:

```bash
# Replace <example_dir> with the example you want to run
cd <example_dir>

# Apply all migrations
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up

# Apply only the next migration
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up-one

# Check migration status
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable status

# Show current DB version
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable version

# Rollback last migration
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable down
```

## Comparison

| Feature | `with_std_sql` | `with_sqlx` | `with_pg_advisory_lock` |
|---------|----------------|-------------|------------------------|
| **Dependencies** | Standard library only | Adds the sqlx package | Standard library + tiny wrapper |
| **DB Connection** | `*sql.DB` | `*sqlx.DB` | `*LockedDB` (wraps `*sql.DB`) |
| **Transaction Type** | `*sql.Tx` | `*sql.Tx` | `*sql.Tx` |
| **Non-TX Migrations** | `*sql.DB` | `*sqlx.DB` | `*LockedDB` keeps the lock held |
| **Type Aliases** | Reuses `gosmig.MigrationSQL` | Defines custom `MigrationSQLX` | Reuses `gosmig.Migration` with lock-aware types |
| **Concurrency Control** | None | None | PostgreSQL advisory lock |
| **Use When** | You want the basics | You like sqlx ergonomics | You need single-run safety |

## Database Setup

All examples expect a PostgreSQL database. Use the provided `db_init.sql` to create the necessary user and database:

```sql
CREATE USER gosmig WITH ENCRYPTED PASSWORD 'gosmig';
CREATE DATABASE gosmig OWNER gosmig;
```

Pass the database URL as the 1st CLI argument:

```bash
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up
```

## CLI Commands

All examples support the same CLI commands:

- **`up`** - Apply all pending migrations
- **`up-one`** - Apply only the next migration
- **`down`** - Rollback the last migration
- **`status`** - Show status of all migrations (applied/pending)
- **`version`** - Show current database version

## Key Takeaways

1. **Type safety**: All examples demonstrate proper type definitions for [**`GoSMig`**](https://github.com/ogg/gosmig)'s generic interfaces
2. **Transactional vs Non-transactional**: Use `UpDown` for transactions, `UpDownNoTX` when transactions aren't needed or supported (such as when using PostgreSQL's `CONCURRENTLY`, which must run outside a transaction)
3. **Flexibility**: The same migration pattern works with both the standard library's [**`database/sql`**](https://pkg.go.dev/database/sql) and the [**`sqlx`**](https://github.com/jmoiron/sqlx) library
4. **Concurrency control**: Wrap your connection (like the advisory-lock sample) when you need to guard against concurrent runners
5. **Real-world ready**: Each example is a production-ready pattern you can adapt for your projects

## Creating Your Own CLI

To create your own migration CLI:

1. Define your migration types (like `MigrationSQL` or `MigrationSQLX`)
2. Create individual migration files with unique versions
3. Collect all migrations in a slice
4. Implement a database connection function (wrap it if you need cross-process locking or other guarantees)
5. Call `gosmig.New()` with your migrations and connection function
6. Call the returned migration function to handle CLI arguments

```go
func main() {
    migrate, err := gosmig.New(migrations.Migrations, connectToDB, nil)
    if err != nil {
        log.Fatalf("Failed to create migration tool: %v", err)
    }
    migrate() // Parses os.Args and executes the appropriate command
}
```

That's it! [**`GoSMig`**](https://github.com/ogg/gosmig) handles all the CLI parsing, command execution, and output formatting.
