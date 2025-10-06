package gosmig

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGosmig(t *testing.T) {
	testCases := []struct {
		name        string
		migrations  []MigrationSQL
		connectToDB func(url string, timeout time.Duration) (*sql.DB, error)
		config      *Config
		getArgs     func() []string
		osExit      func(int)
		out, errOut io.Writer
		wantErr     string
	}{
		{
			name:       "nil migrations slice",
			migrations: nil,
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  nil,
			getArgs: func() []string { return nil },
			osExit:  func(code int) {},
			out:     io.Discard,
			errOut:  io.Discard,
			wantErr: "no migrations provided",
		},
		{
			name:       "empty migrations slice",
			migrations: []MigrationSQL{},
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  nil,
			getArgs: func() []string { return nil },
			osExit:  func(code int) {},
			out:     io.Discard,
			errOut:  io.Discard,
			wantErr: "no migrations provided",
		},
		{
			name:        "nil connectToDB function",
			migrations:  []MigrationSQL{{Version: 1}},
			connectToDB: nil,
			config:      nil,
			getArgs:     func() []string { return nil },
			osExit:      func(code int) {},
			out:         io.Discard,
			errOut:      io.Discard,
			wantErr:     "connectToDB function is nil",
		},
		{
			name:       "nil getArgs function",
			migrations: []MigrationSQL{{Version: 1}},
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  &Config{}, // this should result in config.EnsureDefaults being called
			getArgs: nil,
			osExit:  func(code int) {},
			out:     io.Discard,
			errOut:  io.Discard,
			wantErr: "getArgs function is nil",
		},
		{
			name:       "nil osExit function",
			migrations: []MigrationSQL{{Version: 1}},
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  nil,
			getArgs: func() []string { return nil },
			osExit:  nil,
			out:     io.Discard,
			errOut:  io.Discard,
			wantErr: "osExit function is nil",
		},
		{
			name:       "nil out writer",
			migrations: []MigrationSQL{{Version: 1}},
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  nil,
			getArgs: func() []string { return nil },
			osExit:  func(code int) {},
			out:     nil,
			errOut:  io.Discard,
			wantErr: "out writer is nil",
		},
		{
			name:       "nil errOut writer",
			migrations: []MigrationSQL{{Version: 1}},
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  nil,
			getArgs: func() []string { return nil },
			osExit:  func(code int) {},
			out:     io.Discard,
			errOut:  nil,
			wantErr: "errOut writer is nil",
		},
		{
			name: "invalid migration",
			migrations: []MigrationSQL{
				{Version: 1, UpDown: &UpDownSQL{Up: nil, Down: nil}},
			},
			connectToDB: func(url string, timeout time.Duration) (*sql.DB, error) {
				return nil, nil
			},
			config:  nil,
			getArgs: func() []string { return nil },
			osExit:  func(code int) {},
			out:     io.Discard,
			errOut:  io.Discard,
			wantErr: "migration 1 UpDown must have both Up and Down functions defined",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newGosmig(
				tc.migrations,
				tc.connectToDB,
				tc.config,
				tc.getArgs,
				tc.osExit,
				tc.out,
				tc.errOut,
			)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}

	t.Run("invalid args", func(t *testing.T) {
		migrations := []MigrationSQL{
			{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   func(ctx context.Context, tx *sql.Tx) error { return nil },
					Down: func(ctx context.Context, tx *sql.Tx) error { return nil },
				},
			},
		}
		connectToDB := func(url string, timeout time.Duration) (*sql.DB, error) {
			return nil, nil
		}
		getArgs := func() []string { return []string{"only-one-arg"} }
		var exitCode int
		osExit := func(code int) { exitCode = code }
		var outW, errW strings.Builder

		goSMig, err := newGosmig(
			migrations, connectToDB, nil, getArgs, osExit, &outW, &errW)
		require.NoError(t, err)
		goSMig()
		require.Equal(t, 1, exitCode)
		require.Contains(t, errW.String(), "wrong number of arguments")
	})

	t.Run("connectToDB error", func(t *testing.T) {
		migrations := []MigrationSQL{
			{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   func(ctx context.Context, tx *sql.Tx) error { return nil },
					Down: func(ctx context.Context, tx *sql.Tx) error { return nil },
				},
			},
		}
		connectToDB := func(url string, timeout time.Duration) (*sql.DB, error) {
			return nil, fmt.Errorf("failed to connect to database: %s", "timed out")
		}
		getArgs := func() []string { return []string{"postgres://localhost/db", "status"} }
		var exitCode int
		osExit := func(code int) { exitCode = code }
		var outW, errW strings.Builder

		goSMig, err := newGosmig(
			migrations, connectToDB, nil, getArgs, osExit, &outW, &errW)
		require.NoError(t, err)
		goSMig()
		require.Equal(t, 2, exitCode)
		require.Contains(t, errW.String(), "failed to connect to database: timed out")
	})

	t.Run("create migrations table and close db errors", func(t *testing.T) {
		errCreateMigrationsTable := errors.New("failed to create migrations table")
		errCloseDB := errors.New("failed to close the database connection")

		dbMockInstance := new(dbMock)
		dbMockInstance.On("ExecContext", mock.Anything, mock.Anything).
			Return(new(dbResultMock), errCreateMigrationsTable)
		dbMockInstance.On("Close").Return(errCloseDB)

		migrations := []migrationMock{
			{
				Version: 1,
				UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
					Up:   func(ctx context.Context, tx *txMock) error { return nil },
					Down: func(ctx context.Context, tx *txMock) error { return nil },
				},
			},
		}
		connectToDB := func(url string, timeout time.Duration) (*dbMock, error) {
			return dbMockInstance, nil
		}
		getArgs := func() []string {
			return []string{"postgres://localhost/db", "status"}
		}
		var exitCode int
		osExit := func(code int) { exitCode = code }
		var outW, errW strings.Builder

		goSMig, err := newGosmig(
			migrations, connectToDB, nil, getArgs, osExit, &outW, &errW)
		require.NoError(t, err)
		goSMig()
		require.Equal(t, 3, exitCode)
		dbMockInstance.AssertExpectations(t)
		require.Contains(t, errW.String(), errCreateMigrationsTable.Error())
		require.Contains(t, errW.String(), errCloseDB.Error())
	})

	for i, cmd := range allCommands {
		t.Run(fmt.Sprintf("executes command %s", cmd), func(t *testing.T) {
			dbRowMockInstance := new(dbRowMock)
			dbVersion := 0
			if cmd == cmdDown {
				dbVersion = 1
			}
			var scanErr error
			if slices.Contains([]string{cmdStatus, cmdVersion}, cmd) {
				scanErr = fmt.Errorf("scan error when getting db version for command %s", cmd)
			}
			dbRowMockInstance.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
				arg := args.Get(0).([]any)
				*(arg[0].(*int)) = dbVersion
			}).Return(scanErr)

			dbMockInstance := new(dbMock)
			dbMockInstance.On("QueryRowContext", mock.Anything, mock.Anything).
				Return(dbRowMockInstance, nil)
			dbMockInstance.On("ExecContext", mock.Anything, mock.Anything).
				Return(new(dbResultMock), nil)

			txMockInstance := new(txMock)
			if scanErr == nil {
				dbMockInstance.On("BeginTx", mock.Anything, mock.Anything).
					Return(txMockInstance, nil)
			}

			dbMockInstance.On("Close").Return(nil)

			if scanErr == nil {
				txMockInstance.On("QueryRowContext", mock.Anything, mock.Anything).
					Return(dbRowMockInstance, nil)
				txMockInstance.On("Rollback").Return(nil)
			}

			wantUpOrDownErr := fmt.Errorf("some error in %s", cmd)
			migrations := []migrationMock{
				{
					Version: 1,
					UpDown: &UpDown[*dbRowMock, *dbResultMock, *txMock]{
						Up: func(ctx context.Context, tx *txMock) error {
							if slices.Contains([]string{cmd, cmdUpOne}, cmd) {
								return wantUpOrDownErr
							}
							return nil
						},
						Down: func(ctx context.Context, tx *txMock) error {
							if cmd == cmdDown {
								return wantUpOrDownErr
							}
							return nil
						},
					},
				},
			}
			connectToDB := func(url string, timeout time.Duration) (*dbMock, error) {
				return dbMockInstance, nil
			}
			getArgs := func() []string {
				return []string{"postgres://localhost/db", cmd}
			}
			var exitCode int
			osExit := func(code int) { exitCode = code }
			var outW, errW strings.Builder

			goSMig, err := newGosmig(
				migrations, connectToDB, nil, getArgs, osExit, &outW, &errW)
			require.NoError(t, err)
			goSMig()
			require.Equal(t, 5+i, exitCode)
			dbRowMockInstance.AssertExpectations(t)
			dbMockInstance.AssertExpectations(t)
			txMockInstance.AssertExpectations(t)
			upOrDown := "up"
			if cmd == cmdDown {
				upOrDown = "down"
			}
			var wantErr string
			if scanErr != nil {
				wantErr = scanErr.Error()
			} else {
				wantErr = fmt.Sprintf(
					"failed to apply migration.%s version %d: %v",
					upOrDown, migrations[0].Version, wantErr)
			}
			require.Contains(t, errW.String(), wantErr)
			require.Empty(t, outW)
		})
	}
}

