package migrations

import (
	"context"
	"database/sql"

	"github.com/padurean/gosmig"
)

var Migration00001 = MigrationSQLX{
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
}
