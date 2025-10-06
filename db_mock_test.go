package gosmig

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// implements the DBRow interface
type dbRowMock struct {
	mock.Mock
}

func (r *dbRowMock) Scan(dest ...any) error {
	args := r.Called(dest)
	return args.Error(0)
}

func (r *dbRowMock) Err() error {
	args := r.Called()
	return args.Error(0)
}

// implements the DBResult interface
type dbResultMock struct {
	mock.Mock
}

func (r *dbResultMock) LastInsertId() (int64, error) {
	args := r.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (r *dbResultMock) RowsAffected() (int64, error) {
	args := r.Called()
	return args.Get(0).(int64), args.Error(1)
}

// implements the DBOrTX interface
type dbOrTxMock struct {
	mock.Mock
}

func (d *dbOrTxMock) QueryRowContext(ctx context.Context, query string, args ...any) *dbRowMock {
	calledArgs := d.Called(append([]any{ctx, query}, args...)...)
	return calledArgs.Get(0).(*dbRowMock)
}

func (d *dbOrTxMock) ExecContext(ctx context.Context, query string, args ...any) (*dbResultMock, error) {
	calledArgs := d.Called(append([]any{ctx, query}, args...)...)
	return calledArgs.Get(0).(*dbResultMock), calledArgs.Error(1)
}

// implements the TXOptions interface
type txOptionsMock struct{}

// implements the TX interface
type txMock struct {
	mock.Mock
}

func (t *txMock) QueryRowContext(ctx context.Context, query string, args ...any) *dbRowMock {
	calledArgs := t.Called(append([]any{ctx, query}, args...)...)
	return calledArgs.Get(0).(*dbRowMock)
}

func (t *txMock) ExecContext(ctx context.Context, query string, args ...any) (*dbResultMock, error) {
	calledArgs := t.Called(append([]any{ctx, query}, args...)...)
	return calledArgs.Get(0).(*dbResultMock), calledArgs.Error(1)
}

func (t *txMock) Commit() error {
	args := t.Called()
	return args.Error(0)
}

func (t *txMock) Rollback() error {
	args := t.Called()
	return args.Error(0)
}

// implements the DB interface
type dbMock struct {
	mock.Mock
}

func (d *dbMock) QueryRowContext(ctx context.Context, query string, args ...any) *dbRowMock {
	calledArgs := d.Called(append([]any{ctx, query}, args...)...)
	return calledArgs.Get(0).(*dbRowMock)
}

func (d *dbMock) ExecContext(ctx context.Context, query string, args ...any) (*dbResultMock, error) {
	calledArgs := d.Called(append([]any{ctx, query}, args...)...)
	return calledArgs.Get(0).(*dbResultMock), calledArgs.Error(1)
}

func (d *dbMock) BeginTx(ctx context.Context, opts txOptionsMock) (*txMock, error) {
	calledArgs := d.Called(ctx, opts)
	return calledArgs.Get(0).(*txMock), calledArgs.Error(1)
}

func (d *dbMock) Close() error {
	args := d.Called()
	return args.Error(0)
}

// Migration mock
type migrationMock = Migration[*dbRowMock, *dbResultMock, *txMock, txOptionsMock, *dbMock]
