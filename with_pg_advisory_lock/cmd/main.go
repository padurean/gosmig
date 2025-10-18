package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
	"with_pg_advisory_lock/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/padurean/gosmig"
)

func main() {
	migrate, err := gosmig.New(migrations.Migrations, connectToDB, nil)
	if err != nil {
		log.Fatalf("Failed to create migration tool: %v", err)
	}
	migrate()
}

func connectToDB(url string, timeout time.Duration) (*migrations.LockedDB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ldb, err := migrations.NewLockedDB(db, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create locked database: %w", err)
	}

	return ldb, nil
}
