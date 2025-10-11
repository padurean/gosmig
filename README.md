<img src="gosmig-banner.png" title="GoSMig" alt="GoSMig logo" />

[![Go Test](https://github.com/padurean/gosmig/actions/workflows/checks.yml/badge.svg)](https://github.com/padurean/gosmig/actions/workflows/checks.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/padurean/gosmig)](https://goreportcard.com/report/github.com/padurean/gosmig)
[![Go Reference](https://pkg.go.dev/badge/github.com/padurean/gosmig.svg)](https://pkg.go.dev/github.com/padurean/gosmig)
[![License: Unlicense](https://img.shields.io/badge/license-Unlicense-blue.svg)](http://unlicense.org/)


**GoSMig** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp; ☆⋆｡𖦹°‧★🛸

Simple, minimal SQL migrations written in Go.

Build your own migration CLI with it.

Standard library [**`database/sql`**](https://pkg.go.dev/database/sql) and [**`sqlx`**](https://github.com/jmoiron/sqlx) supported out of the box.

Can be used with any database library that implements the standard interfaces.

## Features

- [x] **Database Agnostic** - Works with any database that implements Go's standard `database/sql` interfaces
- [x] **Type-Safe** - Full Go generics support for compile-time type safety
- [x] **Flexible** - Supports both transactional and non-transactional migrations
- [x] **Simple** - Minimal API with clear semantics
- [x] **CLI-Ready** - Built-in command-line interface
- [x] **Rollback Support** - Safe migration rollbacks
- [x] **Status Tracking** - View migration status with paging support
- [x] **Tested** - Comprehensive test suite with PostgreSQL integration tests

## Installation

```bash
go get github.com/padurean/gosmig
```

## Quick Start

### Basic Example with database/sql

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
                    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN age INT`)
                    return err
                },
                Down: func(ctx context.Context, db *sql.DB) error {
                    _, err := db.ExecContext(ctx, `ALTER TABLE users DROP COLUMN age`)
                    return err
                },
            },
        },
    }

    // Create the migration tool
    migrate, err := gosmig.New(migrations, connectToDB, 10*time.Second)
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

### Using with sqlx

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
                    _, err := db.ExecContext(ctx, `ALTER TABLE products ADD COLUMN price DECIMAL(10,2)`)
                    return err
                },
                Down: func(ctx context.Context, db *sqlx.DB) error {
                    _, err := db.ExecContext(ctx, `ALTER TABLE products DROP COLUMN price`)
                    return err
                },
            },
        },
    }

    migrate, err := gosmig.New(migrations, connectToSQLXDB, 10*time.Second)
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

## CLI Usage

Once you've built your migration tool, use it from the command line:

```bash
# Apply all pending migrations
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up

# Apply only the next migration
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up-one

# Roll back the last migration
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" down

# Check migration status
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" status

# Get current database version
./your-migration-tool "postgres://user:pass@localhost:5432/dbname?sslmode=disable" version
```

## Commands

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
migrate, err := gosmig.New(migrations, connectToDB, 30*time.Second) // 30 second timeout
```

If you pass `0` or a negative duration, the default timeout of 10 seconds will be used.

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

## Supported Databases

gosmig works with any database that implements Go's standard `database/sql` interfaces. Tested with:

- PostgreSQL (with pgx driver)
- Should work with MySQL, SQLite, SQL Server, and others

## API Reference

### Core Types

```go
type Migration[TDBRow, TDBResult, TTX, TTXO, TDB] struct {
    Version    int
    UpDown     *UpDown[TDBRow, TDBResult, TTX]     // Transactional migration
    UpDownNoTX *UpDown[TDBRow, TDBResult, TDB]     // Non-transactional migration
}

type UpDown[TDBRow, TDBResult, TDBOrTX] struct {
    Up   func(ctx context.Context, dbOrTx TDBOrTX) error
    Down func(ctx context.Context, dbOrTx TDBOrTX) error
}
```

### Main Function

```go
func New[TDBRow, TDBResult, TTX, TTXO, TDB](
    migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
    connectToDB func(url string, timeout time.Duration) (TDB, error),
    timeout time.Duration,
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

### v1.0.0 (Current)

- Initial release
- Support for both transactional and non-transactional migrations
- CLI interface with up, up-one, down, status, and version commands
- Full Go generics support
- Support for database/sql and sqlx
- Comprehensive test suite
