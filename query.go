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
	includes   []string // relationship field names to eager-load via Include
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

// Include marks a relationship for eager loading. The relation argument must
// match a Go field name that was registered via HasMany or BelongsTo in the
// entity's Configure method. Included relationships are loaded in a second
// query using an IN clause after the primary entities are fetched.
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

	// Eager-load included relationships.
	if len(q.includes) > 0 && len(results) > 0 {
		if err := q.loadIncludes(schema, results); err != nil {
			return nil, err
		}
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

	// Eager-load included relationships.
	if len(q.includes) > 0 {
		results := []T{*entity}
		if err := q.loadIncludes(schema, results); err != nil {
			return nil, err
		}
		*entity = results[0]
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

// loadIncludes processes all eager-load includes for the given results slice.
// It modifies results in-place by setting relationship fields.
func (q *QueryBuilder[T]) loadIncludes(schema *EntitySchema, results []T) error {
	for _, inc := range q.includes {
		rel := findRelationship(schema, inc)
		if rel == nil {
			return fmt.Errorf("ego: Include(%q): no such relationship on %s", inc, schema.GoType.Name())
		}

		switch rel.Type {
		case HasManyRel:
			if err := q.loadHasMany(schema, rel, results); err != nil {
				return err
			}
		case BelongsToRel:
			if err := q.loadBelongsTo(schema, rel, results); err != nil {
				return err
			}
		case HasOneRel:
			if err := q.loadHasOne(schema, rel, results); err != nil {
				return err
			}
		case ManyToManyRel:
			if err := q.loadManyToMany(schema, rel, results); err != nil {
				return err
			}
		default:
			return fmt.Errorf("ego: Include(%q): unsupported relationship type %d", inc, rel.Type)
		}
	}
	return nil
}

// findRelationship looks up a RelationshipSchema by Go field name.
func findRelationship(schema *EntitySchema, fieldName string) *RelationshipSchema {
	for i := range schema.Relationships {
		if schema.Relationships[i].FieldName == fieldName {
			return &schema.Relationships[i]
		}
	}
	return nil
}

// loadHasMany implements eager loading for HasMany relationships.
// It collects parent IDs, queries the related table with an IN clause,
// groups results by FK, and sets the slice field on each parent entity.
func (q *QueryBuilder[T]) loadHasMany(schema *EntitySchema, rel *RelationshipSchema, results []T) error {
	if schema.PrimaryKey == nil {
		return fmt.Errorf("ego: Include: parent entity %s has no primary key", schema.GoType.Name())
	}

	// Resolve the related entity's schema.
	relSchema, err := resolveSchemaForType(q.ex, rel.RelatedType)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}

	// Collect parent IDs.
	parentIDs := make([]any, len(results))
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		parentIDs[i] = entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
	}

	// Build: SELECT ... FROM related_table WHERE fk_col IN (?, ?, ?)
	d := q.ex.dialect()
	var sb strings.Builder
	sb.WriteString("SELECT ")
	for i, col := range relSchema.Columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(d.QuoteIdentifier(col.DBName))
	}
	sb.WriteString(" FROM ")
	sb.WriteString(d.QuoteIdentifier(relSchema.TableName))
	sb.WriteString(" WHERE ")
	sb.WriteString(d.QuoteIdentifier(rel.ForeignKey))
	sb.WriteString(" IN (")
	placeholders := make([]string, len(parentIDs))
	for i := range parentIDs {
		placeholders[i] = d.Placeholder(i + 1)
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	rows, err := q.ex.QueryContext(q.ctx, sb.String(), parentIDs...)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): query: %w", rel.FieldName, err)
	}
	defer rows.Close()

	// Find the FK column index in the related schema so we can group by it.
	fkColIdx := -1
	for i, col := range relSchema.Columns {
		if col.DBName == rel.ForeignKey {
			fkColIdx = i
			break
		}
	}
	if fkColIdx < 0 {
		return fmt.Errorf("ego: Include(%q): FK column %q not found in %s schema",
			rel.FieldName, rel.ForeignKey, relSchema.GoType.Name())
	}

	// Scan results and group by FK value.
	// grouped maps parent_id -> []reflect.Value (each value is a related entity struct).
	grouped := make(map[int64][]reflect.Value)
	for rows.Next() {
		relEntity := reflect.New(rel.RelatedType).Elem()
		scanDest := make([]any, len(relSchema.Columns))
		for i, col := range relSchema.Columns {
			scanDest[i] = relEntity.FieldByIndex(col.Index).Addr().Interface()
		}
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("ego: Include(%q): scan: %w", rel.FieldName, err)
		}
		fkVal := relEntity.FieldByIndex(relSchema.Columns[fkColIdx].Index).Int()
		grouped[fkVal] = append(grouped[fkVal], relEntity)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}

	// Set the slice field on each parent entity.
	sliceType := reflect.SliceOf(rel.RelatedType)
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		parentID := entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
		children := grouped[parentID]

		// Always set a non-nil slice (empty, not nil).
		slice := reflect.MakeSlice(sliceType, len(children), len(children))
		for j, child := range children {
			slice.Index(j).Set(child)
		}
		entityVal.FieldByIndex(rel.FieldIndex).Set(slice)
	}

	return nil
}

