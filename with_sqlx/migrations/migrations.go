package migrations

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/padurean/gosmig"
)

type (
	MigrationSQLX  = gosmig.Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *sqlx.DB]
	UpDownNoTXSQLX = gosmig.UpDown[*sql.Row, sql.Result, *sqlx.DB]
)

var Migrations = []MigrationSQLX{
	Migration00001,
	Migration00002,
	Migration00003,
}
