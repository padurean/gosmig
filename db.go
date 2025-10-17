package gosmig

import (
	"context"
	"fmt"
	"time"
)

type (
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
)

const (
	migrationsTableName = "gosmig"

	// SQL statements
	createMigsTblSQL = `CREATE TABLE IF NOT EXISTS ` + migrationsTableName +
		` (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`
	selectDBVersionSQL  = "SELECT COALESCE(MAX(version), 0) FROM " + migrationsTableName
	insertMigVersionSQL = "INSERT INTO " + migrationsTableName + " (version) VALUES ($1)"
	deleteMigVersionSQL = "DELETE FROM " + migrationsTableName + " WHERE version = $1"
)

func createMigrationsTableIfNotExists[TDBRow DBRow, TDBResult DBResult](
	ctx context.Context,
	dbOrTX DBOrTX[TDBRow, TDBResult],
	timeout time.Duration,
) error {

	ctxCreateMigsTbl, cancelCreateMigsTable := context.WithTimeout(ctx, timeout)
	defer cancelCreateMigsTable()

	_, err := dbOrTX.ExecContext(ctxCreateMigsTbl, createMigsTblSQL)
	if err != nil {
		return fmt.Errorf("failed to create migrations table if not exists: %w", err)
	}

	return nil
}

func getDBVersion[TDBRow DBRow, TDBResult DBResult](
	ctx context.Context,
	dbOrTX DBOrTX[TDBRow, TDBResult],
	timeout time.Duration,
) (int, error) {

	ctxGetDBVersion, cancelGetDBVersion := context.WithTimeout(ctx, timeout)
	defer cancelGetDBVersion()
	var dbVersion int
	err := dbOrTX.QueryRowContext(ctxGetDBVersion, selectDBVersionSQL).
		Scan(&dbVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get current DB version: %w", err)
	}
	return dbVersion, nil
}

func insertDBVersion[TDBRow DBRow, TDBResult DBResult](
	ctx context.Context,
	dbOrTX DBOrTX[TDBRow, TDBResult],
	version int,
	timeout time.Duration,
) error {

	versionCtx, cancelVersion := context.WithTimeout(ctx, timeout)
	defer cancelVersion()
	_, err := dbOrTX.ExecContext(versionCtx, insertMigVersionSQL, version)
	if err != nil {
		return fmt.Errorf(
			"failed to insert migration version %d into migrations table: %w",
			version, err)
	}

	return nil
}

func deleteDBVersion[TDBRow DBRow, TDBResult DBResult](
	ctx context.Context,
	dbOrTX DBOrTX[TDBRow, TDBResult],
	version int,
	timeout time.Duration,
) error {

	versionCtx, cancelVersion := context.WithTimeout(ctx, timeout)
	defer cancelVersion()
	_, err := dbOrTX.ExecContext(versionCtx, deleteMigVersionSQL, version)
	if err != nil {
		return fmt.Errorf(
			"failed to delete migration version %d from migrations table: %w",
			version, err)
	}

	return nil
}

func executeInTx[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](
	ctx context.Context,
	db TDB,
	fn func(txCtx context.Context, tx TTX) error,
	timeout time.Duration,
) error {

	txCtx, cancel := context.WithTimeout(ctx, timeout*2)
	defer cancel()

	var txOptions TTXO
	tx, err := db.BeginTx(txCtx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to execute in transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func executeNoTx[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](
	ctx context.Context,
	db TDB,
	fn func(ctx context.Context, db TDB) error,
	timeout time.Duration,
) error {

	fnCtx, cancel := context.WithTimeout(ctx, timeout*2)
	defer cancel()

	if err := fn(fnCtx, db); err != nil {
		return fmt.Errorf("failed to execute without transaction: %w", err)
	}

	return nil
}
