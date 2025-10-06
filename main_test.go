package gosmig

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

const testDBURL = "postgres://gosmig:gosmig@localhost:5432/gosmig?sslmode=disable&search_path=public"

func TestMain(m *testing.M) {
	m.Run()
}

func connectToDB_StdLibSQL(url string, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open a database connection: %w", err)
	}

	ctxPing, cancelPing := context.WithTimeout(context.Background(), timeout)
	defer cancelPing()
	if err := db.PingContext(ctxPing); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}

	return db, nil
}

func connectToDB_SQLX(url string, timeout time.Duration) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open a database connection: %w", err)
	}

	ctxPing, cancelPing := context.WithTimeout(context.Background(), timeout)
	defer cancelPing()
	if err := db.PingContext(ctxPing); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}

	return db, nil
}

func dropTables[TDBRow DBRow, TDBResult DBResult, TDBOrTX DBOrTX[TDBRow, TDBResult]](
	t *testing.T,
	tables []string,
	db TDBOrTX,
	timeout time.Duration) {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table))
		assert.NoError(t, err, fmt.Sprintf("failed to drop table %q", table))
	}
}

func cleanup[
	TDBRow DBRow,
	TDBResult DBResult,
	TTX TX[TDBRow, TDBResult],
	TXO TXOptions,
	TDB DB[TDBRow, TDBResult, TTX, TXO]](

	t *testing.T, tables []string, db TDB,
) {

	dropTables(t, tables, db, defaultTimeout)
}
