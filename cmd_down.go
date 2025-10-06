package gosmig

import (
	"context"
	"fmt"
	"io"
	"time"
)

func runCmdDown[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](

	ctx context.Context,
	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
	db TDB,
	output io.Writer,
	timeout time.Duration,
) error {

	sortMigrationsDesc(migrations)

	dbVersion, err := getDBVersion(ctx, db, timeout)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version > dbVersion {
			continue
		}

		if migration.UpDown != nil {
			err = executeInTx(
				ctx, db, migrateDown(migration.Version, migration.UpDown.Down, timeout), timeout)
			if err != nil {
				return fmt.Errorf("execute in TX: %w", err)
			}
		} else {
			err = executeNoTx(
				ctx, db, migrateDown(migration.Version, migration.UpDownNoTX.Down, timeout), timeout)
			if err != nil {
				return fmt.Errorf("execute without TX: %w", err)
			}
		}

		_, _ = fmt.Fprintf(
			output, "[x]-->[ ] Rolled back migration version %d\n", migration.Version)
		return nil
	}

	_, _ = fmt.Fprintln(output, "No migrations to roll back")

	return nil
}

func migrateDown[TDBRow DBRow, TDBResult DBResult, TDBOrTX DBOrTX[TDBRow, TDBResult]](
	version int,
	down func(ctx context.Context, dbOrTX TDBOrTX) error,
	timeout time.Duration,
) func(context.Context, TDBOrTX) error {

	return func(ctx context.Context, dbOrTX TDBOrTX) error {
		dbVersion, err := getDBVersion(ctx, dbOrTX, timeout)
		if err != nil {
			return err
		}

		if version > dbVersion {
			return fmt.Errorf(
				"%w: migration version %d > current DB version %d",
				errDBVersionChangedDown, version, dbVersion)
		}

		migCtx, cancelMig := context.WithTimeout(ctx, timeout)
		defer cancelMig()
		if err := down(migCtx, dbOrTX); err != nil {
			return fmt.Errorf(
				"failed to apply migration.down version %d: %w", version, err)
		}

		if err := deleteDBVersion(ctx, dbOrTX, version, timeout); err != nil {
			return err
		}

		return nil
	}
}
