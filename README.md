# [**`GoSMig`**](https://github.com/ogg/gosmig) Examples

This directory contains two example implementations demonstrating how to use [**`GoSMig`**](https://github.com/ogg/gosmig) with different database libraries.

## Overview

Both examples implement the same migration workflow but with different database drivers:

- **`with_std_sql/`** - Uses Go's standard library [**`database/sql`**](https://pkg.go.dev/database/sql)
- **`with_sqlx/`** - Uses the [**`sqlx`**](https://github.com/jmoiron/sqlx) library

## Example Structure

Each example follows the same structure:

```text
with_.../
├── go.mod              # Module dependencies
├── cmd/
│   └── main.go        # CLI entry point
└── migrations/
    ├── migrations.go  # Migration list and type definitions
    ├── 00001.go       # First migration (transactional)
    ├── 00002.go       # Second migration (non-transactional)
    └── 00003.go       # Third migration (transactional)
```

## Example 1: Using Standard Library (`with_std_sql/`)

This example uses Go's built-in [**`database/sql`**](https://pkg.go.dev/database/sql) package for database interactions.

### Running the Standard Library Example

```bash
cd with_std_sql
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up      # Apply all migrations
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up-one  # Apply only the next migration
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable status  # Check migration status
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable version # Show current DB version
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable down    # Rollback last migration
```

## Example 2: Using sqlx (`with_sqlx/`)

This example uses the [**`sqlx`**](https://github.com/jmoiron/sqlx) library, which extends [**`database/sql`**](https://pkg.go.dev/database/sql) with additional features like named queries and struct scanning.

### Running the sqlx Example

```bash
cd with_sqlx
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up      # Apply all migrations
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up-one  # Apply only the next migration
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable status  # Check migration status
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable version # Show current DB version
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable down    # Rollback last migration
```

## Comparison

| Feature | `with_std_sql` | `with_sqlx` |
|---------|----------------|-------------|
| **Dependencies** | Standard library only | Requires sqlx package |
| **DB Connection** | `*sql.DB` | `*sqlx.DB` |
| **Transaction Type** | `*sql.Tx` | `*sql.Tx` (standard) |
| **Non-TX Migrations** | `*sql.DB` | `*sqlx.DB` |
| **Type Aliases** | Uses `gosmig.MigrationSQL` | Defines custom `MigrationSQLX` |
| **Complexity** | Simpler | Slightly more complex |
| **Features** | Basic SQL execution | Extended query features |

## Database Setup

Both examples expect a PostgreSQL database. Use the provided `db_init.sql` to create the necessary user and database:

```sql
CREATE USER gosmig WITH ENCRYPTED PASSWORD 'gosmig';
CREATE DATABASE gosmig OWNER gosmig;
```

Pass the database URL as the 1st CLI argument:

```bash
go run ./cmd postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable up
```

## CLI Commands

Both examples support the same CLI commands:

- **`up`** - Apply all pending migrations
- **`up-one`** - Apply only the next migration
- **`down`** - Rollback the last migration
- **`status`** - Show status of all migrations (applied/pending)
- **`version`** - Show current database version

## Key Takeaways

1. **Type safety**: Both examples demonstrate proper type definitions for [**`GoSMig`**](https://github.com/ogg/gosmig)'s generic interfaces
2. **Transactional vs Non-transactional**: Use `UpDown` for transactions, `UpDownNoTX` when transactions aren't needed or supported
3. **Flexibility**: The same migration pattern works with both standard library's [**`database/sql`**](https://pkg.go.dev/database/sql) and the [**`sqlx`**](https://github.com/jmoiron/sqlx) library
4. **Real-world ready**: Both examples are production-ready patterns you can adapt for your projects

## Creating Your Own CLI

To create your own migration CLI:

1. Define your migration types (like `MigrationSQL` or `MigrationSQLX`)
2. Create individual migration files with unique versions
3. Collect all migrations in a slice
4. Implement a database connection function
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
