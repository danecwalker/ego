package ego

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// EntitySchema holds the parsed metadata for an entity struct.
type EntitySchema struct {
	TableName     string
	GoType        reflect.Type
	Columns       []ColumnSchema
	PrimaryKey    *ColumnSchema
	Relationships []RelationshipSchema
}

// ColumnSchema describes a single database column derived from a struct field.
type ColumnSchema struct {
	FieldName string       // Go field name
	DBName    string       // snake_case column name
	GoType    reflect.Type // Go type of the field
	Index     []int        // reflect field index path for nested access

	// Configuration (set by EntityBuilder later)
	MaxLength    int
	Required     bool
	Unique       bool
	DefaultValue any
}

// ParseSchema inspects a struct (or pointer to struct) and produces an
// EntitySchema describing its table name, columns, and primary key.
func ParseSchema(entity any) (*EntitySchema, error) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("ego: ParseSchema expects a struct or pointer to struct, got %s", t.Kind())
	}

	schema := &EntitySchema{
		TableName: deriveTableName(t.Name()),
		GoType:    t,
	}

	// Collect columns by flattening embedded structs.
	collectFields(t, nil, schema)

	// Auto-detect primary key: look for an "id" column that came from Model embedding.
	for i := range schema.Columns {
		col := &schema.Columns[i]
		if col.DBName == "id" && col.GoType.Kind() == reflect.Int64 {
			schema.PrimaryKey = col
			break
		}
	}

	return schema, nil
}

// collectFields recursively inspects struct fields and appends column
// definitions to the schema. Embedded structs are flattened; slices and
// pointer-to-struct fields are skipped (they represent relationships).
func collectFields(t reflect.Type, indexPrefix []int, schema *EntitySchema) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields.
		if !field.IsExported() {
			continue
		}

		fieldType := field.Type
		index := append(append([]int{}, indexPrefix...), field.Index...)

		// Handle embedded structs: flatten their fields into the parent.
		if field.Anonymous && fieldType.Kind() == reflect.Struct {
			collectFields(fieldType, index, schema)
			continue
		}

		// Skip slice fields (e.g., []ChildEntity — relationships).
		if fieldType.Kind() == reflect.Slice {
			continue
		}

		// Skip pointer-to-struct fields (e.g., *Profile — relationships).
		if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
			continue
		}

		schema.Columns = append(schema.Columns, ColumnSchema{
			FieldName: field.Name,
			DBName:    toSnakeCase(field.Name),
			GoType:    fieldType,
			Index:     index,
		})
	}
}

// deriveTableName converts a Go struct name to a pluralized snake_case table name.
// e.g., "SimpleEntity" -> "simple_entities", "User" -> "users".
func deriveTableName(name string) string {
	return pluralize(toSnakeCase(name))
}

// toSnakeCase converts a CamelCase or PascalCase string to snake_case.
// It handles consecutive uppercase letters (acronyms) correctly:
//
//	"HTMLBody"  -> "html_body"
//	"FirstName" -> "first_name"
//	"ID"        -> "id"
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				// Insert underscore before an uppercase letter when:
				// 1. The previous character is lowercase, OR
				// 2. The previous character is uppercase AND the next character is lowercase
				//    (handles transitions like "HTML" + "Body" -> "html_body").
				prevLower := unicode.IsLower(runes[i-1])
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if prevLower || (unicode.IsUpper(runes[i-1]) && nextLower) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// pluralize applies simple English pluralization rules.
// It handles common suffixes; it is not a full inflector.
func pluralize(s string) string {
	if s == "" {
		return s
	}

	// Words ending in s, x, z, ch, sh -> add "es"
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}

	// Words ending in consonant + y -> replace y with "ies"
	if strings.HasSuffix(s, "y") {
		// Check if the character before 'y' is a consonant.
		if len(s) >= 2 {
			prev := rune(s[len(s)-2])
			if !isVowel(prev) {
				return s[:len(s)-1] + "ies"
			}
		}
		return s + "s"
	}

	return s + "s"
}

// isVowel returns true if the rune is an English vowel (lowercase).
func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}
