package main

import (
	"context"
	"database/sql"
	"log"
	"time"
	"with_std_sql/migrations"

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

func connectToDB(url string, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
