package gosmig

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRunCmdVersion(t *testing.T) {
	testCases := []struct {
		name      string
		setupMock func(*dbOrTxMock, *dbRowMock)
		wantOut   string
		wantErr   string
	}{
		{
			name: "success - version 0 (no migrations applied)",
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock) {
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
			},
			wantOut: "Current database version:\n0\n",
		},
		{
			name: "success - version 5",
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock) {
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
			wantOut: "Current database version:\n5\n",
		},
		{
			name: "error - failed to get DB version",
			setupMock: func(dbOrTX *dbOrTxMock, row *dbRowMock) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbOrTX := new(dbOrTxMock)
			row := new(dbRowMock)
			tc.setupMock(dbOrTX, row)

			var output bytes.Buffer

			err := runCmdVersion(context.Background(), dbOrTX, &output, defaultTimeout)

			dbOrTX.AssertExpectations(t)
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
