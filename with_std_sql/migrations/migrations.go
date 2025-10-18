package migrations

import "github.com/padurean/gosmig"

var Migrations = []gosmig.MigrationSQL{
	Migration00001,
	Migration00002,
	Migration00003,
}
