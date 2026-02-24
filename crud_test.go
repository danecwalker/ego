// crud_test.go
package ego_test

import (
	"context"
	"testing"
	"time"

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

// === Update Tests ===

func TestUpdateModifiesEntity(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Alice", Email: "alice@test.com", Age: 30}
	ego.Create(db, ctx, e)

	e.Name = "Alice Updated"
	e.Age = 31
	err := ego.Update(db, ctx, e)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	fetched, _ := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("id").Eq(e.ID)).First()
	if fetched.Name != "Alice Updated" {
		t.Errorf("expected 'Alice Updated', got %q", fetched.Name)
	}
	if fetched.Age != 31 {
		t.Errorf("expected age 31, got %d", fetched.Age)
	}
}

func TestUpdateSetsUpdatedAt(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Alice"}
	ego.Create(db, ctx, e)
	originalUpdatedAt := e.UpdatedAt

	// Ensure some time passes
	time.Sleep(time.Millisecond)

	e.Name = "Alice Updated"
	ego.Update(db, ctx, e)

	if !e.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to advance")
	}
}

func TestUpdateWithZeroIDReturnsError(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "NoID"}
	err := ego.Update(db, ctx, e)
	if err == nil {
		t.Error("expected error updating entity with zero ID")
	}
}

// === Delete Tests ===

func TestDeleteRemovesEntity(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Alice"}
	ego.Create(db, ctx, e)

	err := ego.Delete(db, ctx, e)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("id").Eq(e.ID)).First()
	if err != ego.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteWithZeroIDReturnsError(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "NoID"}
	err := ego.Delete(db, ctx, e)
	if err == nil {
		t.Error("expected error deleting entity with zero ID")
	}
}
