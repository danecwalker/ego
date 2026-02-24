// raw.go
package ego

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

// RawQuery holds a raw SQL query with type-safe scanning.
type RawQuery[T any] struct {
	ex   Executor
	ctx  context.Context
	sql  string
	args []any
}

// Raw creates a new raw SQL query that will scan results into type T.
func Raw[T any](ex Executor, ctx context.Context, query string, args ...any) *RawQuery[T] {
	return &RawQuery[T]{ex: ex, ctx: ctx, sql: query, args: args}
}

// Scan executes the query and scans results into the provided slice pointer.
func (r *RawQuery[T]) Scan(dest *[]T) error {
	// Get schema for T to know column->field mapping
	t := reflect.TypeOf((*T)(nil)).Elem()
	schema := r.ex.schemaFor(t)
	if schema == nil {
		// Auto-register
		entity := reflect.New(t).Interface()
		var err error
		schema, err = parseAndRegister(r.ex, entity)
		if err != nil {
			return fmt.Errorf("ego: Raw.Scan: %w", err)
		}
	}

	rows, err := r.ex.QueryContext(r.ctx, r.sql, r.args...)
	if err != nil {
		return fmt.Errorf("ego: Raw.Scan: %w", err)
	}
	defer rows.Close()

	// Get the columns returned by the query
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("ego: Raw.Scan: %w", err)
	}

	// Build column name -> schema index mapping
	// This handles cases where the raw SQL returns a subset of columns or different order
	colIndexMap := make(map[string]int)
	for i, col := range schema.Columns {
		colIndexMap[col.DBName] = i
	}

	for rows.Next() {
		entity := new(T)
		entityVal := reflect.ValueOf(entity).Elem()

		scanDest := make([]any, len(cols))
		for i, colName := range cols {
			if idx, ok := colIndexMap[colName]; ok {
				scanDest[i] = entityVal.FieldByIndex(schema.Columns[idx].Index).Addr().Interface()
			} else {
				// Column not in schema -- scan into a discard variable
				var discard any
				scanDest[i] = &discard
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("ego: Raw.Scan: %w", err)
		}
		*dest = append(*dest, *entity)
	}

	return rows.Err()
}

// RawExec executes a raw SQL statement (INSERT, UPDATE, DELETE, etc.)
// and returns the sql.Result.
func RawExec(ex Executor, ctx context.Context, query string, args ...any) (sql.Result, error) {
	return ex.ExecContext(ctx, query, args...)
}
