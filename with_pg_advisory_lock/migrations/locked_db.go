package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const lockID = "gosmig_advisory_lock_example"

type LockedDB struct {
	*sql.DB
	timeout time.Duration
}

func NewLockedDB(db *sql.DB, timeout time.Duration) (*LockedDB, error) {
	ldb := &LockedDB{
		DB:      db,
		timeout: timeout,
	}
	if err := ldb.acquireAdvisoryLock(); err != nil {
		return nil, err
	}

	return ldb, nil
}

func (ldb *LockedDB) Close() error {
	if err := ldb.releaseAdvisoryLock(); err != nil {
		return err
	}
	return ldb.DB.Close()
}

func (ldb *LockedDB) acquireAdvisoryLock() error {
	fmt.Println("Acquiring advisory lock...")

	ctx, cancel := context.WithTimeout(context.Background(), ldb.timeout)
	defer cancel()

	var acquired bool
	err := ldb.DB.
		QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", lockID).
		Scan(&acquired)
	if err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("failed to acquire advisory lock: already held")
	}

	fmt.Println("Acquired advisory lock.")
	return nil
}

func (ldb *LockedDB) releaseAdvisoryLock() error {
	fmt.Println("Releasing advisory lock...")

	ctx, cancel := context.WithTimeout(context.Background(), ldb.timeout)
	defer cancel()

	var released bool
	err := ldb.DB.
		QueryRowContext(ctx, "SELECT pg_advisory_unlock(hashtext($1))", lockID).
		Scan(&released)
	if err != nil {
		return fmt.Errorf("failed to release advisory lock: %w", err)
	}
	if !released {
		return fmt.Errorf("failed to release advisory lock: not held")
	}

	fmt.Println("Released advisory lock.")

	return nil
}