// loadBelongsTo implements eager loading for BelongsTo relationships.
// It collects FK values from the primary entities, queries the related table
// with an IN clause, indexes results by ID, and sets the pointer field on each
// primary entity.
func (q *QueryBuilder[T]) loadBelongsTo(schema *EntitySchema, rel *RelationshipSchema, results []T) error {
	// Resolve the related entity's schema.
	relSchema, err := resolveSchemaForType(q.ex, rel.RelatedType)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}
	if relSchema.PrimaryKey == nil {
		return fmt.Errorf("ego: Include(%q): related entity %s has no primary key",
			rel.FieldName, relSchema.GoType.Name())
	}

	// Find the FK column on the primary entity's schema.
	var fkCol *ColumnSchema
	for i := range schema.Columns {
		if schema.Columns[i].DBName == rel.ForeignKey {
			fkCol = &schema.Columns[i]
			break
		}
	}
	if fkCol == nil {
		return fmt.Errorf("ego: Include(%q): FK column %q not found in %s schema",
			rel.FieldName, rel.ForeignKey, schema.GoType.Name())
	}

	// Collect unique FK values from the primary entities.
	fkSet := make(map[int64]bool)
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		fkVal := entityVal.FieldByIndex(fkCol.Index).Int()
		if fkVal != 0 {
			fkSet[fkVal] = true
		}
	}
	if len(fkSet) == 0 {
		return nil // no FKs to load
	}

	fkValues := make([]any, 0, len(fkSet))
	for id := range fkSet {
		fkValues = append(fkValues, id)
	}

	// Build: SELECT ... FROM related_table WHERE id IN (?, ?, ?)
	d := q.ex.dialect()
	var sb strings.Builder
	sb.WriteString("SELECT ")
	for i, col := range relSchema.Columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(d.QuoteIdentifier(col.DBName))
	}
	sb.WriteString(" FROM ")
	sb.WriteString(d.QuoteIdentifier(relSchema.TableName))
	sb.WriteString(" WHERE ")
	sb.WriteString(d.QuoteIdentifier(relSchema.PrimaryKey.DBName))
	sb.WriteString(" IN (")
	placeholders := make([]string, len(fkValues))
	for i := range fkValues {
		placeholders[i] = d.Placeholder(i + 1)
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	rows, err := q.ex.QueryContext(q.ctx, sb.String(), fkValues...)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): query: %w", rel.FieldName, err)
	}
	defer rows.Close()

	// Scan and index by primary key.
	indexed := make(map[int64]reflect.Value)
	for rows.Next() {
		relEntity := reflect.New(rel.RelatedType) // *RelatedType
		relEntityElem := relEntity.Elem()          // RelatedType
		scanDest := make([]any, len(relSchema.Columns))
		for i, col := range relSchema.Columns {
			scanDest[i] = relEntityElem.FieldByIndex(col.Index).Addr().Interface()
		}
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("ego: Include(%q): scan: %w", rel.FieldName, err)
		}
		pkVal := relEntityElem.FieldByIndex(relSchema.PrimaryKey.Index).Int()
		indexed[pkVal] = relEntity // store as *RelatedType
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}

	// Set the pointer field on each primary entity.
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		fkVal := entityVal.FieldByIndex(fkCol.Index).Int()
		if related, ok := indexed[fkVal]; ok {
			entityVal.FieldByIndex(rel.FieldIndex).Set(related)
		}
	}

	return nil
}

