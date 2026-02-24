// tx.go
package ego

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sync"
)

// Tx wraps a *sql.Tx with ego metadata, implementing the Executor interface.
// A Tx shares the schema registry with its parent DB so that entity schemas
// registered outside the transaction are visible inside it, and vice versa.
type Tx struct {
	sqlTx       *sql.Tx
	dial        Dialect
	schemas     map[reflect.Type]*EntitySchema // shared reference with parent DB
	schemasMu   *sync.RWMutex                 // shared mutex with parent DB
	middlewares []MiddlewareFunc               // inherited from parent DB
}

// Executor interface methods — these allow *Tx to satisfy Executor.

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.sqlTx.ExecContext(ctx, query, args...)
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.sqlTx.QueryContext(ctx, query, args...)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.sqlTx.QueryRowContext(ctx, query, args...)
}

func (tx *Tx) dialect() Dialect { return tx.dial }

func (tx *Tx) schemaFor(t reflect.Type) *EntitySchema {
	tx.schemasMu.RLock()
	s := tx.schemas[t]
	tx.schemasMu.RUnlock()
	return s
}

func (tx *Tx) registerSchema(t reflect.Type, s *EntitySchema) {
	tx.schemasMu.Lock()
	tx.schemas[t] = s
	tx.schemasMu.Unlock()
}

func (tx *Tx) getMiddlewares() []MiddlewareFunc { return tx.middlewares }

// Transaction starts a database transaction and calls fn with the new Tx.
// If fn returns nil, the transaction is committed. If fn returns an error,
// the transaction is rolled back and the error is returned. If fn panics,
// the transaction is rolled back and the panic is re-raised.
func Transaction(db *DB, ctx context.Context, fn func(tx *Tx) error) (err error) {
	sqlTx, err := db.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ego: Transaction: begin: %w", err)
	}

	tx := &Tx{
		sqlTx:       sqlTx,
		dial:        db.dial,
		schemas:     db.schemas,       // share the same schema registry
		schemasMu:   &db.schemasMu,    // share the same mutex
		middlewares: db.middlewares,    // inherit parent's middlewares
	}

	defer func() {
		if p := recover(); p != nil {
			sqlTx.Rollback()
			panic(p) // re-panic after rollback
		}
	}()

	if err = fn(tx); err != nil {
		sqlTx.Rollback()
		return err
	}

	if commitErr := sqlTx.Commit(); commitErr != nil {
		return fmt.Errorf("ego: Transaction: commit: %w", commitErr)
	}
	return nil
}
