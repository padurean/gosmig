package gosmig

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGosmig_StdLibSQL(t *testing.T) {
	db, err := connectToDB_StdLibSQL(testDBURL, defaultTimeout)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	defer cleanup(t, []string{"example", migrationsTableName}, db)

	var outW, errW strings.Builder

	runCmd := func(
		migrations []MigrationSQL,
		command string,
	) {
		outW.Reset()
		errW.Reset()
		getArgs := func() []string { return []string{testDBURL, command} }
		osExit := func(code int) {
			require.FailNow(
				t, fmt.Sprintf("osExit called with code %d", code), "Error output:\n%s", errW.String())
		}
		goSMig, err := newGosmig(
			migrations, connectToDB_StdLibSQL, nil, getArgs, osExit, &outW, &errW)
		require.NoError(t, err)
		goSMig()
	}

	// NOTE: migrations don't have the right order to test that they are sorted and
	// applied in the correct order.
	migrations1 := []MigrationSQL{
		{
			Version: 2,
			UpDownNoTX: &UpDownNoTXSQL{
				Up: func(ctx context.Context, db *sql.DB) error {
					_, err := db.ExecContext(
						ctx, `ALTER TABLE example ADD COLUMN name TEXT`)
					return err
				},
				Down: func(ctx context.Context, db *sql.DB) error {
					_, err := db.ExecContext(ctx, `ALTER TABLE example DROP COLUMN name`)
					return err
				},
			},
		},
		{
			Version: 1,
			UpDown: &UpDownSQL{
				Up: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(
						ctx, `CREATE TABLE example (id SERIAL PRIMARY KEY)`)
					return err
				},
				Down: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, `DROP TABLE example`)
					return err
				},
			},
		},
	}

	// NOTE: In a real-world scenario, all migrations would typically be in a
	// single slice.
	// But things should still work fine for "up" command even if only the new
	// migrations are provided.
	migrations2 := []MigrationSQL{
		{
			Version: 3,
			UpDown: &UpDownSQL{
				Up: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, `ALTER TABLE example ADD COLUMN age INT`)
					return err
				},
				Down: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, `ALTER TABLE example DROP COLUMN age`)
					return err
				},
			},
		},
	}

	testGosmig(t, db, runCmd, &outW, &errW, migrations1, migrations2)
}

func TestGosmig_SQLX(t *testing.T) {
	db, err := connectToDB_SQLX(testDBURL, defaultTimeout)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	defer cleanup(t, []string{"example", migrationsTableName}, db)

	type (
		MigrationSQLX  = Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sqlx.DB]
		UpDownNoTXSQLX = UpDown[*sql.Row, sql.Result, *sqlx.DB]
	)

	var outW, errW strings.Builder

	runCmd := func(
		migrations []MigrationSQLX,
		command string,
	) {
		outW.Reset()
		errW.Reset()
		getArgs := func() []string { return []string{testDBURL, command} }
		osExit := func(code int) {
			require.FailNow(
				t, "osExit called with code %d. Error output:\n%s", code, errW.String())
		}
		goSMig, err := newGosmig(
			migrations, connectToDB_SQLX, nil, getArgs, osExit, &outW, &errW)
		require.NoError(t, err)
		goSMig()
	}

	// NOTE: migrations don't have the right order to test that they are sorted and
	// applied in the correct order.
	migrations1 := []MigrationSQLX{
		{
			Version: 2,
			UpDownNoTX: &UpDownNoTXSQLX{
				Up: func(ctx context.Context, db *sqlx.DB) error {
					_, err := db.ExecContext(ctx, `ALTER TABLE example ADD COLUMN name TEXT`)
					return err
				},
				Down: func(ctx context.Context, db *sqlx.DB) error {
					_, err := db.ExecContext(ctx, `ALTER TABLE example DROP COLUMN name`)
					return err
				},
			},
		},
		{
			Version: 1,
			UpDown: &UpDownSQL{
				Up: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(
						ctx, `CREATE TABLE example (id SERIAL PRIMARY KEY)`)
					return err
				},
				Down: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, `DROP TABLE example`)
					return err
				},
			},
		},
	}

	// NOTE: In a real-world scenario, all migrations would typically be in a
	// single slice.
	// But things should still work fine for "up" command even if only the new
	// migrations are provided.
	migrations2 := []MigrationSQLX{
		{
			Version: 3,
			UpDown: &UpDownSQL{
				Up: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, `ALTER TABLE example ADD COLUMN age INT`)
					return err
				},
				Down: func(ctx context.Context, tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, `ALTER TABLE example DROP COLUMN age`)
					return err
				},
			},
		},
	}

	testGosmig(t, db, runCmd, &outW, &errW, migrations1, migrations2)
}