func TestParseArgs(t *testing.T) {
	type testCase struct {
		name        string
		args        []string
		wantURL     string
		wantCommand string
		wantErr     string
	}

	testCases := []testCase{
		{
			name:    "wrong number of arguments",
			args:    []string{"postgres://localhost/db"},
			wantErr: "wrong number of arguments",
		},
		{
			name:    "unknown command",
			args:    []string{"postgres://localhost/db", "invalid"},
			wantErr: "unknown command",
		},
	}

	for _, cmd := range allCommands {
		testCases = append(testCases, testCase{
			name:        fmt.Sprintf("valid command %s", cmd),
			args:        []string{"postgres://localhost/db", cmd},
			wantURL:     "postgres://localhost/db",
			wantCommand: cmd,
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotCommand, err := parseArgs(tc.args)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantURL, gotURL)
			require.Equal(t, tc.wantCommand, gotCommand)
		})
	}
}

func TestUsage(t *testing.T) {
	want := "Usage: gosmig <db_url> <up|up-one|down|status|version>"
	require.Equal(t, want, usage())
}

func TestErrExit(t *testing.T) {
	var exitCode int
	mockOsExit := func(code int) {
		exitCode = code
	}

	err := fmt.Errorf("failed to connect to database: %s", "timed out")
	var output strings.Builder
	errExit(3, err, &output, mockOsExit)

	wantOutput := `Usage: gosmig <db_url> <up|up-one|down|status|version>
failed to connect to database: timed out
`
	require.Equal(t, wantOutput, output.String())
	require.Equal(t, 3, exitCode)
}
