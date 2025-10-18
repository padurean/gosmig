package migrations

import (
	"database/sql"

	"github.com/padurean/gosmig"
)

type (
	MigrationSQL  = gosmig.Migration[*sql.Row, sql.Result, *sql.Tx, *sql.TxOptions, *LockedDB]
	UpDownNoTXSQL = gosmig.UpDown[*sql.Row, sql.Result, *LockedDB]
)

var Migrations = []MigrationSQL{
	Migration00001,
	Migration00002,
	Migration00003,
}
