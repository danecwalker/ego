package ego

import (
	"reflect"
	"strings"
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

// RelationshipBuilder provides a fluent API for configuring a relationship.
type RelationshipBuilder struct {
	rel *RelationshipSchema
}

// WithForeignKey overrides the conventional foreign key column name.
func (rb *RelationshipBuilder) WithForeignKey(fk string) *RelationshipBuilder {
	rb.rel.ForeignKey = fk
	return rb
}

// HasMany registers a one-to-many relationship. The fieldPtr must be a pointer
// to a []RelatedType slice field on the entity (e.g., &a.Posts).
// The foreign key is inferred by convention: lowercased parent type name + "_id"
// (e.g., Author -> "author_id").
func (b *EntityBuilder[T]) HasMany(fieldPtr any) *RelationshipBuilder {
	ptrVal := reflect.ValueOf(fieldPtr).Pointer()
	zeroVal := reflect.ValueOf(b.zero).Elem()
	t := zeroVal.Type()

	rel := &RelationshipSchema{Type: HasManyRel}

	// Walk struct fields looking for the matching slice field by address.
	findRelField(t, nil, zeroVal, ptrVal, rel)

	if rel.FieldName != "" {
		// Infer foreign key: lowercased parent type name + "_id"
		parentName := t.Name()
		rel.ForeignKey = strings.ToLower(parentName) + "_id"
		b.schema.Relationships = append(b.schema.Relationships, *rel)
	}

	return &RelationshipBuilder{rel: &b.schema.Relationships[len(b.schema.Relationships)-1]}
}

// BelongsTo registers an inverse one-to-many (belongs-to) relationship.
// The fieldPtr must be a pointer to a *RelatedType pointer field on the entity
// (e.g., &p.Author). The foreign key column is on the current entity and is
// inferred by convention: lowercased related type name + "_id"
// (e.g., *Author -> "author_id").
func (b *EntityBuilder[T]) BelongsTo(fieldPtr any) *RelationshipBuilder {
	ptrVal := reflect.ValueOf(fieldPtr).Pointer()
	zeroVal := reflect.ValueOf(b.zero).Elem()
	t := zeroVal.Type()

	rel := &RelationshipSchema{Type: BelongsToRel}

	// Walk struct fields looking for the matching pointer-to-struct field by address.
	findRelField(t, nil, zeroVal, ptrVal, rel)

	if rel.FieldName != "" {
		// Infer foreign key: lowercased related type name + "_id"
		relatedName := rel.RelatedType.Name()
		rel.ForeignKey = strings.ToLower(relatedName) + "_id"
		b.schema.Relationships = append(b.schema.Relationships, *rel)
	}

	return &RelationshipBuilder{rel: &b.schema.Relationships[len(b.schema.Relationships)-1]}
}

// HasOne registers a one-to-one relationship where the foreign key is on the
// related entity's table. The fieldPtr must be a pointer to a *RelatedType
// pointer field on the entity (e.g., &a.Profile).
// For Author.HasOne(&a.Profile), the FK is "author_id" on the profiles table.
func (b *EntityBuilder[T]) HasOne(fieldPtr any) *RelationshipBuilder {
	ptrVal := reflect.ValueOf(fieldPtr).Pointer()
	zeroVal := reflect.ValueOf(b.zero).Elem()
	t := zeroVal.Type()

	rel := &RelationshipSchema{Type: HasOneRel}

	// Walk struct fields looking for the matching pointer-to-struct field by address.
	findRelField(t, nil, zeroVal, ptrVal, rel)

	if rel.FieldName != "" {
		// Infer foreign key: lowercased parent type name + "_id"
		parentName := t.Name()
		rel.ForeignKey = strings.ToLower(parentName) + "_id"
		b.schema.Relationships = append(b.schema.Relationships, *rel)
	}

	return &RelationshipBuilder{rel: &b.schema.Relationships[len(b.schema.Relationships)-1]}
}

// ManyToMany registers a many-to-many relationship via a pivot table.
// The fieldPtr must be a pointer to a []RelatedType slice field on the entity
// (e.g., &a.Tags).
// For Article.ManyToMany(&a.Tags):
//   - Pivot table: "article_tags" (singular owner + "_" + plural related)
//   - PivotFKSelf: "article_id"
//   - PivotFKOther: "tag_id"
func (b *EntityBuilder[T]) ManyToMany(fieldPtr any) *RelationshipBuilder {
	ptrVal := reflect.ValueOf(fieldPtr).Pointer()
	zeroVal := reflect.ValueOf(b.zero).Elem()
	t := zeroVal.Type()

	rel := &RelationshipSchema{Type: ManyToManyRel}

	// Walk struct fields looking for the matching slice field by address.
	findRelField(t, nil, zeroVal, ptrVal, rel)

	if rel.FieldName != "" {
		ownerName := strings.ToLower(t.Name())
		relatedName := strings.ToLower(rel.RelatedType.Name())

		// Pivot table: singular owner + "_" + plural related
		rel.PivotTable = ownerName + "_" + pluralize(relatedName)
		rel.PivotFKSelf = ownerName + "_id"
		rel.PivotFKOther = relatedName + "_id"
		b.schema.Relationships = append(b.schema.Relationships, *rel)
	}

	return &RelationshipBuilder{rel: &b.schema.Relationships[len(b.schema.Relationships)-1]}
}

// findRelField walks a struct type recursively (flattening embedded structs)
// to find a relationship field (slice or pointer-to-struct) matching ptrVal.
// When found, it populates the RelationshipSchema with field metadata.
// rootVal must be the reflect.Value of the top-level struct so that
// FieldByIndex works correctly with the full index path.
func findRelField(t reflect.Type, indexPrefix []int, rootVal reflect.Value, ptrVal uintptr, rel *RelationshipSchema) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldType := field.Type
		index := append(append([]int{}, indexPrefix...), field.Index...)

		// Flatten embedded structs.
		if field.Anonymous && fieldType.Kind() == reflect.Struct {
			findRelField(fieldType, index, rootVal, ptrVal, rel)
			continue
		}

		// Check slice fields (HasMany candidates).
		if fieldType.Kind() == reflect.Slice {
			fieldAddr := rootVal.FieldByIndex(index).Addr().Pointer()
			if fieldAddr == ptrVal {
				rel.FieldName = field.Name
				rel.FieldIndex = index
				rel.RelatedType = fieldType.Elem() // element type of the slice
				return
			}
		}

		// Check pointer-to-struct fields (BelongsTo candidates).
		if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
			fieldAddr := rootVal.FieldByIndex(index).Addr().Pointer()
			if fieldAddr == ptrVal {
				rel.FieldName = field.Name
				rel.FieldIndex = index
				rel.RelatedType = fieldType.Elem() // the struct type pointed to
				return
			}
		}
	}
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
