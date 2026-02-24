package ego

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// QueryBuilder provides a fluent API for constructing SELECT queries.
// It is parameterized by the entity type T.
type QueryBuilder[T any] struct {
	ex         Executor
	ctx        context.Context
	conditions []Condition
	orders     []OrderClause
	limit      int
	offset     int
	includes   []string // reserved for future relationship eager-loading (Task 16)
}

// Query creates a new QueryBuilder for the entity type T.
func Query[T any](ex Executor, ctx context.Context) *QueryBuilder[T] {
	return &QueryBuilder[T]{
		ex:  ex,
		ctx: ctx,
	}
}

// Where adds a condition to the query. Multiple conditions are combined with AND.
func (q *QueryBuilder[T]) Where(c Condition) *QueryBuilder[T] {
	q.conditions = append(q.conditions, c)
	return q
}

// OrderBy adds an ordering clause to the query.
func (q *QueryBuilder[T]) OrderBy(o OrderClause) *QueryBuilder[T] {
	q.orders = append(q.orders, o)
	return q
}

// Limit sets the maximum number of rows to return.
func (q *QueryBuilder[T]) Limit(n int) *QueryBuilder[T] {
	q.limit = n
	return q
}

// Offset sets the number of rows to skip before returning results.
func (q *QueryBuilder[T]) Offset(n int) *QueryBuilder[T] {
	q.offset = n
	return q
}

// Include marks a relationship for eager loading (reserved for Task 16).
func (q *QueryBuilder[T]) Include(relation string) *QueryBuilder[T] {
	q.includes = append(q.includes, relation)
	return q
}

// All executes the query and returns all matching entities.
func (q *QueryBuilder[T]) All() ([]T, error) {
	schema, err := q.resolveSchema()
	if err != nil {
		return nil, err
	}

	query, args := q.buildSelect(schema, false)

	rows, err := q.ex.QueryContext(q.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ego: Query.All: %w", err)
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		entity := new(T)
		entityVal := reflect.ValueOf(entity).Elem()
		scanDest := make([]any, len(schema.Columns))
		for i, col := range schema.Columns {
			scanDest[i] = entityVal.FieldByIndex(col.Index).Addr().Interface()
		}
		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("ego: Query.All: scan: %w", err)
		}
		results = append(results, *entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ego: Query.All: %w", err)
	}

	return results, nil
}

// First executes the query with LIMIT 1 and returns the first matching entity.
// Returns ErrNotFound if no rows match.
func (q *QueryBuilder[T]) First() (*T, error) {
	schema, err := q.resolveSchema()
	if err != nil {
		return nil, err
	}

	// Force limit 1 for First.
	origLimit := q.limit
	q.limit = 1
	query, args := q.buildSelect(schema, false)
	q.limit = origLimit

	row := q.ex.QueryRowContext(q.ctx, query, args...)

	entity := new(T)
	entityVal := reflect.ValueOf(entity).Elem()
	scanDest := make([]any, len(schema.Columns))
	for i, col := range schema.Columns {
		scanDest[i] = entityVal.FieldByIndex(col.Index).Addr().Interface()
	}

	if err := row.Scan(scanDest...); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ego: Query.First: %w", err)
	}

	return entity, nil
}

// Count executes a COUNT(*) query with the same WHERE conditions and returns
// the number of matching rows.
func (q *QueryBuilder[T]) Count() (int64, error) {
	schema, err := q.resolveSchema()
	if err != nil {
		return 0, err
	}

	query, args := q.buildSelect(schema, true)

	var count int64
	err = q.ex.QueryRowContext(q.ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ego: Query.Count: %w", err)
	}

	return count, nil
}

// resolveSchema looks up or auto-registers the schema for type T.
func (q *QueryBuilder[T]) resolveSchema() (*EntitySchema, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	schema := q.ex.schemaFor(t)
	if schema == nil {
		entity := new(T)
		var err error
		schema, err = parseAndRegister(q.ex, entity)
		if err != nil {
			return nil, fmt.Errorf("ego: Query: %w", err)
		}
	}
	return schema, nil
}

// buildSelect constructs the SQL SELECT statement and its argument list.
// If countOnly is true, it builds SELECT COUNT(*) instead of selecting columns.
func (q *QueryBuilder[T]) buildSelect(schema *EntitySchema, countOnly bool) (string, []any) {
	var sb strings.Builder
	var args []any
	placeholderIdx := 1

	d := q.ex.dialect()

	// SELECT clause
	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM ")
	} else {
		sb.WriteString("SELECT ")
		for i, col := range schema.Columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(d.QuoteIdentifier(col.DBName))
		}
		sb.WriteString(" FROM ")
	}

	sb.WriteString(d.QuoteIdentifier(schema.TableName))

	// WHERE clause
	if len(q.conditions) > 0 {
		sb.WriteString(" WHERE ")
		for i, cond := range q.conditions {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(d.QuoteIdentifier(cond.Column))
			sb.WriteString(" ")
			sb.WriteString(cond.Op)
			sb.WriteString(" ")
			sb.WriteString(d.Placeholder(placeholderIdx))
			args = append(args, cond.Value)
			placeholderIdx++
		}
	}

	// ORDER BY clause (skip for count queries)
	if !countOnly && len(q.orders) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, order := range q.orders {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(d.QuoteIdentifier(order.Column))
			sb.WriteString(" ")
			sb.WriteString(order.Dir)
		}
	}

	// LIMIT clause (skip for count queries)
	if !countOnly && q.limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", q.limit))
	}

	// OFFSET clause (skip for count queries)
	if !countOnly && q.offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", q.offset))
	}

	return sb.String(), args
}
