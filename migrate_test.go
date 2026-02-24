// migrate_test.go
package ego_test

import (
	"testing"

	"github.com/danecwalker/ego"
	"github.com/danecwalker/ego/sqlite"
)

func TestAutoMigrateCreatesTable(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	err := ego.AutoMigrate(db, &SimpleEntity{})
	if err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	exists := db.TableExists("simple_entities")
	if !exists {
		t.Error("expected table 'simple_entities' to exist")
	}
}

func TestAutoMigrateCreatesCorrectColumns(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	ego.AutoMigrate(db, &SimpleEntity{})

	cols := db.ColumnNames("simple_entities")
	expected := []string{"id", "created_at", "updated_at", "name", "email", "age"}
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d: %v", len(expected), len(cols), cols)
	}
}

func TestAutoMigrateWithConfiguredEntity(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	ego.AutoMigrate(db, &UserEntity{})

	exists := db.TableExists("users")
	if !exists {
		t.Error("expected table 'users' (configured name) to exist")
	}
}

func TestAutoMigrateIdempotent(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	ego.AutoMigrate(db, &SimpleEntity{})
	err := ego.AutoMigrate(db, &SimpleEntity{})
	if err != nil {
		t.Errorf("second AutoMigrate should not error: %v", err)
	}
}
