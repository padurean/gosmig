package gosmig

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRunCmdUp(t *testing.T) {
	testCases := []struct {
		name       string
		migrations []migrationMock
		setupMock  func(*dbMock, *txMock, *dbRowMock, *dbResultMock)
		limit      int
		wantOut    string
		wantErr    string
	}{
		{
			name:       "no migrations to apply - db already at latest version",
			migrations: createTestMigrations(1, 2),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 2 (latest)
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 2
					}).
					Return(nil).
					Once()
			},
			wantOut: "No migrations to apply\n",
		},
		{
			name:       "apply all migrations - db at version 0",
			migrations: createTestMigrations(1, 2, 3),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 0
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration 1
				setupMigrationUpMocks(db, tx, row, result, 0, 1)

				// Migration 2
				setupMigrationUpMocks(db, tx, row, result, 1, 2)

				// Migration 3
				setupMigrationUpMocks(db, tx, row, result, 2, 3)
			},
			wantOut: "[x] Applied migration version 1\n" +
				"[x] Applied migration version 2\n" +
				"[x] Applied migration version 3\n" +
				"3 migration(s) applied\n",
		},
		{
			name:       "apply remaining migrations - db at version 1",
			migrations: createTestMigrations(1, 2, 3),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 1
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 1
					}).
					Return(nil).
					Once()

				// Migration 2
				setupMigrationUpMocks(db, tx, row, result, 1, 2)

				// Migration 3
				setupMigrationUpMocks(db, tx, row, result, 2, 3)
			},
			wantOut: "[x] Applied migration version 2\n" +
				"[x] Applied migration version 3\n" +
				"2 migration(s) applied\n",
		},
		{
			name:       "apply one migration with limit - up-one command",
			migrations: createTestMigrations(1, 2, 3),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 0
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration 1 only
				setupMigrationUpMocks(db, tx, row, result, 0, 1)
			},
			limit: 1,
			wantOut: "[x] Applied migration version 1\n" +
				"1 migration(s) applied\n",
		},
		{
			name:       "error getting initial DB version",
			migrations: createTestMigrations(1),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Return(errors.New("connection error")).
					Once()
			},
			wantErr: "failed to get current DB version: connection error",
		},
		{
			name:       "error during migration execution - with TX",
			migrations: createTestMigrations(1),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get initial DB version - returns 0
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// BeginTx
				db.On("BeginTx", mock.Anything, mock.Anything).
					Return(tx, nil).
					Once()

				// Get DB version in TX - returns 0
				tx.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration fails
				tx.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, errors.New("migration failed")).
					Once()

				// Rollback
				tx.On("Rollback").
					Return(nil).
					Once()
			},
			wantErr: "execute in TX",
		},
		{
			name: "error during migration execution - no TX",
			migrations: []migrationMock{
				{
					Version: 1,
					UpDownNoTX: &UpDown[*dbRowMock, *dbResultMock, *dbMock]{
						Up: func(ctx context.Context, db *dbMock) error {
							_, err := db.ExecContext(ctx, "CREATE INDEX CONCURRENTLY idx_test ON test(id)")
							return err
						},
						Down: func(ctx context.Context, db *dbMock) error {
							_, err := db.ExecContext(ctx, "DROP INDEX CONCURRENTLY idx_test")
							return err
						},
					},
				},
			},
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get initial DB version - returns 0
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Get DB version for no-TX migration - returns 0
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration fails
				db.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, errors.New("cannot create index concurrently in transaction")).
					Once()
			},
			wantErr: "execute without TX",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := new(dbMock)
			tx := new(txMock)
			row := new(dbRowMock)
			result := new(dbResultMock)

			tc.setupMock(db, tx, row, result)

			var output bytes.Buffer

			err := runCmdUp(context.Background(), tc.migrations, db, &output, tc.limit, defaultTimeout)

			db.AssertExpectations(t)
			tx.AssertExpectations(t)
			row.AssertExpectations(t)
			result.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantOut, output.String())
		})
	}
}

