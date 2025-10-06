package gosmig

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRunCmdDown(t *testing.T) {
	testCases := []struct {
		name       string
		migrations []migrationMock
		setupMock  func(*dbMock, *txMock, *dbRowMock, *dbResultMock)
		wantOut    string
		wantErr    string
	}{
		{
			name:       "no migrations to roll back - db at version 0",
			migrations: createTestMigrations(1, 2),
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
			},
			wantOut: "No migrations to roll back\n",
		},
		{
			name:       "roll back one migration - db at version 3",
			migrations: createTestMigrations(1, 2, 3),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 3
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
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

				// Roll back migration 3
				setupMigrationDownMocks(db, tx, row, result, 3, 3)
			},
			wantOut: "[x]-->[ ] Rolled back migration version 3\n",
		},
		{
			name:       "roll back one migration - db at version 2",
			migrations: createTestMigrations(1, 2, 3),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 2
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

				// Roll back migration 2
				setupMigrationDownMocks(db, tx, row, result, 2, 2)
			},
			wantOut: "[x]-->[ ] Rolled back migration version 2\n",
		},
		{
			name:       "roll back one migration - db at version 1",
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

				// Roll back migration 1
				setupMigrationDownMocks(db, tx, row, result, 1, 1)
			},
			wantOut: "[x]-->[ ] Rolled back migration version 1\n",
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
			migrations: createTestMigrations(2),
			setupMock: func(db *dbMock, tx *txMock, row *dbRowMock, result *dbResultMock) {
				// Get initial DB version - returns 2
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

				// BeginTx
				db.On("BeginTx", mock.Anything, mock.Anything).
					Return(tx, nil).
					Once()

				// Get DB version in TX - returns 2
				tx.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
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
				// Get initial DB version - returns 1
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

				// Get DB version for no-TX migration - returns 1
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

				// Migration fails
				db.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, errors.New("cannot drop index concurrently in transaction")).
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

			err := runCmdDown(context.Background(), tc.migrations, db, &output, defaultTimeout)

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

func TestMigrateDown(t *testing.T) {
	testCases := []struct {
		name      string
		version   int
		setupMock func(*dbOrTxMock, *dbRowMock, *dbResultMock)
		wantErr   string
	}{
		{
			name:    "success - migrate from version 1 to 0",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 1
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
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

				// Migration executes
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, nil).
					Once()

				// Delete version
				dbOrTX.On("ExecContext", mock.Anything, deleteMigVersionSQL, 1).
					Return(result, nil).
					Once()
			},
		},
		{
			name:    "success - migrate from version 3 to 2",
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

				// Migration executes
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, nil).
					Once()

				// Delete version
				dbOrTX.On("ExecContext", mock.Anything, deleteMigVersionSQL, 3).
					Return(result, nil).
					Once()
			},
		},
		{
			name:    "error - version not applied (version 3 > current DB version 2)",
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
			},
			wantErr: "migration version 3 > current DB version 2",
		},
		{
			name:    "error - version not applied (version 5 > current DB version 3)",
			version: 5,
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
			wantErr: "migration version 5 > current DB version 3",
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
			name:    "error - migration.down fails",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 1
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
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

				// Migration fails
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, errors.New("table does not exist")).
					Once()
			},
			wantErr: "failed to apply migration.down version 1: table does not exist",
		},
		{
			name:    "error - failed to delete version",
			version: 1,
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock, result *dbResultMock) {
				// Get DB version - returns 1
				dbOrTX.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
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

				// Migration executes
				dbOrTX.On("ExecContext", mock.Anything, mock.Anything).
					Return(result, nil).
					Once()

				// Delete version fails
				dbOrTX.On("ExecContext", mock.Anything, deleteMigVersionSQL, 1).
					Return(result, errors.New("constraint violation")).
					Once()
			},
			wantErr: "failed to delete migration version 1 from migrations table: " +
				"constraint violation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			row := new(dbRowMock)
			result := new(dbResultMock)

			tc.setupMock(dbOrTX, row, result)

			// Create a simple down function that executes a dummy query
			downFunc := func(ctx context.Context, db *dbOrTxMock) error {
				_, err := db.ExecContext(ctx, "DROP TABLE test")
				return err
			}

			// Call migrateDown and execute the returned function
			migrateFn := migrateDown(tc.version, downFunc, defaultTimeout)
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

// Helper function to set up mocks for a migration down operation
func setupMigrationDownMocks(
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

	// Delete version
	tx.On("ExecContext", mock.Anything, deleteMigVersionSQL, targetVersion).
		Return(result, nil).
		Once()

	// Commit
	tx.On("Commit").
		Return(nil).
		Once()
}
