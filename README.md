<img src="graphics/gosmig-banner.png" title="GoSMig" alt="GoSMig logo" />

[![Coverage](https://img.shields.io/badge/coverage-100%25-limegreen?style=flat&labelColor=black&logo=go&logoColor=white&logoSize=auto)](https://github.com/padurean/gosmig/actions/workflows/checks.yml)
[![Go Test](https://github.com/padurean/gosmig/actions/workflows/checks.yml/badge.svg)](https://github.com/padurean/gosmig/actions/workflows/checks.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/padurean/gosmig)](https://goreportcard.com/report/github.com/padurean/gosmig)
[![Go Reference](https://pkg.go.dev/badge/github.com/padurean/gosmig.svg)](https://pkg.go.dev/github.com/padurean/gosmig)
[![License: Unlicense](https://img.shields.io/badge/license-Unlicense-blue.svg)](http://unlicense.org/)

**GoSMig** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp; ☆⋆｡𖦹°‧★🛸

Simple, minimal SQL migrations written in Go.

Build your own migration CLI with it.

Standard library [**`database/sql`**](https://pkg.go.dev/database/sql) and [**`sqlx`**](https://github.com/jmoiron/sqlx) supported out of the box.

See the [examples](https://github.com/padurean/gosmig/tree/examples) for usage with both.

Can be used with any database library that implements the standard interfaces.

**GoSMig 💫 in action:**
<img src="https://raw.githubusercontent.com/padurean/gosmig/877ff597533e76a80deb88dfacbdd18a8a4b5e43/gosmig-demo.svg" title="GoSMig demo" alt="GoSMig demo" />

## Features

- [x] **Database Agnostic** - Works with any database that implements a subset of Go's standard `database/sql` interfaces (see the  [Core Types](#core-types) section lower for more details)
- [x] **Type-Safe** - Full Go generics support for compile-time type safety
- [x] **Flexible** - Supports both transactional and non-transactional migrations
- [x] **Simple** - Minimal API with clear semantics
- [x] **CLI-Ready** - No actual CLI is provided, but a built-in command-line interface handler makes it easy to build your own CLI tool
- [x] **Timeouts** - Configurable operation timeouts
- [x] **Robust Error Handling** - Validation, version conflict detection, transaction safety, and clear error messages
- [x] **Rollback Support** - Safe migration rollbacks
- [x] **Status Tracking** - View migration status with paging support
- [x] **Tested** - Comprehensive test suite with PostgreSQL integration tests
- [x] **Zero Dependencies** - No external dependencies, only the Go standard library (and [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) for pager support - i.e. for pagination - in the `status` command output)

**ℹ️ NOTEs**:

- You will need a database driver (e.g., [**`pgx`**](https://github.com/jackc/pgx) for PostgreSQL) and optionally [**`sqlx`**](https://github.com/jmoiron/sqlx).
- These are not dependencies of gosmig itself: while [**`pgx`**](https://github.com/jackc/pgx) and [**`sqlx`**](https://github.com/jmoiron/sqlx) show up in [go.mod](go.mod), they are used only in the examples and tests - gosmig does not actually depend on them.

## Installation

```bash
go get github.com/padurean/gosmig
```

## Quick Start

### Example with [**`database/sql`**](https://pkg.go.dev/database/sql)

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "time"

    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/padurean/gosmig"
)

func main() {
    // Define your migrations
    migrations := []gosmig.MigrationSQL{
        {
            Version: 1,
            UpDown: &gosmig.UpDownSQL{
                Up: func(ctx context.Context, tx *sql.Tx) error {
                    _, err := tx.ExecContext(ctx, `
                        CREATE TABLE users (
                            id SERIAL PRIMARY KEY,
                            name TEXT NOT NULL,
                            email TEXT UNIQUE NOT NULL,
                            created_at TIMESTAMPTZ DEFAULT NOW()
                        )`)
                    return err
                },
                Down: func(ctx context.Context, tx *sql.Tx) error {
                    _, err := tx.ExecContext(ctx, `DROP TABLE users`)
                    return err
                },
            },
        },
        {
            Version: 2,
            UpDownNoTX: &gosmig.UpDownNoTXSQL{
                Up: func(ctx context.Context, db *sql.DB) error {
                    _, err := db.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_users_created_at ON users (created_at)`)
                    return err
                },
                Down: func(ctx context.Context, db *sql.DB) error {
                    _, err := db.ExecContext(ctx, `DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at`)
                    return err
                },
            },
        },
    }

    // Create the migration tool
    migrate, err := gosmig.New(migrations, connectToDB, nil)
    if err != nil {
        log.Fatalf("Failed to create migration tool: %v", err)
    }

    // Run migrations - handles CLI arguments automatically
    migrate()
}

func connectToDB(url string, timeout time.Duration) (*sql.DB, error) {
    db, err := sql.Open("pgx", url)
    if err != nil {
        return nil, err
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, err
    }

    return db, nil
}
```

### Example with [**`sqlx`**](https://github.com/jmoiron/sqlx)

Very similar with the [**`database/sql`**](https://pkg.go.dev/database/sql) example above,
but with the following changes:

- define 2 type aliases:
  - `MigrationSQLX  = gosmig.Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sqlx.DB]`
  - `UpDownNoTXSQLX = gosmig.UpDown[*sql.Row, sql.Result, *sqlx.DB]`
- replace `sql.DB` with `sqlx.DB` and `sql.Open` with `sqlx.Open`

Full example:

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "time"

    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/jmoiron/sqlx"
    "github.com/padurean/gosmig"
)

func main() {
    // Define type aliases for convenience
    type (
        MigrationSQLX  = gosmig.Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sqlx.DB]
        UpDownNoTXSQLX = gosmig.UpDown[*sql.Row, sql.Result, *sqlx.DB]
    )

    migrations := []MigrationSQLX{
        {
            Version: 1,
            UpDown: &gosmig.UpDownSQL{ // Note: UpDownSQL works for both stdlib and sqlx
                Up: func(ctx context.Context, tx *sql.Tx) error {
                    _, err := tx.ExecContext(ctx, `CREATE TABLE products (id SERIAL PRIMARY KEY, name TEXT)`)
                    return err
                },
                Down: func(ctx context.Context, tx *sql.Tx) error {
                    _, err := tx.ExecContext(ctx, `DROP TABLE products`)
                    return err
                },
            },
        },
        {
            Version: 2,
            UpDownNoTX: &UpDownNoTXSQLX{
                Up: func(ctx context.Context, db *sqlx.DB) error {
                    _, err := db.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_users_created_at ON users (created_at)`)
                    return err
                },
                Down: func(ctx context.Context, db *sqlx.DB) error {
                    _, err := db.ExecContext(ctx, `DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at`)
                    return err
                },
            },
        },
    }

    migrate, err := gosmig.New(migrations, connectToSQLXDB, nil)
    if err != nil {
        log.Fatalf("Failed to create migration tool: %v", err)
    }

    migrate()
}

func connectToSQLXDB(url string, timeout time.Duration) (*sqlx.DB, error) {
    db, err := sqlx.Open("pgx", url)
    if err != nil {
        return nil, err
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, err
    }

    return db, nil
}
```

💡 Check the [examples](https://github.com/padurean/gosmig/tree/examples) branch for complete, runnable examples.

## CLI Usage

Once you've built your migration tool, use it from the command line:

### Migrate Up

```console
# Apply all pending migrations
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up

[x] Applied migration version 1
[x] Applied migration version 2
[x] Applied migration version 3
3 migration(s) applied
```

### Migrate Up One

```console
# Apply only the next migration
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up-one

[x] Applied migration version 4
1 migration(s) applied
```

### Migrate Down (One)

```console
# Roll back the last migration
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" down

[x]-->[ ] Rolled back migration version 4
```

### Check Migration Status

```console
# Check migration status
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" status

VERSION    STATUS
3          [ ] PENDING
2          [x] APPLIED
1          [x] APPLIED
```

### Get Current Database Version

```console
# Get current database version
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" version

Current database version:
5
```

## Commands Summary

| Command | Description |
|---------|-------------|
| `up` | Apply all pending migrations |
| `up-one` | Apply only the next pending migration |
| `down` | Roll back the most recent migration |
| `status` | Show the status of all migrations (uses pager for long lists) |
| `version` | Show the current database version |

## Migration Types

### Transactional Migrations (`UpDown`)

Use `UpDown` for migrations that should run within a database transaction. This is the recommended approach for most migrations as it ensures atomicity.

```go
{
    Version: 1,
    UpDown: &gosmig.UpDownSQL{
        Up: func(ctx context.Context, tx *sql.Tx) error {
            // Your migration code here
            return nil
        },
        Down: func(ctx context.Context, tx *sql.Tx) error {
            // Your rollback code here
            return nil
        },
    },
}
```

### Non-Transactional Migrations (`UpDownNoTX`)

Use `UpDownNoTX` for migrations that cannot or should not run in a transaction, such as:

- Index creation with `CONCURRENTLY` in PostgreSQL
- Operations that require multiple transactions
- DDL operations that don't support transactions in some databases

```go
{
    Version: 2,
    UpDownNoTX: &gosmig.UpDownNoTXSQL{
        Up: func(ctx context.Context, db *sql.DB) error {
            // Your non-transactional migration code here
            return nil
        },
        Down: func(ctx context.Context, db *sql.DB) error {
            // Your non-transactional rollback code here
            return nil
        },
    },
}
```

## Configuration

### Timeout Configuration

Configure operation timeouts when creating the migration tool:

```go
migrate, err := gosmig.New(migrations, connectToDB, &gosmig.Config{Timeout: 30 * time.Second})
```

If you pass **`0`** or a **negative** duration, the default timeout of **10 seconds** will be used.

### Database Connection

The `connectToDB` function should establish a connection and verify it's working:

```go
func connectToDB(url string, timeout time.Duration) (*sql.DB, error) {
    db, err := sql.Open("pgx", url)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return db, nil
}
```

### Coordinating Concurrent Runs

Running multiple migration processes at the same time can lead to conflicting writes.
Use a database-level advisory lock (or the closest equivalent) to ensure only one
gosmig instance runs at a time:

- **PostgreSQL**: call `SELECT pg_try_advisory_lock(...)` before starting migrations and
    `SELECT pg_advisory_unlock(...)` afterwards. Choose a consistent 64-bit key so all
    processes compete for the same lock.
- **MySQL / MariaDB**: wrap migration runs with `SELECT GET_LOCK('gosmig', timeout)`
    and `SELECT RELEASE_LOCK('gosmig')` using a shared lock name.
- **SQL Server**: use `sp_getapplock` / `sp_releaseapplock` with a well-known
    resource name.
- **SQLite**: serialize runs with a file lock (one process) or by executing
    `BEGIN EXCLUSIVE` on a dedicated coordination table before applying migrations.

For an example for PostgreSQL's advisory locks, see the [with_pg_advisory_lock example](https://github.com/padurean/gosmig/tree/examples/with_pg_advisory_lock) from the [**`examples`** branch](https://github.com/padurean/gosmig/tree/examples).

This pattern keeps migrations simple while preventing concurrent runs from stepping on each other.

## Type Aliases

For convenience, gosmig provides type aliases for common use cases:

```go
// For database/sql
type MigrationSQL  = Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sql.DB]
type UpDownSQL     = UpDown[*sql.Row, sql.Result, *sql.Tx]
type UpDownNoTXSQL = UpDown[*sql.Row, sql.Result, *sql.DB]

// Define your own for sqlx
type MigrationSQLX  = Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sqlx.DB]
type UpDownNoTXSQLX = UpDown[*sql.Row, sql.Result, *sqlx.DB]
// Note: UpDownSQL works for both database/sql and sqlx
```

## Migration Table

gosmig automatically creates a `gosmig` table to track applied migrations:

```sql
CREATE TABLE gosmig (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Error Handling

gosmig provides robust error handling:

- **Validation**: Migrations are validated at startup
- **Version Conflicts**: Prevents applying migrations if database version changes during execution
- **Transaction Safety**: Automatic rollback on errors in transactional migrations
- **Clear Error Messages**: Descriptive error messages with context

## Testing

### Running Tests

```bash
# Run tests with local PostgreSQL
make test

# Run tests with Docker PostgreSQL
make test-docker

# Clean up Docker containers
# NOTE: this doesn't need to be run as `make test-docker` already does it - it
# is provided here "just in case".
make test-docker-down
```

### Test Requirements

- PostgreSQL database (local or Docker)
- Go 1.25+

## Development

### Prerequisites

- Go 1.25+
- golangci-lint for linting
- govulncheck for vulnerability scanning

### Development Commands

```bash
# Lint code
make lint

# Check for vulnerabilities
make vulncheck

# Build (includes linting and vulnerability checks)
make build

# Build only (skip checks)
make build-only

# Run tests
make test

# Test with Docker
make test-docker
```

See [Makefile](Makefile) for all available development commands.

## Supported Databases

gosmig works with any database that implements Go's standard `database/sql` interfaces. Tested with:

- [PostgreSQL](https://www.postgresql.org)
- Should work with [MySQL](https://www.mysql.com), [SQLite](https://www.sqlite.org), [SQL Server](https://www.microsoft.com/en-us/sql-server), and others

## API Reference

### Core Types

```go
type (
    // Interfaces for database operations (subset of database/sql).
    // database/sql and sqlx satisfy these interfaces out of the box.
    // It should be possible to use any database library (wrapper) that satisfies them.

    DBRow interface {
        Scan(dest ...any) error
        Err() error
    }

    DBResult interface {
        LastInsertId() (int64, error)
        RowsAffected() (int64, error)
    }

    DBOrTX[TDBRow DBRow, TDBResult DBResult] interface {
        QueryRowContext(context.Context, string, ...any) TDBRow
        ExecContext(context.Context, string, ...any) (TDBResult, error)
    }

    TXOptions interface{}

    TX[TDBRow DBRow, TDBResult DBResult] interface {
        QueryRowContext(context.Context, string, ...any) TDBRow
        ExecContext(context.Context, string, ...any) (TDBResult, error)
        Commit() error
        Rollback() error
    }

    DB[TDBRow DBRow, TDBResult DBResult, TTX TX[TDBRow, TDBResult], TTXO TXOptions] interface {
        QueryRowContext(context.Context, string, ...any) TDBRow
        ExecContext(context.Context, string, ...any) (TDBResult, error)
        BeginTx(context.Context, TTXO) (TTX, error)
        Close() error
    }

    // Migration and related types

    UpDown[TDBRow DBRow, TDBResult DBResult, TDBOrTX DBOrTX[TDBRow, TDBResult]] struct {
        Up   func(ctx context.Context, tx TDBOrTX) error
        Down func(ctx context.Context, tx TDBOrTX) error
    }

    Migration[TDBRow DBRow, TDBResult DBResult, TTX TX[TDBRow, TDBResult], TTXO TXOptions, TDB DB[TDBRow, TDBResult, TTX, TTXO]] struct {
        Version    int
        UpDown     *UpDown[TDBRow, TDBResult, TTX]
        UpDownNoTX *UpDown[TDBRow, TDBResult, TDB]
    }

    MigrationSQL  = Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sql.DB]
    UpDownSQL     = UpDown[*sql.Row, sql.Result, *sql.Tx]   // Transactional migration
    UpDownNoTXSQL = UpDown[*sql.Row, sql.Result, *sql.DB]   // Non-transactional migration
)
```

### Main Function

```go
func New[TDBRow, TDBResult, TTX, TTXO, TDB](
    migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
    connectToDB func(url string, timeout time.Duration) (TDB, error),
    config *Config,
) (func(), error)
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is released into the public domain under the [Unlicense](http://unlicense.org/). See the [LICENSE](LICENSE) file for details.

## Changelog

### v0.0.1 (Current)

- Initial release
- Support for both transactional and non-transactional migrations
- CLI interface with `up`, `up-one`, `down`, `status`, and `version` commands
- Full Go generics support
- Support for [**`database/sql`**](https://pkg.go.dev/database/sql) and [**`sqlx`**](https://github.com/jmoiron/sqlx) out of the box
- Other database libraries supported via [interfaces](#core-types)
- Comprehensive test suite
