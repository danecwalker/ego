package ego

// Dialect abstracts database-specific SQL generation.
type Dialect interface {
	// Name returns the dialect identifier (e.g. "postgres", "sqlite").
	Name() string

	// Placeholder returns the parameter placeholder for the given 1-based index.
	// PostgreSQL: "$1", "$2". SQLite: "?", "?".
	Placeholder(index int) string

	// QuoteIdentifier quotes a table or column name.
	QuoteIdentifier(name string) string

	// AutoIncrementDef returns the column definition fragment for auto-increment.
	AutoIncrementDef() string

	// TypeMapping maps a Go type name to the SQL column type.
	TypeMapping(goType string) string

	// SupportsReturning reports whether the dialect supports RETURNING clauses.
	SupportsReturning() bool
}
