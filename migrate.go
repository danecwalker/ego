package ego

import (
	"fmt"
	"reflect"
	"strings"
)

// AutoMigrate creates database tables for the given entities using
// CREATE TABLE IF NOT EXISTS DDL. Each entity's schema is parsed (applying
// any Configure customizations) and registered on the DB if not already present.
func AutoMigrate(db *DB, entities ...any) error {
	for _, entity := range entities {
		schema, err := parseAndRegister(db, entity)
		if err != nil {
			return fmt.Errorf("ego: AutoMigrate: %w", err)
		}

		ddl := generateCreateTable(db.dialect, schema)
		if _, err := db.sqlDB.Exec(ddl); err != nil {
			return fmt.Errorf("ego: AutoMigrate: failed to create table %q: %w", schema.TableName, err)
		}
	}
	return nil
}

// parseAndRegister parses the schema from an entity value (calling Configure
// if implemented) and registers it on the DB. If already registered, the
// existing schema is returned.
func parseAndRegister(db *DB, entity any) (*EntitySchema, error) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Return existing schema if already registered.
	if schema, exists := db.schemas[t]; exists {
		return schema, nil
	}

	// Build the schema using the non-generic variant that also calls Configure.
	schema, err := buildSchemaAny(entity)
	if err != nil {
		return nil, err
	}

	db.schemas[t] = schema
	return schema, nil
}

// generateCreateTable produces a CREATE TABLE IF NOT EXISTS DDL statement
// from the entity schema using the given dialect for quoting and type mapping.
func generateCreateTable(d Dialect, schema *EntitySchema) string {
	var b strings.Builder

	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(d.QuoteIdentifier(schema.TableName))
	b.WriteString(" (\n")

	for i, col := range schema.Columns {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("  ")
		b.WriteString(generateColumnDef(d, schema, &col))
	}

	b.WriteString("\n)")
	return b.String()
}

// generateColumnDef produces the DDL fragment for a single column, e.g.:
//
//	"id" INTEGER PRIMARY KEY AUTOINCREMENT
//	"name" TEXT NOT NULL
//	"email" TEXT NOT NULL UNIQUE
func generateColumnDef(d Dialect, schema *EntitySchema, col *ColumnSchema) string {
	var parts []string

	// Column name
	parts = append(parts, d.QuoteIdentifier(col.DBName))

	// Determine if this is the primary key column.
	isPK := schema.PrimaryKey != nil && schema.PrimaryKey.DBName == col.DBName

	// SQL type
	goTypeName := goTypeString(col.GoType)
	sqlType := d.TypeMapping(goTypeName)
	parts = append(parts, sqlType)

	// Primary key + auto-increment
	if isPK {
		parts = append(parts, "PRIMARY KEY")
		autoinc := d.AutoIncrementDef()
		if autoinc != "" {
			parts = append(parts, autoinc)
		}
	}

	// NOT NULL (skip for PK since it's implicitly NOT NULL)
	if !isPK && col.Required {
		parts = append(parts, "NOT NULL")
	}

	// UNIQUE
	if col.Unique {
		parts = append(parts, "UNIQUE")
	}

	// DEFAULT
	if col.DefaultValue != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %v", col.DefaultValue))
	}

	return strings.Join(parts, " ")
}

// goTypeString returns a string representation of a Go type suitable for
// dialect TypeMapping lookups. For example: "int64", "string", "time.Time".
func goTypeString(t reflect.Type) string {
	return t.String()
}
