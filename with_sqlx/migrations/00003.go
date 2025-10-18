package migrations

import (
	"context"
	"database/sql"

	"github.com/padurean/gosmig"
)

var Migration00003 = MigrationSQLX{
	Version: 3,
	UpDown: &gosmig.UpDownSQL{
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `ALTER TABLE users ADD COLUMN address TEXT`)
			return err
		},
		Down: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `ALTER TABLE users DROP COLUMN address`)
			return err
		},
	},
}
