package gosmig

import (
	"context"
	"fmt"
	"io"
	"time"
)

func runCmdUp[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](

	ctx context.Context,
	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
	db TDB,
	output io.Writer,
	limit int,
	timeout time.Duration,
) error {

	sortMigrationsAsc(migrations)

	dbVersion, err := getDBVersion(ctx, db, timeout)
	if err != nil {
		return err
	}

	var nbAppliedMigrations int

	for _, migration := range migrations {
		if migration.Version <= dbVersion {
			continue
		}

		if migration.UpDown != nil {
			err = executeInTx(
				ctx, db, migrateUp(migration.Version, migration.UpDown.Up, timeout), timeout)
			if err != nil {
				return fmt.Errorf("execute in TX: %w", err)
			}
		} else {
			err = executeNoTx(
				ctx, db, migrateUp(migration.Version, migration.UpDownNoTX.Up, timeout), timeout)
			if err != nil {
				return fmt.Errorf("execute without TX: %w", err)
			}
		}

		_, _ = fmt.Fprintf(
			output, "[x] Applied migration version %d\n", migration.Version)

		nbAppliedMigrations++
		if limit > 0 && nbAppliedMigrations == limit {
			break
		}
	}

	if nbAppliedMigrations == 0 {
		_, _ = fmt.Fprintln(output, "No migrations to apply")
		return nil
	}

	_, _ = fmt.Fprintf(output, "%d migration(s) applied\n", nbAppliedMigrations)

	return nil
}

func migrateUp[TDBRow DBRow, TDBResult DBResult, TDBOrTX DBOrTX[TDBRow, TDBResult]](
	version int,
	up func(ctx context.Context, dbOrTX TDBOrTX) error,
	timeout time.Duration,
) func(context.Context, TDBOrTX) error {

	return func(ctx context.Context, dbOrTX TDBOrTX) error {
		dbVersion, err := getDBVersion(ctx, dbOrTX, timeout)
		if err != nil {
			return err
		}

		if version <= dbVersion {
			return fmt.Errorf(
				"%w: migration version %d <= current DB version %d",
				errDBVersionChangedUp, version, dbVersion)
		}

		migCtx, cancelMig := context.WithTimeout(ctx, timeout)
		defer cancelMig()
		if err := up(migCtx, dbOrTX); err != nil {
			return fmt.Errorf(
				"failed to apply migration.up version %d: %w", version, err)
		}

		if err := insertDBVersion(ctx, dbOrTX, version, timeout); err != nil {
			return err
		}

		return nil
	}
}
