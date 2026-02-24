package ego

// ColumnRef refers to a database column by name. Used for building queries.
type ColumnRef struct {
	name string
}

// Condition represents a WHERE clause condition.
type Condition struct {
	Column string
	Op     string
	Value  any
}

// OrderClause represents an ORDER BY clause.
type OrderClause struct {
	Column string
	Dir    string // "ASC" or "DESC"
}

// Col creates a column reference for use in query conditions and ordering.
func Col(name string) *ColumnRef {
	return &ColumnRef{name: name}
}

// Eq creates an equality condition (column = value).
func (c *ColumnRef) Eq(v any) Condition { return Condition{Column: c.name, Op: "=", Value: v} }

// Gt creates a greater-than condition (column > value).
func (c *ColumnRef) Gt(v any) Condition { return Condition{Column: c.name, Op: ">", Value: v} }

// Lt creates a less-than condition (column < value).
func (c *ColumnRef) Lt(v any) Condition { return Condition{Column: c.name, Op: "<", Value: v} }

// Gte creates a greater-than-or-equal condition (column >= value).
func (c *ColumnRef) Gte(v any) Condition { return Condition{Column: c.name, Op: ">=", Value: v} }

// Lte creates a less-than-or-equal condition (column <= value).
func (c *ColumnRef) Lte(v any) Condition { return Condition{Column: c.name, Op: "<=", Value: v} }

// Ne creates a not-equal condition (column != value).
func (c *ColumnRef) Ne(v any) Condition { return Condition{Column: c.name, Op: "!=", Value: v} }

// Like creates a LIKE pattern condition (column LIKE value).
func (c *ColumnRef) Like(v any) Condition {
	return Condition{Column: c.name, Op: "LIKE", Value: v}
}

// Asc creates an ascending order clause for this column.
func (c *ColumnRef) Asc() OrderClause { return OrderClause{Column: c.name, Dir: "ASC"} }

// Desc creates a descending order clause for this column.
func (c *ColumnRef) Desc() OrderClause { return OrderClause{Column: c.name, Dir: "DESC"} }
