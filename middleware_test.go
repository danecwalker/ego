package ego_test

import (
	"context"
	"testing"

	"github.com/danecwalker/ego"
)

func TestMiddlewareExecutesOnCreate(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	var called bool
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			called = true
			return next(op)
		}
	})

	ego.Create(db, ctx, &SimpleEntity{Name: "Alice"})

	if !called {
		t.Error("expected middleware to be called")
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	var order []int
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			order = append(order, 1)
			err := next(op)
			order = append(order, 3)
			return err
		}
	})
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			order = append(order, 2)
			return next(op)
		}
	})

	ego.Create(db, ctx, &SimpleEntity{Name: "Alice"})

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("unexpected middleware order: %v", order)
	}
}

func TestMiddlewareCanInspectOperation(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	var opType string
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			opType = op.Type
			return next(op)
		}
	})

	ego.Create(db, ctx, &SimpleEntity{Name: "Alice"})
	if opType != "create" {
		t.Errorf("expected 'create', got %q", opType)
	}
}

func TestMiddlewareExecutesOnUpdate(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Alice"}
	ego.Create(db, ctx, e)

	var opType string
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			opType = op.Type
			return next(op)
		}
	})

	e.Name = "Bob"
	ego.Update(db, ctx, e)

	if opType != "update" {
		t.Errorf("expected 'update', got %q", opType)
	}
}

func TestMiddlewareExecutesOnDelete(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	e := &SimpleEntity{Name: "Alice"}
	ego.Create(db, ctx, e)

	var opType string
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			opType = op.Type
			return next(op)
		}
	})

	ego.Delete(db, ctx, e)

	if opType != "delete" {
		t.Errorf("expected 'delete', got %q", opType)
	}
}

func TestMiddlewareReceivesSQLAndArgs(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	var capturedSQL string
	var capturedArgs []any
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			capturedSQL = op.SQL
			capturedArgs = op.Args
			return next(op)
		}
	})

	ego.Create(db, ctx, &SimpleEntity{Name: "Alice"})

	if capturedSQL == "" {
		t.Error("expected SQL to be populated in operation")
	}
	if len(capturedArgs) == 0 {
		t.Error("expected args to be populated in operation")
	}
}

func TestMiddlewareReceivesEntity(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	var capturedEntity any
	db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
		return func(op *ego.Operation) error {
			capturedEntity = op.Entity
			return next(op)
		}
	})

	original := &SimpleEntity{Name: "Alice"}
	ego.Create(db, ctx, original)

	if capturedEntity == nil {
		t.Error("expected entity to be populated in operation")
	}
	if se, ok := capturedEntity.(*SimpleEntity); !ok || se.Name != "Alice" {
		t.Errorf("expected entity with name Alice, got %v", capturedEntity)
	}
}