func testGosmig[TDB DB[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions]](
	t *testing.T,
	db TDB,
	runCmd func(migrations []Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, TDB], command string),
	outW, errW *strings.Builder,
	migrations1 []Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, TDB],
	migrations2 []Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, TDB],
) {

	ctx := context.Background()

	// -- 1st run - up

	runCmd(migrations1, "up")

	expectedOut := "[x] Applied migration version 1\n" +
		"[x] Applied migration version 2\n" +
		"2 migration(s) applied\n"
	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
	checkDBTables(ctx, t, db, []int{1, 2}, []string{"id", "name"})

	// -- 2nd run - up

	runCmd(migrations2, "up")

	expectedOut = "[x] Applied migration version 3\n" +
		"1 migration(s) applied\n"
	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
	checkDBTables(ctx, t, db, []int{1, 2, 3}, []string{"id", "name", "age"})

	// -- 3rd run - version

	runCmd(migrations1, "version") // migrations don't matter for "version" command

	expectedOut = "Current database version:\n3\n"

	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())

	// -- 4th run - up with no new migrations

	allAppliedMigrations := append(migrations1, migrations2...)
	runCmd(allAppliedMigrations, "up")

	expectedOut = "No migrations to apply\n"

	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
	checkDBTables(ctx, t, db, []int{1, 2, 3}, []string{"id", "name", "age"})

	// -- 5th run - down

	runCmd(allAppliedMigrations, "down")

	expectedOut = "[x]-->[ ] Rolled back migration version 3\n"
	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
	checkDBTables(ctx, t, db, []int{1, 2}, []string{"id", "name"})

	// -- 6th run - status

	runCmd(allAppliedMigrations, "status")

	expectedOut = "VERSION    STATUS      \n" +
		"3          [ ] PENDING \n" +
		"2          [x] APPLIED \n" +
		"1          [x] APPLIED \n"

	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())

	// -- 7th run - version after down

	runCmd(allAppliedMigrations, "version")

	expectedOut = "Current database version:\n2\n"

	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())

	// -- 8th run - up-one

	runCmd(allAppliedMigrations, "up-one")

	expectedOut = "[x] Applied migration version 3\n" +
		"1 migration(s) applied\n"
	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
	checkDBTables(ctx, t, db, []int{1, 2, 3}, []string{"id", "name", "age"})

	// -- 9th run - up-one with no new migrations

	runCmd(allAppliedMigrations, "up-one")

	expectedOut = "No migrations to apply\n"

	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
	checkDBTables(ctx, t, db, []int{1, 2, 3}, []string{"id", "name", "age"})

	// -- 10th run - version after up-one

	runCmd(allAppliedMigrations, "version")

	expectedOut = "Current database version:\n3\n"

	require.Equal(t, expectedOut, outW.String())
	require.Empty(t, errW.String())
}

func checkDBTables[TDB DB[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions]](
	ctx context.Context,
	t *testing.T,
	db TDB,
	expectedVersions []int,
	expectedTableColumns []string,
) {

	var err error
	ctx1, cancel1 := context.WithTimeout(ctx, defaultTimeout)
	defer cancel1()
	rowVersion := db.QueryRowContext(
		ctx1, `SELECT version FROM gosmig ORDER BY version ASC`)
	versions := []int{}
	var v int
	for {
		if err = rowVersion.Scan(&v); err == sql.ErrNoRows {
			break
		}
		require.NoError(t, err)
		versions = append(versions, v)
		rowVersion = db.QueryRowContext(
			ctx1,
			`SELECT version FROM gosmig WHERE version > $1 ORDER BY version ASC`,
			v)
	}
	require.Equal(t, expectedVersions, versions)

	// Check that the "example" table exists with the expected columns.
	// NOTE: This is Postgres-specific SQL.
	ctx2, cancel2 := context.WithTimeout(ctx, defaultTimeout)
	defer cancel2()
	rowExample := db.QueryRowContext(ctx2, `
		SELECT column_name, ordinal_position
		FROM information_schema.columns
		WHERE table_name='example'
		ORDER BY ordinal_position`)
	columns := []string{}
	var col string
	var pos int
	for {
		if err = rowExample.Scan(&col, &pos); err == sql.ErrNoRows {
			break
		}
		require.NoError(t, err)
		columns = append(columns, col)
		rowExample = db.QueryRowContext(ctx2, `
			SELECT column_name, ordinal_position
			FROM information_schema.columns
			WHERE table_name='example' AND ordinal_position > $1
			ORDER BY ordinal_position`, pos)
	}
	require.Equal(t, expectedTableColumns, columns)
}
