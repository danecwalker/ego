// register_test.go
package ego_test

import (
	"testing"

	"github.com/danewilson/ego"
	"github.com/danewilson/ego/sqlite"
)

func TestRegisterEntity(t *testing.T) {
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = ego.Register[UserEntity](db)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	schema := ego.SchemaFor[UserEntity](db)
	if schema == nil {
		t.Fatal("expected schema to be registered")
	}
	if schema.TableName != "users" {
		t.Errorf("expected 'users', got %q", schema.TableName)
	}
}

func TestRegisterDuplicateIsNoop(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	ego.Register[UserEntity](db)
	err := ego.Register[UserEntity](db)
	if err != nil {
		t.Errorf("duplicate register should not error: %v", err)
	}
}
