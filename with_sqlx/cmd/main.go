package main

import (
	"context"
	"log"
	"time"
	"with_sqlx/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/padurean/gosmig"
)

func main() {
	migrate, err := gosmig.New(migrations.Migrations, connectToDB, nil)
	if err != nil {
		log.Fatalf("Failed to create migration tool: %v", err)
	}
	migrate()
}

func connectToDB(url string, timeout time.Duration) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", url)
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
