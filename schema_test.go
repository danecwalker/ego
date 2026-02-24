// schema_test.go
package ego_test

import (
	"testing"

	"github.com/danecwalker/ego"
)

type SimpleEntity struct {
	ego.Model
	Name  string
	Email string
	Age   int
}

func TestParseSchemaFromStruct(t *testing.T) {
	schema, err := ego.ParseSchema(&SimpleEntity{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.TableName != "simple_entities" {
		t.Errorf("expected table 'simple_entities', got %q", schema.TableName)
	}
	// Model fields + own fields
	expectedCols := []string{"id", "created_at", "updated_at", "name", "email", "age"}
	if len(schema.Columns) != len(expectedCols) {
		t.Fatalf("expected %d columns, got %d", len(expectedCols), len(schema.Columns))
	}
	for i, col := range schema.Columns {
		if col.DBName != expectedCols[i] {
			t.Errorf("column %d: expected %q, got %q", i, expectedCols[i], col.DBName)
		}
	}
}

func TestParseSchemaFindsIDPrimaryKey(t *testing.T) {
	schema, err := ego.ParseSchema(&SimpleEntity{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.PrimaryKey == nil {
		t.Fatal("expected primary key to be set")
	}
	if schema.PrimaryKey.DBName != "id" {
		t.Errorf("expected PK 'id', got %q", schema.PrimaryKey.DBName)
	}
}

func TestParseSchemaSnakeCaseConversion(t *testing.T) {
	type MyComplexName struct {
		ego.Model
		FirstName string
		LastName  string
		HTMLBody  string
	}
	schema, err := ego.ParseSchema(&MyComplexName{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.TableName != "my_complex_names" {
		t.Errorf("expected 'my_complex_names', got %q", schema.TableName)
	}
	names := make(map[string]bool)
	for _, c := range schema.Columns {
		names[c.DBName] = true
	}
	for _, expected := range []string{"first_name", "last_name", "html_body"} {
		if !names[expected] {
			t.Errorf("expected column %q not found", expected)
		}
	}
}

type NoModelEntity struct {
	MyID   int64
	Detail string
}

func TestParseSchemaWithoutModelEmbedding(t *testing.T) {
	schema, err := ego.ParseSchema(&NoModelEntity{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.TableName != "no_model_entities" {
		t.Errorf("expected 'no_model_entities', got %q", schema.TableName)
	}
	// Without Model, no auto primary key — PK is nil until configured
	if schema.PrimaryKey != nil {
		t.Errorf("expected no primary key without Model embedding")
	}
}

func TestParseSchemaSkipsSliceAndPointerFields(t *testing.T) {
	type Parent struct {
		ego.Model
		Name     string
		Children []SimpleEntity
		Profile  *SimpleEntity
	}
	schema, err := ego.ParseSchema(&Parent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, col := range schema.Columns {
		if col.DBName == "children" || col.DBName == "profile" {
			t.Errorf("relationship field %q should not be a column", col.DBName)
		}
	}
}