func TestMigrateUp(t *testing.T) {
	testCases := []struct {
		name      string
		version   int
		setupMock func(*dbOrTxMock, *dbRowMock, *dbResultMock)
		wantErr   string
	}{
		{
			name:    "success - migrate from version 0 to 1",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 0
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration executes
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, nil).
					Once()

				// Insert version
				dbOrTX.On("ExecContext", mock.Anything, insertMigVersionSQL, 1).
					Return(result, nil).
					Once()
			},
		},
		{
			name:    "success - migrate from version 2 to 3",
			version: 3,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 2
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 2
					}).
					Return(nil).
					Once()

				// Migration executes
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, nil).
					Once()

				// Insert version
				dbOrTX.On("ExecContext", mock.Anything, insertMigVersionSQL, 3).
					Return(result, nil).
					Once()
			},
		},
		{
			name:    "error - version already applied (version 3 <= current DB version 3)",
			version: 3,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 3
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 3
					}).
					Return(nil).
					Once()
			},
			wantErr: "migration version 3 <= current DB version 3",
		},
		{
			name:    "error - version already applied (version 2 < current DB version 5)",
			version: 2,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 5
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 5
					}).
					Return(nil).
					Once()
			},
			wantErr: "migration version 2 <= current DB version 5",
		},
		{
			name:    "error - failed to get DB version",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Return(errors.New("database connection error")).
					Once()
			},
			wantErr: "failed to get current DB version: database connection error",
		},
		{
			name:    "error - migration.up fails",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 0
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration fails
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, errors.New("table already exists")).
					Once()
			},
			wantErr: "failed to apply migration.up version 1: table already exists",
		},
		{
			name:    "error - failed to insert version",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 0
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Run(func(args mock.Arguments) {
						dest := args.Get(0).([]any)
						ptr := dest[0].(*int)
						*ptr = 0
					}).
					Return(nil).
					Once()

				// Migration executes
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, nil).
					Once()

				// Insert version fails
				dbOrTX.On("ExecContext", mock.Anything, insertMigVersionSQL, 1).
					Return(result, errors.New("unique constraint violation")).
					Once()
			},
			wantErr: "failed to insert migration version 1 into migrations table: " +
				"unique constraint violation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			row := new(dbRowMock)
			result := new(dbResultMock)

			tc.setupMock(dbOrTX, row, result)

			// Create a simple up function that executes a dummy query
			upFunc := func(ctx context.Context, db *dbOrTxMock) error {
				_, err := db.ExecContext(ctx, "CREATE TABLE test (id INT)")
				return err
			}

			// Call migrateUp and execute the returned function
			migrateFn := migrateUp(tc.version, upFunc, defaultTimeout)
			err := migrateFn(context.Background(), dbOrTX)

			dbOrTX.AssertExpectations(t)
			row.AssertExpectations(t)
			result.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// Helper function to create test migrations with UpDown
func createTestMigrations(versions ...int) []migrationMock {
	migrations := make([]migrationMock, len(versions))
	for i, v := range versions {
		migrations[i] = migrationMock{
			Version: v,
			UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
				Up: func(ctx context.Context, tx *txMock) error {
					_, err := tx.ExecContext(ctx, "CREATE TABLE test (id INT)")
					return err
				},
				Down: func(ctx context.Context, tx *txMock) error {
					_, err := tx.ExecContext(ctx, "DROP TABLE test")
					return err
				},
			},
		}
	}
	return migrations
}

// Helper function to set up mocks for a migration up operation
func setupMigrationUpMocks(
	db *dbMock,
	tx *txMock,
	row *dbRowMock,
	result *dbResultMock,
	currentVersion, targetVersion int,
) {

	// BeginTx
	db.On("BeginTx", mock.Anything, mock.Anything).
		Return(tx, nil).
		Once()

	// Get DB version in TX
	tx.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
		Return(row).
		Once()
	row.On("Scan", mock.MatchedBy(func(dest []any) bool {
		return len(dest) == 1
	})).
		Run(func(args mock.Arguments) {
			dest := args.Get(0).([]any)
			ptr := dest[0].(*int)
			*ptr = currentVersion
		}).
		Return(nil).
		Once()

	// Migration executes
	tx.On("ExecContext", mock.Anything, mock.Anything).
		Return(result, nil).
		Once()

	// Insert version
	tx.On("ExecContext", mock.Anything, insertMigVersionSQL, targetVersion).
		Return(result, nil).
		Once()

	// Commit
	tx.On("Commit").
		Return(nil).
		Once()
}
