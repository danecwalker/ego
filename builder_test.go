// builder_test.go
package ego_test

import (
	"testing"

	"github.com/danewilson/ego"
)

type UserEntity struct {
	ego.Model
	Name  string
	Email string
	Age   int
}

func (u *UserEntity) Configure(b *ego.EntityBuilder[UserEntity]) {
	b.ToTable("users")
	b.Property(&u.Name).HasMaxLength(255).IsRequired()
	b.Property(&u.Email).HasMaxLength(255).IsRequired().IsUnique()
	b.Property(&u.Age).HasDefault(0)
}

func TestEntityBuilderToTable(t *testing.T) {
	schema := ego.BuildSchema(&UserEntity{})
	if schema.TableName != "users" {
		t.Errorf("expected 'users', got %q", schema.TableName)
	}
}

func TestEntityBuilderPropertyRequired(t *testing.T) {
	schema := ego.BuildSchema(&UserEntity{})
	col := findColumn(schema, "name")
	if col == nil {
		t.Fatal("column 'name' not found")
	}
	if !col.Required {
		t.Error("expected name to be required")
	}
	if col.MaxLength != 255 {
		t.Errorf("expected max length 255, got %d", col.MaxLength)
	}
}

func TestEntityBuilderPropertyUnique(t *testing.T) {
	schema := ego.BuildSchema(&UserEntity{})
	col := findColumn(schema, "email")
	if col == nil {
		t.Fatal("column 'email' not found")
	}
	if !col.Unique {
		t.Error("expected email to be unique")
	}
}

func TestEntityBuilderPropertyDefault(t *testing.T) {
	schema := ego.BuildSchema(&UserEntity{})
	col := findColumn(schema, "age")
	if col == nil {
		t.Fatal("column 'age' not found")
	}
	if col.DefaultValue != 0 {
		t.Errorf("expected default 0, got %v", col.DefaultValue)
	}
}

func TestBuildSchemaFallsBackToParsedDefaults(t *testing.T) {
	// Entity without Configure method uses convention defaults
	schema := ego.BuildSchema(&SimpleEntity{})
	if schema.TableName != "simple_entities" {
		t.Errorf("expected 'simple_entities', got %q", schema.TableName)
	}
}

func findColumn(s *ego.EntitySchema, dbName string) *ego.ColumnSchema {
	for i := range s.Columns {
		if s.Columns[i].DBName == dbName {
			return &s.Columns[i]
		}
	}
	return nil
}
