package migrate

import (
	"fmt"
	"strings"

	"github.com/danewilson/ego"
)

// Schema collects DDL statements to be executed during a migration.
type Schema struct {
	db         *ego.DB
	statements []string
}

// CreateTable defines a new table using the provided builder function.
func (s *Schema) CreateTable(name string, fn func(t *Table)) {
	t := &Table{name: name, dialect: s.db.Dialect()}
	fn(t)
	s.statements = append(s.statements, t.toSQL())
}

// DropTable adds a DROP TABLE IF EXISTS statement.
func (s *Schema) DropTable(name string) {
	s.statements = append(s.statements, fmt.Sprintf("DROP TABLE IF EXISTS %s", s.db.Dialect().QuoteIdentifier(name)))
}

// Table represents a table being defined in a migration.
type Table struct {
	name    string
	dialect ego.Dialect
	columns []columnDef
}

type columnDef struct {
	name       string
	sqlType    string
	primaryKey bool
	autoInc    bool
	notNull    bool
	unique     bool
	defaultVal *string
}

// BigInt adds a big integer column.
// Maps to INTEGER for SQLite, BIGINT for PostgreSQL.
func (t *Table) BigInt(name string) *columnDef {
	c := &columnDef{
		name:    name,
		sqlType: t.dialect.TypeMapping("int64"),
	}
	t.columns = append(t.columns, *c)
	return &t.columns[len(t.columns)-1]
}

// Int adds an integer column.
// Maps to INTEGER for both SQLite and PostgreSQL.
func (t *Table) Int(name string) *columnDef {
	c := &columnDef{
		name:    name,
		sqlType: t.dialect.TypeMapping("int"),
	}
	t.columns = append(t.columns, *c)
	return &t.columns[len(t.columns)-1]
}

// String adds a varchar/text column with a maximum length.
// For SQLite this maps to TEXT; for PostgreSQL it maps to VARCHAR(maxLen).
func (t *Table) String(name string, maxLen int) *columnDef {
	var sqlType string
	if t.dialect.Name() == "postgres" {
		sqlType = fmt.Sprintf("VARCHAR(%d)", maxLen)
	} else {
		// SQLite and others: TEXT (SQLite ignores length constraints)
		sqlType = t.dialect.TypeMapping("string")
	}
	c := &columnDef{
		name:    name,
		sqlType: sqlType,
	}
	t.columns = append(t.columns, *c)
	return &t.columns[len(t.columns)-1]
}

// Text adds a text column (unlimited length).
func (t *Table) Text(name string) *columnDef {
	c := &columnDef{
		name:    name,
		sqlType: t.dialect.TypeMapping("string"),
	}
	t.columns = append(t.columns, *c)
	return &t.columns[len(t.columns)-1]
}

// Bool adds a boolean column.
// Maps to INTEGER for SQLite, BOOLEAN for PostgreSQL.
func (t *Table) Bool(name string) *columnDef {
	c := &columnDef{
		name:    name,
		sqlType: t.dialect.TypeMapping("bool"),
	}
	t.columns = append(t.columns, *c)
	return &t.columns[len(t.columns)-1]
}

// Timestamps adds created_at and updated_at DATETIME columns.
func (t *Table) Timestamps() {
	timeType := t.dialect.TypeMapping("time.Time")
	t.columns = append(t.columns, columnDef{
		name:    "created_at",
		sqlType: timeType,
	})
	t.columns = append(t.columns, columnDef{
		name:    "updated_at",
		sqlType: timeType,
	})
}

// PrimaryKey marks this column as the primary key.
func (c *columnDef) PrimaryKey() *columnDef {
	c.primaryKey = true
	return c
}

// AutoIncrement marks this column as auto-incrementing.
func (c *columnDef) AutoIncrement() *columnDef {
	c.autoInc = true
	return c
}

// NotNull marks this column as NOT NULL.
func (c *columnDef) NotNull() *columnDef {
	c.notNull = true
	return c
}

// Unique adds a UNIQUE constraint to this column.
func (c *columnDef) Unique() *columnDef {
	c.unique = true
	return c
}

// Default sets a default value for this column.
func (c *columnDef) Default(v string) *columnDef {
	c.defaultVal = &v
	return c
}

// toSQL generates the CREATE TABLE DDL statement for this table.
func (t *Table) toSQL() string {
	var b strings.Builder

	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(t.dialect.QuoteIdentifier(t.name))
	b.WriteString(" (\n")

	for i, col := range t.columns {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("  ")
		b.WriteString(t.columnToSQL(&col))
	}

	b.WriteString("\n)")
	return b.String()
}

// columnToSQL generates the DDL fragment for a single column definition.
func (t *Table) columnToSQL(col *columnDef) string {
	var parts []string

	// Column name
	parts = append(parts, t.dialect.QuoteIdentifier(col.name))

	// SQL type
	parts = append(parts, col.sqlType)

	// Primary key + auto-increment
	if col.primaryKey {
		parts = append(parts, "PRIMARY KEY")
		if col.autoInc {
			autoinc := t.dialect.AutoIncrementDef()
			if autoinc != "" {
				parts = append(parts, autoinc)
			}
		}
	}

	// NOT NULL (skip for PK since it's implicitly NOT NULL)
	if !col.primaryKey && col.notNull {
		parts = append(parts, "NOT NULL")
	}

	// UNIQUE
	if col.unique {
		parts = append(parts, "UNIQUE")
	}

	// DEFAULT
	if col.defaultVal != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", *col.defaultVal))
	}

	return strings.Join(parts, " ")
}
