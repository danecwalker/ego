package sqlite

import (
	"github.com/danewilson/ego"
	_ "modernc.org/sqlite" // register the sqlite driver
)

type sqliteDialect struct{}

func (d *sqliteDialect) Name() string                       { return "sqlite" }
func (d *sqliteDialect) Placeholder(index int) string       { return "?" }
func (d *sqliteDialect) QuoteIdentifier(name string) string { return `"` + name + `"` }
func (d *sqliteDialect) AutoIncrementDef() string           { return "AUTOINCREMENT" }
func (d *sqliteDialect) SupportsReturning() bool            { return false }

func (d *sqliteDialect) TypeMapping(goType string) string {
	switch goType {
	case "int64", "int", "int32", "int16", "int8":
		return "INTEGER"
	case "string":
		return "TEXT"
	case "bool":
		return "INTEGER" // SQLite has no bool type
	case "float64", "float32":
		return "REAL"
	case "time.Time":
		return "DATETIME"
	case "[]byte":
		return "BLOB"
	default:
		return "TEXT"
	}
}

// New returns a DriverConfig for SQLite with the given DSN.
// Common DSNs: ":memory:" for in-memory, "file:path.db" for file-based.
//
// For in-memory databases, MaxOpenConns is set to 1 to avoid issues
// with multiple connections each getting their own separate database.
func New(dsn string) ego.DriverConfig {
	return ego.DriverConfig{
		DriverName: "sqlite", // modernc.org/sqlite registers as "sqlite"
		DSN:        dsn,
		Dialect:    &sqliteDialect{},
	}
}
