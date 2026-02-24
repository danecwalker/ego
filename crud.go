package ego

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Create inserts a single entity into the database. It automatically sets
// CreatedAt and UpdatedAt timestamps (if the entity has those fields), builds
// an INSERT statement excluding the auto-increment primary key, executes it,
// and populates the entity's ID from the last insert ID.
func Create[T any](db *DB, ctx context.Context, entity *T) error {
	if entity == nil {
		return fmt.Errorf("ego: Create: entity must not be nil")
	}

	// Look up the schema for T, auto-registering if needed.
	t := reflect.TypeOf((*T)(nil)).Elem()
	schema, exists := db.schemas[t]
	if !exists {
		var err error
		schema, err = parseAndRegister(db, entity)
		if err != nil {
			return fmt.Errorf("ego: Create: %w", err)
		}
	}

	entityVal := reflect.ValueOf(entity).Elem()
	now := time.Now()

	// Set CreatedAt and UpdatedAt timestamps if the entity has those fields.
	for _, col := range schema.Columns {
		if col.GoType == reflect.TypeOf(time.Time{}) {
			switch col.DBName {
			case "created_at":
				entityVal.FieldByIndex(col.Index).Set(reflect.ValueOf(now))
			case "updated_at":
				entityVal.FieldByIndex(col.Index).Set(reflect.ValueOf(now))
			}
		}
	}

	// Build the INSERT statement, skipping the primary key column.
	var columns []string
	var placeholders []string
	var args []any
	placeholderIdx := 1

	for _, col := range schema.Columns {
		if schema.PrimaryKey != nil && col.DBName == schema.PrimaryKey.DBName {
			continue
		}
		columns = append(columns, db.dialect.QuoteIdentifier(col.DBName))
		placeholders = append(placeholders, db.dialect.Placeholder(placeholderIdx))
		args = append(args, entityVal.FieldByIndex(col.Index).Interface())
		placeholderIdx++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		db.dialect.QuoteIdentifier(schema.TableName),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Execute the INSERT.
	result, err := db.sqlDB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ego: Create: %w", err)
	}

	// Set the ID back on the entity using LastInsertId.
	if schema.PrimaryKey != nil {
		lastID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("ego: Create: failed to get last insert id: %w", err)
		}
		entityVal.FieldByIndex(schema.PrimaryKey.Index).SetInt(lastID)
	}

	return nil
}
