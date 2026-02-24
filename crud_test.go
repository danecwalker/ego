// crud_test.go
package ego_test

import (
	"context"
	"testing"

	"github.com/danewilson/ego"
	"github.com/danewilson/ego/sqlite"
)

func setupTestDB(t *testing.T, entities ...any) *ego.DB {
	t.Helper()
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := ego.AutoMigrate(db, entities...); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateInsertsSingleEntity(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Alice", Email: "alice@example.com", Age: 30}
	err := ego.Create(db, ctx, e)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if e.ID == 0 {
		t.Error("expected ID to be populated after insert")
	}
}

func TestCreateSetsTimestamps(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Bob"}
	ego.Create(db, ctx, e)

	if e.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if e.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestCreateReturnsErrorForNil(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	err := ego.Create[SimpleEntity](db, ctx, nil)
	if err == nil {
		t.Error("expected error for nil entity")
	}
}