// loadHasOne implements eager loading for HasOne relationships.
// Similar to HasMany but the FK is on the related entity, and we set a pointer
// (not a slice) on the parent. If no related entity exists, the pointer stays nil.
func (q *QueryBuilder[T]) loadHasOne(schema *EntitySchema, rel *RelationshipSchema, results []T) error {
	if schema.PrimaryKey == nil {
		return fmt.Errorf("ego: Include: parent entity %s has no primary key", schema.GoType.Name())
	}

	// Resolve the related entity's schema.
	relSchema, err := resolveSchemaForType(q.ex, rel.RelatedType)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}

	// Collect parent IDs.
	parentIDs := make([]any, len(results))
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		parentIDs[i] = entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
	}

	// Build: SELECT ... FROM related_table WHERE fk_col IN (?, ?, ?)
	d := q.ex.dialect()
	var sb strings.Builder
	sb.WriteString("SELECT ")
	for i, col := range relSchema.Columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(d.QuoteIdentifier(col.DBName))
	}
	sb.WriteString(" FROM ")
	sb.WriteString(d.QuoteIdentifier(relSchema.TableName))
	sb.WriteString(" WHERE ")
	sb.WriteString(d.QuoteIdentifier(rel.ForeignKey))
	sb.WriteString(" IN (")
	placeholders := make([]string, len(parentIDs))
	for i := range parentIDs {
		placeholders[i] = d.Placeholder(i + 1)
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	rows, err := q.ex.QueryContext(q.ctx, sb.String(), parentIDs...)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): query: %w", rel.FieldName, err)
	}
	defer rows.Close()

	// Find the FK column index in the related schema so we can index by it.
	fkColIdx := -1
	for i, col := range relSchema.Columns {
		if col.DBName == rel.ForeignKey {
			fkColIdx = i
			break
		}
	}
	if fkColIdx < 0 {
		return fmt.Errorf("ego: Include(%q): FK column %q not found in %s schema",
			rel.FieldName, rel.ForeignKey, relSchema.GoType.Name())
	}

	// Scan results and index by FK value.
	// indexed maps parent_id -> *RelatedType (as reflect.Value pointer).
	indexed := make(map[int64]reflect.Value)
	for rows.Next() {
		relEntity := reflect.New(rel.RelatedType) // *RelatedType
		relEntityElem := relEntity.Elem()          // RelatedType
		scanDest := make([]any, len(relSchema.Columns))
		for i, col := range relSchema.Columns {
			scanDest[i] = relEntityElem.FieldByIndex(col.Index).Addr().Interface()
		}
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("ego: Include(%q): scan: %w", rel.FieldName, err)
		}
		fkVal := relEntityElem.FieldByIndex(relSchema.Columns[fkColIdx].Index).Int()
		indexed[fkVal] = relEntity // store as *RelatedType
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}

	// Set the pointer field on each parent entity (or leave nil if no match).
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		parentID := entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
		if related, ok := indexed[parentID]; ok {
			entityVal.FieldByIndex(rel.FieldIndex).Set(related)
		}
	}

	return nil
}

