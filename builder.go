package ego

import (
	"reflect"
	"unsafe"
)

// EntityBuilder provides a fluent API for configuring an entity's schema.
// It is parameterized by the entity type T so that the Configure method
// receives a correctly typed builder.
type EntityBuilder[T any] struct {
	schema *EntitySchema
	zero   *T
}

// PropertyBuilder provides a fluent API for configuring a single column.
type PropertyBuilder struct {
	col *ColumnSchema
}

// ToTable overrides the default table name derived from the struct name.
func (b *EntityBuilder[T]) ToTable(name string) {
	b.schema.TableName = name
}

// Property returns a PropertyBuilder for the column that corresponds to the
// given field pointer. The fieldPtr must be a pointer to a field on the same
// entity instance that was passed to Configure (i.e., the receiver).
//
// It works by comparing the address of fieldPtr with the addresses of fields
// on the builder's zero instance.
func (b *EntityBuilder[T]) Property(fieldPtr any) *PropertyBuilder {
	ptrVal := reflect.ValueOf(fieldPtr).Pointer()
	zeroVal := reflect.ValueOf(b.zero).Elem()

	for i := range b.schema.Columns {
		col := &b.schema.Columns[i]
		fieldAddr := zeroVal.FieldByIndex(col.Index).Addr().Pointer()
		if fieldAddr == ptrVal {
			return &PropertyBuilder{col: col}
		}
	}

	// If no column matched, return a no-op PropertyBuilder to avoid panics.
	// In practice, this means the caller passed a pointer to a field that
	// isn't part of the schema (e.g., a relationship field).
	return &PropertyBuilder{col: &ColumnSchema{}}
}

// HasMaxLength sets the maximum length for a string column.
func (pb *PropertyBuilder) HasMaxLength(n int) *PropertyBuilder {
	pb.col.MaxLength = n
	return pb
}

// IsRequired marks the column as NOT NULL.
func (pb *PropertyBuilder) IsRequired() *PropertyBuilder {
	pb.col.Required = true
	return pb
}

// IsUnique adds a UNIQUE constraint to the column.
func (pb *PropertyBuilder) IsUnique() *PropertyBuilder {
	pb.col.Unique = true
	return pb
}

// HasDefault sets a default value for the column.
func (pb *PropertyBuilder) HasDefault(v any) *PropertyBuilder {
	pb.col.DefaultValue = v
	return pb
}

// BuildSchema produces a fully configured EntitySchema for the given entity.
// It first calls ParseSchema to derive convention-based defaults, then checks
// if the entity implements a Configure method and calls it to apply fluent
// customizations.
func BuildSchema[T any](entity *T) *EntitySchema {
	schema, err := ParseSchema(entity)
	if err != nil {
		panic("ego: BuildSchema: " + err.Error())
	}

	// Create the builder with the provided entity as the zero instance.
	// The Configure method's receiver will be this same pointer, so field
	// pointer addresses will match.
	builder := &EntityBuilder[T]{
		schema: schema,
		zero:   entity,
	}

	// Check if *T has a Configure(*EntityBuilder[T]) method using reflect.
	// We cannot use a type assertion for generic interface checking, so we
	// look for the method by name and call it dynamically.
	entityVal := reflect.ValueOf(entity)
	configureMethod := entityVal.MethodByName("Configure")
	if configureMethod.IsValid() {
		configureMethod.Call([]reflect.Value{reflect.ValueOf(builder)})
	}

	return schema
}

// buildSchemaAny is a non-generic variant of BuildSchema that accepts any.
// It is used by AutoMigrate which receives entities as []any.
// It uses unsafe to set unexported fields on the reflectively-created EntityBuilder.
func buildSchemaAny(entity any) (*EntitySchema, error) {
	schema, err := ParseSchema(entity)
	if err != nil {
		return nil, err
	}

	entityVal := reflect.ValueOf(entity)
	configureMethod := entityVal.MethodByName("Configure")
	if !configureMethod.IsValid() {
		return schema, nil
	}

	// The Configure method expects an *EntityBuilder[T].
	// Determine the concrete builder type from the method signature.
	methodType := configureMethod.Type()
	if methodType.NumIn() != 1 {
		return schema, nil
	}
	builderPtrType := methodType.In(0) // *EntityBuilder[T]
	if builderPtrType.Kind() != reflect.Ptr {
		return schema, nil
	}
	builderType := builderPtrType.Elem() // EntityBuilder[T]

	// Allocate a new EntityBuilder[T] via reflection.
	builderPtr := reflect.New(builderType) // *EntityBuilder[T]
	builderElem := builderPtr.Elem()       // EntityBuilder[T]

	// Set the unexported 'schema' field using unsafe.
	schemaField := builderElem.Field(0) // first field: schema *EntitySchema
	schemaFieldPtr := unsafe.Pointer(schemaField.UnsafeAddr())
	*(*unsafe.Pointer)(schemaFieldPtr) = unsafe.Pointer(schema)

	// Set the unexported 'zero' field using unsafe.
	// 'zero' is the second field and is a *T, which is the same type as entity.
	zeroField := builderElem.Field(1) // second field: zero *T
	zeroFieldPtr := unsafe.Pointer(zeroField.UnsafeAddr())
	*(*unsafe.Pointer)(zeroFieldPtr) = unsafe.Pointer(entityVal.Pointer())

	configureMethod.Call([]reflect.Value{builderPtr})

	return schema, nil
}
