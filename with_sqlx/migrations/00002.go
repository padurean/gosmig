package migrations

import (
	"context"

	"github.com/jmoiron/sqlx"
)

var Migration00002 = MigrationSQLX{
	Version: 2,
	UpDownNoTX: &UpDownNoTXSQLX{
		Up: func(ctx context.Context, db *sqlx.DB) error {
			_, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN age INT`)
			return err
		},
		Down: func(ctx context.Context, db *sqlx.DB) error {
			_, err := db.ExecContext(ctx, `ALTER TABLE users DROP COLUMN age`)
			return err
		},
	},
}