// loadManyToMany implements eager loading for ManyToMany relationships.
// It queries the pivot table to find associations, then loads the related
// entities and assigns them to slice fields on each parent.
func (q *QueryBuilder[T]) loadManyToMany(schema *EntitySchema, rel *RelationshipSchema, results []T) error {
	if schema.PrimaryKey == nil {
		return fmt.Errorf("ego: Include: parent entity %s has no primary key", schema.GoType.Name())
	}

	// Resolve the related entity's schema.
	relSchema, err := resolveSchemaForType(q.ex, rel.RelatedType)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): %w", rel.FieldName, err)
	}
	if relSchema.PrimaryKey == nil {
		return fmt.Errorf("ego: Include(%q): related entity %s has no primary key",
			rel.FieldName, relSchema.GoType.Name())
	}

	// Collect parent IDs.
	parentIDs := make([]any, len(results))
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		parentIDs[i] = entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
	}

	d := q.ex.dialect()

	// Step 1: Query pivot table to get associations.
	// SELECT self_fk, other_fk FROM pivot_table WHERE self_fk IN (?, ?, ?)
	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(d.QuoteIdentifier(rel.PivotFKSelf))
	sb.WriteString(", ")
	sb.WriteString(d.QuoteIdentifier(rel.PivotFKOther))
	sb.WriteString(" FROM ")
	sb.WriteString(d.QuoteIdentifier(rel.PivotTable))
	sb.WriteString(" WHERE ")
	sb.WriteString(d.QuoteIdentifier(rel.PivotFKSelf))
	sb.WriteString(" IN (")
	placeholders := make([]string, len(parentIDs))
	for i := range parentIDs {
		placeholders[i] = d.Placeholder(i + 1)
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	pivotRows, err := q.ex.QueryContext(q.ctx, sb.String(), parentIDs...)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): pivot query: %w", rel.FieldName, err)
	}
	defer pivotRows.Close()

	// Build maps: parent_id -> []related_id, and collect all unique related IDs.
	pivotMap := make(map[int64][]int64)
	relatedIDSet := make(map[int64]bool)
	for pivotRows.Next() {
		var selfFK, otherFK int64
		if err := pivotRows.Scan(&selfFK, &otherFK); err != nil {
			return fmt.Errorf("ego: Include(%q): pivot scan: %w", rel.FieldName, err)
		}
		pivotMap[selfFK] = append(pivotMap[selfFK], otherFK)
		relatedIDSet[otherFK] = true
	}
	if err := pivotRows.Err(); err != nil {
		return fmt.Errorf("ego: Include(%q): pivot: %w", rel.FieldName, err)
	}

	// If no associations found, set empty slices on all parents and return.
	sliceType := reflect.SliceOf(rel.RelatedType)
	if len(relatedIDSet) == 0 {
		for i := range results {
			entityVal := reflect.ValueOf(&results[i]).Elem()
			entityVal.FieldByIndex(rel.FieldIndex).Set(reflect.MakeSlice(sliceType, 0, 0))
		}
		return nil
	}

	// Step 2: Query the related table for all referenced entities.
	relatedIDs := make([]any, 0, len(relatedIDSet))
	for id := range relatedIDSet {
		relatedIDs = append(relatedIDs, id)
	}

	var sb2 strings.Builder
	sb2.WriteString("SELECT ")
	for i, col := range relSchema.Columns {
		if i > 0 {
			sb2.WriteString(", ")
		}
		sb2.WriteString(d.QuoteIdentifier(col.DBName))
	}
	sb2.WriteString(" FROM ")
	sb2.WriteString(d.QuoteIdentifier(relSchema.TableName))
	sb2.WriteString(" WHERE ")
	sb2.WriteString(d.QuoteIdentifier(relSchema.PrimaryKey.DBName))
	sb2.WriteString(" IN (")
	placeholders2 := make([]string, len(relatedIDs))
	for i := range relatedIDs {
		placeholders2[i] = d.Placeholder(i + 1)
	}
	sb2.WriteString(strings.Join(placeholders2, ", "))
	sb2.WriteString(")")

	relRows, err := q.ex.QueryContext(q.ctx, sb2.String(), relatedIDs...)
	if err != nil {
		return fmt.Errorf("ego: Include(%q): related query: %w", rel.FieldName, err)
	}
	defer relRows.Close()

	// Index related entities by their PK.
	indexed := make(map[int64]reflect.Value)
	for relRows.Next() {
		relEntity := reflect.New(rel.RelatedType).Elem()
		scanDest := make([]any, len(relSchema.Columns))
		for i, col := range relSchema.Columns {
			scanDest[i] = relEntity.FieldByIndex(col.Index).Addr().Interface()
		}
		if err := relRows.Scan(scanDest...); err != nil {
			return fmt.Errorf("ego: Include(%q): related scan: %w", rel.FieldName, err)
		}
		pkVal := relEntity.FieldByIndex(relSchema.PrimaryKey.Index).Int()
		indexed[pkVal] = relEntity
	}
	if err := relRows.Err(); err != nil {
		return fmt.Errorf("ego: Include(%q): related: %w", rel.FieldName, err)
	}

	// Step 3: Build slices for each parent and set the field.
	for i := range results {
		entityVal := reflect.ValueOf(&results[i]).Elem()
		parentID := entityVal.FieldByIndex(schema.PrimaryKey.Index).Int()
		relatedPKs := pivotMap[parentID]

		var children []reflect.Value
		for _, pk := range relatedPKs {
			if child, ok := indexed[pk]; ok {
				children = append(children, child)
			}
		}

		slice := reflect.MakeSlice(sliceType, len(children), len(children))
		for j, child := range children {
			slice.Index(j).Set(child)
		}
		entityVal.FieldByIndex(rel.FieldIndex).Set(slice)
	}

	return nil
}
