// register.go
package ego

import "reflect"

// Register registers an entity type with the database, parsing its schema
// and applying any Configure customizations. Duplicate registrations are no-ops.
func Register[T any](db *DB) error {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if _, exists := db.schemas[t]; exists {
		return nil
	}
	entity := new(T)
	schema := BuildSchema(entity)
	db.schemas[t] = schema
	return nil
}

// SchemaFor returns the registered schema for entity type T, or nil if not registered.
func SchemaFor[T any](db *DB) *EntitySchema {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return db.schemas[t]
}
