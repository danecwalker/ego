// executor.go
package ego

import (
	"context"
	"database/sql"
	"reflect"
)

// Executor is the interface for executing database operations.
// Both *DB and *Tx implement this interface, allowing Create, Update,
// Delete, and Query to work transparently inside or outside transactions.
//
// The unexported methods (dialect, schemaFor, registerSchema) make this a
// package-internal interface — external code cannot implement it directly.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	dialect() Dialect
	schemaFor(t reflect.Type) *EntitySchema
	registerSchema(t reflect.Type, s *EntitySchema)
}
