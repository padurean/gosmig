package migrations

import (
	"context"
	"database/sql"

	"github.com/padurean/gosmig"
)

var Migration00002 = gosmig.MigrationSQL{
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
}
