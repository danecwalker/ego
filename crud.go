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
func Create[T any](ex Executor, ctx context.Context, entity *T) error {
	if entity == nil {
		return fmt.Errorf("ego: Create: entity must not be nil")
	}

	// Look up the schema for T, auto-registering if needed.
	t := reflect.TypeOf((*T)(nil)).Elem()
	schema := ex.schemaFor(t)
	if schema == nil {
		var err error
		schema, err = parseAndRegister(ex, entity)
		if err != nil {
			return fmt.Errorf("ego: Create: %w", err)
		}
	}

	// Run BeforeCreate hook if the entity implements it.
	if hook, ok := any(entity).(BeforeCreator); ok {
		if err := hook.BeforeCreate(ctx); err != nil {
			return fmt.Errorf("ego: Create: BeforeCreate hook: %w", err)
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

	d := ex.dialect()

	for _, col := range schema.Columns {
		if schema.PrimaryKey != nil && col.DBName == schema.PrimaryKey.DBName {
			continue
		}
		columns = append(columns, d.QuoteIdentifier(col.DBName))
		placeholders = append(placeholders, d.Placeholder(placeholderIdx))
		args = append(args, entityVal.FieldByIndex(col.Index).Interface())
		placeholderIdx++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.QuoteIdentifier(schema.TableName),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Build the actual executor (the innermost handler).
	execute := func(op *Operation) error {
		result, err := ex.ExecContext(ctx, op.SQL, op.Args...)
		if err != nil {
			return err
		}
		// Set the ID back on the entity using LastInsertId.
		if schema.PrimaryKey != nil {
			lastID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert id: %w", err)
			}
			entityVal.FieldByIndex(schema.PrimaryKey.Index).SetInt(lastID)
		}
		return nil
	}

	// Build the middleware chain (reverse order so first-registered is outermost).
	handler := buildMiddlewareChain(ex.getMiddlewares(), execute)

	// Execute through the chain.
	op := &Operation{Type: "create", Entity: entity, SQL: query, Args: args}
	if err := handler(op); err != nil {
		return fmt.Errorf("ego: Create: %w", err)
	}

	// Run AfterCreate hook if the entity implements it.
	if hook, ok := any(entity).(AfterCreator); ok {
		if err := hook.AfterCreate(ctx); err != nil {
			return fmt.Errorf("ego: Create: AfterCreate hook: %w", err)
		}
	}

	return nil
}

// Update modifies an existing entity in the database. It locates the row by
// the entity's primary key, sets UpdatedAt to now (if the field exists), and
// builds an UPDATE ... SET ... WHERE id=? statement. The primary key must be
// non-zero; otherwise an error is returned.
func Update[T any](ex Executor, ctx context.Context, entity *T) error {
	if entity == nil {
		return fmt.Errorf("ego: Update: entity must not be nil")
	}

	// Look up the schema for T, auto-registering if needed.
	t := reflect.TypeOf((*T)(nil)).Elem()
	schema := ex.schemaFor(t)
	if schema == nil {
		var err error
		schema, err = parseAndRegister(ex, entity)
		if err != nil {
			return fmt.Errorf("ego: Update: %w", err)
		}
	}

	// The entity must have a primary key defined, and its value must be non-zero.
	if schema.PrimaryKey == nil {
		return fmt.Errorf("ego: Update: entity has no primary key")
	}

	entityVal := reflect.ValueOf(entity).Elem()
	pkVal := entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
	if pkVal == 0 {
		return fmt.Errorf("ego: Update: primary key is zero")
	}

	// Run BeforeUpdate hook if the entity implements it.
	if hook, ok := any(entity).(BeforeUpdater); ok {
		if err := hook.BeforeUpdate(ctx); err != nil {
			return fmt.Errorf("ego: Update: BeforeUpdate hook: %w", err)
		}
	}

	// Set UpdatedAt to now if the entity has that field.
	now := time.Now()
	for _, col := range schema.Columns {
		if col.GoType == reflect.TypeOf(time.Time{}) && col.DBName == "updated_at" {
			entityVal.FieldByIndex(col.Index).Set(reflect.ValueOf(now))
		}
	}

	// Build the UPDATE statement, skipping the primary key in the SET clause.
	var setClauses []string
	var args []any
	placeholderIdx := 1

	d := ex.dialect()

	for _, col := range schema.Columns {
		if col.DBName == schema.PrimaryKey.DBName {
			continue
		}
		setClauses = append(setClauses,
			fmt.Sprintf("%s = %s", d.QuoteIdentifier(col.DBName), d.Placeholder(placeholderIdx)),
		)
		args = append(args, entityVal.FieldByIndex(col.Index).Interface())
		placeholderIdx++
	}

	// Append the primary key value as the final WHERE argument.
	args = append(args, pkVal)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		d.QuoteIdentifier(schema.TableName),
		strings.Join(setClauses, ", "),
		d.QuoteIdentifier(schema.PrimaryKey.DBName),
		d.Placeholder(placeholderIdx),
	)

	// Build the actual executor (the innermost handler).
	execute := func(op *Operation) error {
		_, err := ex.ExecContext(ctx, op.SQL, op.Args...)
		return err
	}

	// Build the middleware chain.
	handler := buildMiddlewareChain(ex.getMiddlewares(), execute)

	// Execute through the chain.
	op := &Operation{Type: "update", Entity: entity, SQL: query, Args: args}
	if err := handler(op); err != nil {
		return fmt.Errorf("ego: Update: %w", err)
	}

	// Run AfterUpdate hook if the entity implements it.
	if hook, ok := any(entity).(AfterUpdater); ok {
		if err := hook.AfterUpdate(ctx); err != nil {
			return fmt.Errorf("ego: Update: AfterUpdate hook: %w", err)
		}
	}

	return nil
}

// Delete removes an entity from the database by its primary key. The primary
// key must be non-zero; otherwise an error is returned.
func Delete[T any](ex Executor, ctx context.Context, entity *T) error {
	if entity == nil {
		return fmt.Errorf("ego: Delete: entity must not be nil")
	}

	// Look up the schema for T, auto-registering if needed.
	t := reflect.TypeOf((*T)(nil)).Elem()
	schema := ex.schemaFor(t)
	if schema == nil {
		var err error
		schema, err = parseAndRegister(ex, entity)
		if err != nil {
			return fmt.Errorf("ego: Delete: %w", err)
		}
	}

	// The entity must have a primary key defined, and its value must be non-zero.
	if schema.PrimaryKey == nil {
		return fmt.Errorf("ego: Delete: entity has no primary key")
	}

	entityVal := reflect.ValueOf(entity).Elem()
	pkVal := entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
	if pkVal == 0 {
		return fmt.Errorf("ego: Delete: primary key is zero")
	}

	// Run BeforeDelete hook if the entity implements it.
	if hook, ok := any(entity).(BeforeDeleter); ok {
		if err := hook.BeforeDelete(ctx); err != nil {
			return fmt.Errorf("ego: Delete: BeforeDelete hook: %w", err)
		}
	}

	d := ex.dialect()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		d.QuoteIdentifier(schema.TableName),
		d.QuoteIdentifier(schema.PrimaryKey.DBName),
		d.Placeholder(1),
	)

	args := []any{pkVal}

	// Build the actual executor (the innermost handler).
	execute := func(op *Operation) error {
		_, err := ex.ExecContext(ctx, op.SQL, op.Args...)
		return err
	}

	// Build the middleware chain.
	handler := buildMiddlewareChain(ex.getMiddlewares(), execute)

	// Execute through the chain.
	op := &Operation{Type: "delete", Entity: entity, SQL: query, Args: args}
	if err := handler(op); err != nil {
		return fmt.Errorf("ego: Delete: %w", err)
	}

	// Run AfterDelete hook if the entity implements it.
	if hook, ok := any(entity).(AfterDeleter); ok {
		if err := hook.AfterDelete(ctx); err != nil {
			return fmt.Errorf("ego: Delete: AfterDelete hook: %w", err)
		}
	}

	return nil
}
