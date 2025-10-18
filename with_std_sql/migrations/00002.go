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
			_, err := db.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_users_created_at ON users (created_at)`)
			return err
		},
		Down: func(ctx context.Context, db *sql.DB) error {
			_, err := db.ExecContext(ctx, `DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at`)
			return err
		},
	},
}
