package gosmig

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateMigrationsTableIfNotExists(t *testing.T) {
	expectedSQL := createMigsTblSQL

	testCases := []struct {
		name      string
		setupMock func(*dbOrTxMock, *dbResultMock)
		wantErr   string
	}{
		{
			name: "success - table created",
			setupMock: func(dbOrTX *dbOrTxMock, result *dbResultMock) {
				dbOrTX.On("ExecContext", mock.Anything, expectedSQL).
					Return(result, nil).
					Once()
			},
		},
		{
			name: "error - exec context fails",
			setupMock: func(dbOrTX *dbOrTxMock, result *dbResultMock) {
				dbOrTX.On("ExecContext", mock.Anything, expectedSQL).
					Return(result, errors.New("database connection error")).
					Once()
			},
			wantErr: "failed to create migrations table if not exists: " +
				"database connection error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			result := new(dbResultMock)

			tc.setupMock(dbOrTX, result)

			ctx := context.Background()
			err := createMigrationsTableIfNotExists(ctx, dbOrTX, defaultTimeout)

			dbOrTX.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)

		})
	}
}

func TestGetDBVersion(t *testing.T) {
	expectedSQL := selectDBVersionSQL

	testCases := []struct {
		name      string
		setupMock func(*dbOrTxMock, *dbRowMock)
		wantVer   int
		wantErr   string
	}{
		{
			name: "success - version 5",
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock) {
				dbOrTX.On("QueryRowContext", mock.Anything, expectedSQL).
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
			wantVer: 5,
		},
		{
			name: "success - version 0 (no migrations)",
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock) {
				dbOrTX.On("QueryRowContext", mock.Anything, expectedSQL).
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
			wantVer: 0,
		},
		{
			name: "error - scan fails",
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock) {
				dbOrTX.On("QueryRowContext", mock.Anything, expectedSQL).
					Return(row).
					Once()
				row.On("Scan", mock.MatchedBy(func(dest []any) bool {
					return len(dest) == 1
				})).
					Return(errors.New("scan error")).
					Once()
			},
			wantErr: "failed to get current DB version: scan error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			row := new(dbRowMock)

			tc.setupMock(dbOrTX, row)

			ctx := context.Background()
			version, err := getDBVersion(ctx, dbOrTX, defaultTimeout)

			dbOrTX.AssertExpectations(t)
			row.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				require.Equal(t, 0, version)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantVer, version)
		})
	}
}

func TestInsertDBVersion(t *testing.T) {
	expectedSQL := insertMigVersionSQL

	testCases := []struct {
		name      string
		version   int
		setupMock func(*dbOrTxMock, *dbResultMock)
		wantErr   string
	}{
		{
			name:    "success",
			version: 3,
			setupMock: func(dbOrTX *dbOrTxMock, result *dbResultMock) {
				dbOrTX.On("ExecContext", mock.Anything, expectedSQL, 3).
					Return(result, nil).
					Once()
			},
		},
		{
			name:    "error - exec context fails",
			version: 5,
			setupMock: func(dbOrTX *dbOrTxMock, result *dbResultMock) {
				dbOrTX.On("ExecContext", mock.Anything, expectedSQL, 5).
					Return(result, errors.New("unique constraint violation")).
					Once()
			},
			wantErr: "failed to insert migration version 5 into migrations table: " +
				"unique constraint violation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			result := new(dbResultMock)

			tc.setupMock(dbOrTX, result)

			ctx := context.Background()
			err := insertDBVersion(ctx, dbOrTX, tc.version, defaultTimeout)

			dbOrTX.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeleteDBVersion(t *testing.T) {
	expectedSQL := deleteMigVersionSQL

	testCases := []struct {
		name      string
		version   int
		setupMock func(*dbOrTxMock, *dbResultMock)
		wantErr   string
	}{
		{
			name:    "success",
			version: 3,
			setupMock: func(dbOrTX *dbOrTxMock, result *dbResultMock) {
				dbOrTX.On("ExecContext", mock.Anything, expectedSQL, 3).
					Return(result, nil).
					Once()
			},
		},
		{
			name:    "error - exec context fails",
			version: 5,
			setupMock: func(dbOrTX *dbOrTxMock, result *dbResultMock) {
				dbOrTX.On("ExecContext", mock.Anything, expectedSQL, 5).
					Return(result, errors.New("database error")).
					Once()
			},
			wantErr: "failed to delete migration version 5 from migrations table: " +
				"database error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			result := new(dbResultMock)

			tc.setupMock(dbOrTX, result)

			ctx := context.Background()
			err := deleteDBVersion(ctx, dbOrTX, tc.version, defaultTimeout)

			dbOrTX.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestExecuteInTx(t *testing.T) {
	testCases := []struct {
		name      string
		setupMock func(*dbMock, *txMock)
		fnToExec  func(context.Context, *txMock) error
		wantErr   string
	}{
		{
			name: "success - transaction commits",
			setupMock: func(db *dbMock, tx *txMock) {
				db.On("BeginTx", mock.Anything, mock.AnythingOfType("txOptionsMock")).
					Return(tx, nil).
					Once()
				tx.On("Commit").
					Return(nil).
					Once()
			},
			fnToExec: func(ctx context.Context, tx *txMock) error {
				return nil
			},
		},
		{
			name: "error - begin transaction fails",
			setupMock: func(db *dbMock, tx *txMock) {
				db.On("BeginTx", mock.Anything, mock.AnythingOfType("txOptionsMock")).
					Return(tx, errors.New("connection error")).
					Once()
			},
			fnToExec: func(ctx context.Context, tx *txMock) error {
				return nil
			},
			wantErr: "failed to begin transaction: connection error",
		},
		{
			name: "error - function execution fails and transaction rolls back",
			setupMock: func(db *dbMock, tx *txMock) {
				db.On("BeginTx", mock.Anything, mock.AnythingOfType("txOptionsMock")).
					Return(tx, nil).
					Once()
				tx.On("Rollback").
					Return(nil).
					Once()
			},
			fnToExec: func(ctx context.Context, tx *txMock) error {
				return errors.New("business logic error")
			},
			wantErr: "failed to execute in transaction: business logic error",
		},
		{
			name: "error - commit fails",
			setupMock: func(db *dbMock, tx *txMock) {
				db.On("BeginTx", mock.Anything, mock.AnythingOfType("txOptionsMock")).
					Return(tx, nil).
					Once()
				tx.On("Commit").
					Return(errors.New("commit failed")).
					Once()
			},
			fnToExec: func(ctx context.Context, tx *txMock) error {
				return nil
			},
			wantErr: "failed to commit transaction: commit failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := new(dbMock)
			tx := new(txMock)

			tc.setupMock(db, tx)

			ctx := context.Background()
			err := executeInTx(ctx, db, tc.fnToExec, defaultTimeout)

			db.AssertExpectations(t)
			tx.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestExecuteNoTx(t *testing.T) {
	testCases := []struct {
		name     string
		fnToExec func(context.Context, *dbMock) error
		wantErr  string
	}{
		{
			name: "success - function executes without transaction",
			fnToExec: func(ctx context.Context, db *dbMock) error {
				return nil
			},
		},
		{
			name: "error - function execution fails",
			fnToExec: func(ctx context.Context, db *dbMock) error {
				return errors.New("execution error")
			},
			wantErr: "failed to execute without transaction: execution error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := new(dbMock)

			ctx := context.Background()
			err := executeNoTx(ctx, db, tc.fnToExec, defaultTimeout)

			db.AssertExpectations(t)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
