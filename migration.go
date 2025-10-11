package gosmig

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
)

type (
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
	UpDownSQL     = UpDown[*sql.Row, sql.Result, *sql.Tx]
	UpDownNoTXSQL = UpDown[*sql.Row, sql.Result, *sql.DB]
)

func (m Migration[TDBRow, TDBResult, TTX, TTXO, TDB]) validate() error {
	if m.Version <= 0 {
		return fmt.Errorf("migration version must be > 0")
	}

	if m.UpDown == nil && m.UpDownNoTX == nil {
		return fmt.Errorf(
			"migration %d must have UpDown xor UpDownNoTX fields defined",
			m.Version)
	}

	if m.UpDown != nil && m.UpDownNoTX != nil {
		return fmt.Errorf(
			"migration %d must have only one of UpDown or UpDownNoTX fields defined",
			m.Version)
	}

	if m.UpDown != nil {
		if m.UpDown.Up == nil || m.UpDown.Down == nil {
			return fmt.Errorf(
				"migration %d UpDown must have both Up and Down functions defined",
				m.Version)
		}
	}

	if m.UpDownNoTX != nil {
		if m.UpDownNoTX.Up == nil || m.UpDownNoTX.Down == nil {
			return fmt.Errorf(
				"migration %d UpDownNoTX must have both Up and Down functions defined",
				m.Version)
		}
	}

	return nil
}

func sortMigrationsDesc[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](
	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
) {
	slices.SortFunc(migrations, func(a, b Migration[TDBRow, TDBResult, TTX, TTXO, TDB]) int {
		return b.Version - a.Version
	})
}

func sortMigrationsAsc[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TTXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TTXO]](
	migrations []Migration[TDBRow, TDBResult, TTX, TTXO, TDB],
) {
	slices.SortFunc(migrations, func(a, b Migration[TDBRow, TDBResult, TTX, TTXO, TDB]) int {
		return a.Version - b.Version
	})
}
