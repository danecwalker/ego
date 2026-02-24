package ego

import "reflect"

// RelType describes the kind of relationship.
type RelType int

const (
	HasManyRel   RelType = iota
	BelongsToRel
	HasOneRel
	ManyToManyRel
)

// RelationshipSchema describes a relationship between two entities.
type RelationshipSchema struct {
	Type        RelType
	FieldName   string       // Go field name (e.g., "Posts", "Author")
	FieldIndex  []int        // reflect index path to the relationship field
	RelatedType reflect.Type // the related entity's type
	ForeignKey  string       // DB column name of the FK (e.g., "author_id")
	// For ManyToMany (Task 17):
	PivotTable   string
	PivotFKSelf  string
	PivotFKOther string
}
