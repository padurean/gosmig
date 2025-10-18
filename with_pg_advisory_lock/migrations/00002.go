package migrations

import (
	"context"
)

var Migration00002 = MigrationSQL{
	Version: 2,
	UpDownNoTX: &UpDownNoTXSQL{
		Up: func(ctx context.Context, db *LockedDB) error {
			_, err := db.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_users_created_at ON users (created_at)`)
			return err
		},
		Down: func(ctx context.Context, db *LockedDB) error {
			_, err := db.ExecContext(ctx, `DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at`)
			return err
		},
	},
}
