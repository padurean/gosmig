package gosmig

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationValidate(t *testing.T) {
	// Helper functions for test cases
	validUpFunc := func(ctx context.Context, tx *sql.Tx) error { return nil }
	validDownFunc := func(ctx context.Context, tx *sql.Tx) error { return nil }
	validUpNoTXFunc := func(ctx context.Context, db *sql.DB) error { return nil }
	validDownNoTXFunc := func(ctx context.Context, db *sql.DB) error { return nil }

	testCases := []struct {
		name      string
		migration MigrationSQL
		wantErr   string
	}{
		{
			name: "valid migration with UpDown",
			migration: MigrationSQL{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   validUpFunc,
					Down: validDownFunc,
				},
			},
		},
		{
			name: "valid migration with UpDownNoTX",
			migration: MigrationSQL{
				Version: 1,
				UpDownNoTX: &UpDownNoTXSQL{
					Up:   validUpNoTXFunc,
					Down: validDownNoTXFunc,
				},
			},
		},
		{
			name: "invalid version zero",
			migration: MigrationSQL{
				Version: 0,
				UpDown: &UpDownSQL{
					Up:   validUpFunc,
					Down: validDownFunc,
				},
			},
			wantErr: "migration version must be > 0",
		},
		{
			name: "invalid version negative",
			migration: MigrationSQL{
				Version: -1,
				UpDown: &UpDownSQL{
					Up:   validUpFunc,
					Down: validDownFunc,
				},
			},
			wantErr: "migration version must be > 0",
		},
		{
			name: "missing both UpDown and UpDownNoTX",
			migration: MigrationSQL{
				Version: 1,
			},
			wantErr: "must have UpDown xor UpDownNoTX fields defined",
		},
		{
			name: "both UpDown and UpDownNoTX defined",
			migration: MigrationSQL{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   validUpFunc,
					Down: validDownFunc,
				},
				UpDownNoTX: &UpDownNoTXSQL{
					Up:   validUpNoTXFunc,
					Down: validDownNoTXFunc,
				},
			},
			wantErr: "must have only one of UpDown or UpDownNoTX fields defined",
		},
		{
			name: "UpDown missing Up function",
			migration: MigrationSQL{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   nil,
					Down: validDownFunc,
				},
			},
			wantErr: "UpDown must have both Up and Down functions defined",
		},
		{
			name: "UpDown missing Down function",
			migration: MigrationSQL{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   validUpFunc,
					Down: nil,
				},
			},
			wantErr: "UpDown must have both Up and Down functions defined",
		},
		{
			name: "UpDown missing both Up and Down functions",
			migration: MigrationSQL{
				Version: 1,
				UpDown: &UpDownSQL{
					Up:   nil,
					Down: nil,
				},
			},
			wantErr: "UpDown must have both Up and Down functions defined",
		},
		{
			name: "UpDownNoTX missing Up function",
			migration: MigrationSQL{
				Version: 1,
				UpDownNoTX: &UpDownNoTXSQL{
					Up:   nil,
					Down: validDownNoTXFunc,
				},
			},
			wantErr: "UpDownNoTX must have both Up and Down functions defined",
		},
		{
			name: "UpDownNoTX missing Down function",
			migration: MigrationSQL{
				Version: 1,
				UpDownNoTX: &UpDownNoTXSQL{
					Up:   validUpNoTXFunc,
					Down: nil,
				},
			},
			wantErr: "UpDownNoTX must have both Up and Down functions defined",
		},
		{
			name: "UpDownNoTX missing both Up and Down functions",
			migration: MigrationSQL{
				Version: 1,
				UpDownNoTX: &UpDownNoTXSQL{
					Up:   nil,
					Down: nil,
				},
			},
			wantErr: "UpDownNoTX must have both Up and Down functions defined",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.migration.validate()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateMigrations(t *testing.T) {
	// Helper functions for test cases
	validUpFunc := func(ctx context.Context, tx *sql.Tx) error { return nil }
	validDownFunc := func(ctx context.Context, tx *sql.Tx) error { return nil }
	validUpNoTXFunc := func(ctx context.Context, db *sql.DB) error { return nil }
	validDownNoTXFunc := func(ctx context.Context, db *sql.DB) error { return nil }

	testCases := []struct {
		name       string
		migrations []MigrationSQL
		wantErrs   []string
	}{
		{
			name: "valid multiple migrations",
			migrations: []MigrationSQL{
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 2,
					UpDownNoTX: &UpDownNoTXSQL{
						Up:   validUpNoTXFunc,
						Down: validDownNoTXFunc,
					},
				},
				{
					Version: 3,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
			},
		},
		{
			name: "duplicate migration version",
			migrations: []MigrationSQL{
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 2,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
			},
			wantErrs: []string{
				"invalid migration(s): migration version 1 is defined 2 times",
			},
		},
		{
			name: "multiple duplicate versions",
			migrations: []MigrationSQL{
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 2,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 2,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
			},
			wantErrs: []string{
				"invalid migration(s): ",
				"migration version 1 is defined 2 times",
				"migration version 2 is defined 2 times",
			},
		},
		{
			name: "triplicate migration version",
			migrations: []MigrationSQL{
				{
					Version: 5,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 5,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 5,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
			},
			wantErrs: []string{
				"invalid migration(s): migration version 5 is defined 3 times",
			},
		},
		{
			name: "multiple invalid migrations propagates individual errors",
			migrations: []MigrationSQL{
				{
					Version: 0,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   nil,
						Down: validDownFunc,
					},
				},
				{
					Version: 2,
				},
			},
			wantErrs: []string{
				"invalid migration(s): ",
				"migration version must be > 0",
				"migration 1 UpDown must have both Up and Down functions defined",
				"migration 2 must have UpDown xor UpDownNoTX fields defined",
			},
		},
		{
			name: "duplicate version combined with invalid migration",
			migrations: []MigrationSQL{
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 1,
					UpDown: &UpDownSQL{
						Up:   validUpFunc,
						Down: validDownFunc,
					},
				},
				{
					Version: 2,
					UpDown: &UpDownSQL{
						Up:   nil,
						Down: validDownFunc,
					},
				},
			},
			wantErrs: []string{
				"invalid migration(s):",
				"migration version 1 is defined 2 times",
				"migration 2 UpDown must have both Up and Down functions defined",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMigrations(tc.migrations)
			if len(tc.wantErrs) > 0 {
				for _, wantErr := range tc.wantErrs {
					require.ErrorContains(t, err, wantErr)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
