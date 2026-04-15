package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Querier is the common interface for *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Transactor defines the interface for executing code within a transaction.
type Transactor interface {
	WithTransaction(ctx context.Context, fn func(q Querier) error) error
}

// TxManager implements Transactor using a real *sql.DB.
type TxManager struct {
	db *sql.DB
}

// NewTxManager creates a new TxManager.
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// WithTransaction executes fn within a database transaction.
func (tm *TxManager) WithTransaction(ctx context.Context, fn func(q Querier) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
