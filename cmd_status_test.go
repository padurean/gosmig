package gosmig

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRunCmdStatus(t *testing.T) {
	testCases := []struct {
		name       string
		migrations []migrationMock
		setupMock  func(*dbMock, *dbRowMock)
		wantOut    string
		wantErr    string
	}{
		{
			name:       "no migrations defined",
			migrations: []migrationMock{},
			setupMock: func(db *dbMock, row *dbRowMock) {
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
			wantOut: "VERSION    STATUS      \n",
		},
		{
			name: "all migrations pending - db at version 0",
			migrations: []migrationMock{
				{
					Version: 1,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 2,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 3,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
			},
			setupMock: func(db *dbMock, row *dbRowMock) {
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
			wantOut: "VERSION    STATUS      \n" +
				"3          [ ] PENDING \n" +
				"2          [ ] PENDING \n" +
				"1          [ ] PENDING \n",
		},
		{
			name: "some migrations applied - db at version 2",
			migrations: []migrationMock{
				{
					Version: 1,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 2,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 3,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
			},
			setupMock: func(db *dbMock, row *dbRowMock) {
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
			},
			wantOut: "VERSION    STATUS      \n" +
				"3          [ ] PENDING \n" +
				"2          [x] APPLIED \n" +
				"1          [x] APPLIED \n",
		},
		{
			name: "all migrations applied - db at version 3",
			migrations: []migrationMock{
				{
					Version: 1,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 2,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 3,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
			},
			setupMock: func(db *dbMock, row *dbRowMock) {
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
			},
			wantOut: "VERSION    STATUS      \n" +
				"3          [x] APPLIED \n" +
				"2          [x] APPLIED \n" +
				"1          [x] APPLIED \n",
		},
		{
			name: "migrations not in order - should be sorted desc",
			migrations: []migrationMock{
				{
					Version: 2,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 5,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
				{
					Version: 1,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
			},
			setupMock: func(db *dbMock, row *dbRowMock) {
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
			},
			wantOut: "VERSION    STATUS      \n" +
				"5          [ ] PENDING \n" +
				"2          [x] APPLIED \n" +
				"1          [x] APPLIED \n",
		},
		{
			name: "error - failed to get DB version",
			migrations: []migrationMock{
				{
					Version: 1,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up:   func(ctx context.Context, tx *txMock) error { return nil },
						Down: func(ctx context.Context, tx *txMock) error { return nil },
					},
				},
			},
			setupMock: func(db *dbMock, row *dbRowMock) {
				// Get DB version - fails
				db.On("QueryRowContext", mock.Anything, selectDBVersionSQL).
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := new(dbMock)
			row := new(dbRowMock)
			tc.setupMock(db, row)

			var output bytes.Buffer

			err := runCmdStatus(
				context.Background(),
				tc.migrations,
				db,
				&output,
				defaultTimeout,
			)

			db.AssertExpectations(t)
			row.AssertExpectations(t)

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
