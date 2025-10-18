package migrations

import (
	"context"

	"github.com/jmoiron/sqlx"
)

var Migration00002 = MigrationSQLX{
	Version: 2,
	UpDownNoTX: &UpDownNoTXSQLX{
		Up: func(ctx context.Context, db *sqlx.DB) error {
			_, err := db.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_users_created_at ON users (created_at)`)
			return err
		},
		Down: func(ctx context.Context, db *sqlx.DB) error {
			_, err := db.ExecContext(ctx, `DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at`)
			return err
		},
	},
}
