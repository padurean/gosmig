package gosmig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	toolName = "gosmig"

	cmdUp      = "up"
	cmdUpOne   = "up-one"
	cmdDown    = "down"
	cmdStatus  = "status"
	cmdVersion = "version"
)

var allCommands = []string{
	cmdUp,
	cmdUpOne,
	cmdDown,
	cmdStatus,
	cmdVersion,
}

// New creates a new gosmig instance. It returns a function that runs the migration tool
// when called, and an error if the provided migrations are invalid.
//
// The returned function should be called to execute the migration commands.
// It handles command-line arguments (gosmig <db_url> and <command>), connects to
// the database using the provided connectToDB function, and performs the
// requested migration command (up, up-one, down, status, version).
//
// It works with any database that implements the DB interface, allowing for flexibility
// in database choice. Stdlib SQL and sqlx are supported out of the box (see the examples
// below and the tests from gosmig_integration_test.go).
//
// Parameters:
//   - migrations: A slice of Migration objects defining the database migrations.
//   - connectToDB: A function that takes a database URL and a timeout duration,
//     and returns a connected database instance or an error.
//   - config: A Config struct specifying the configuration options, such as timeout duration
//     for database operations. If nil, default configuration is used.
//
// Returns:
//   - A function that executes the migration tool when called.
//   - An error if the migrations are invalid or if there are issues during setup.
//
// Example usage:
//
// Using with the standard library's database/sql:
//
//	func main() {
//		migrations := []gosmig.Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sql.DB]{
//			{Version: 1,
//				UpDown: &gosmig.UpDown[*sql.Row, sql.Result, *sql.Tx]{
//					Up: func(ctx context.Context, tx *sql.Tx) error {
//						_, err := tx.ExecContext(ctx, `CREATE TABLE example (id SERIAL PRIMARY KEY, name TEXT)`)
//						return err
//					},
//					Down: func(ctx context.Context, tx *sql.Tx) error {
//						_, err := tx.ExecContext(ctx, `DROP TABLE example`)
//						return err
//					},
//				},
//			},
//			{Version: 2,
//				UpDownNoTX: &gosmig.UpDown[*sql.Row, sql.Result, *sql.DB]{
//					Up: func(ctx context.Context, db *sql.DB) error {
//						_, err := db.ExecContext(ctx, `ALTER TABLE example ADD COLUMN age INT`)
//						return err
//					},
//					Down: func(ctx context.Context, db *sql.DB) error {
//						_, err := db.ExecContext(ctx, `ALTER TABLE example DROP COLUMN age`)
//						return err
//					},
//				},
//			},
//		}
//
//		goSMig, err := gosmig.New(migrations, connectToDBFunc, nil)
//		if err != nil {
//		    log.Fatalf("Failed to create gosmig instance: %v", err)
//		}
//		goSMig() // Run the migration tool
//	}
//
// TIP: There are also type aliases defined for convenience:
// MigrationSQL, UpDownSQL, UpDownNoTXSQL
//
// Using with sqlx, just change the *sql.DB to *sqlx.DB in the Migration type parameters.
// TIP: One can define their own type aliases for convenience, e.g.:
//
//	type MigrationSQLX  = Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sqlx.DB]
//	type UpDownNoTXSQLX = UpDown[*sql.Row, sql.Result, *sqlx.DB]
//
// NOTE: There's no need to define an alias for UpDown, as it always uses *sql.Tx,
// which is the same for both stdlib SQL and sqlx. So the provided UpDownSQL type
// alias works for both.
//
// Once the gosmig instance is created, you can run the resulting function
// to execute the migration commands. The tool accepts command-line arguments
// for the database URL and the desired command (up, up-one, down, status, version).
// CLI example:
//
//	$ go run main.go "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up
//
// Note: Ensure that the database URL is correctly formatted for your database driver and
// that the necessary driver is imported and initialized in your main package.
func New[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](

	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
	connectToDB func(url string, timeout time.Duration) (TDB, error),
	config *Config,
) (func(), error) { // coverage-ignore

	getArgs := func() []string {
		return os.Args[1:]
	}

	return newGosmig(migrations, connectToDB, config, getArgs, os.Exit, os.Stdout, os.Stderr)
}

func newGosmig[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](

	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
	connectToDB func(url string, timeout time.Duration) (TDB, error),
	config *Config,
	getArgs func() []string,
	osExit func(int),
	out, errOut io.Writer,
) (func(), error) {

	if len(migrations) == 0 {
		return nil, fmt.Errorf("no migrations provided")
	}

	if connectToDB == nil {
		return nil, fmt.Errorf("connectToDB function is nil")
	}

	if config == nil {
		config = DefaultConfig()
	} else {
		config.ensureDefaults()
	}

	if getArgs == nil {
		return nil, fmt.Errorf("getArgs function is nil")
	}

	if osExit == nil {
		return nil, fmt.Errorf("osExit function is nil")
	}

	if out == nil {
		return nil, fmt.Errorf("out writer is nil")
	}

	if errOut == nil {
		return nil, fmt.Errorf("errOut writer is nil")
	}

	if err := validateMigrations(migrations); err != nil {
		return nil, err
	}

	return func() {
		var errExitCode int

		url, command, err := parseArgs(getArgs())
		if err != nil {
			errExit(errExitCode+1, err, errOut, osExit)
			return
		}

		ctx := context.Background()

		db, err := connectToDB(url, config.Timeout)
		if err != nil {
			errExit(errExitCode+2, err, errOut, osExit)
			return
		}
		defer func() {
			if err := db.Close(); err != nil {
				errExit(errExitCode+3, err, errOut, osExit)
				return
			}
		}()

		if err := createMigrationsTableIfNotExists(ctx, db, config.Timeout); err != nil {
			errExit(errExitCode+4, err, errOut, osExit)
			return
		}

		switch command {
		case cmdUp:
			if err := runCmdUp(ctx, migrations, db, out, 0, config.Timeout); err != nil {
				errExit(errExitCode+5, err, errOut, osExit)
				return
			}
		case cmdUpOne:
			if err := runCmdUp(ctx, migrations, db, out, 1, config.Timeout); err != nil {
				errExit(errExitCode+6, err, errOut, osExit)
				return
			}
		case cmdDown:
			if err := runCmdDown(ctx, migrations, db, out, config.Timeout); err != nil {
				errExit(errExitCode+7, err, errOut, osExit)
				return
			}
		case cmdStatus:
			if err := runCmdStatus(ctx, migrations, db, out, config.Timeout); err != nil {
				errExit(errExitCode+8, err, errOut, osExit)
				return
			}
		case cmdVersion:
			if err := runCmdVersion(ctx, db, out, config.Timeout); err != nil {
				errExit(errExitCode+9, err, errOut, osExit)
				return
			}
		}
	}, nil
}

func parseArgs(args []string) (string, string, error) {
	if len(args) != 2 {
		return "", "", errors.New("wrong number of arguments")
	}

	url := args[0]
	command := args[1]
	if !slices.Contains(allCommands, command) {
		return "", "", fmt.Errorf("unknown command: %q", command)
	}

	return url, command, nil
}

func usage() string {
	return fmt.Sprintf(
		"Usage: %s <db_url> <%s>", toolName, strings.Join(allCommands, "|"))
}

func errExit(errCode int, err error, output io.Writer, osExit func(int)) {
	_, _ = fmt.Fprintf(output, "%s\n%v\n", usage(), err)
	osExit(errCode)
}
