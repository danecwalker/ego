// tx_test.go
package ego_test

import (
	"context"
	"errors"
	"testing"

	"github.com/danewilson/ego"
)

func TestTransactionCommitsOnSuccess(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		return ego.Create(tx, ctx, &SimpleEntity{Name: "Alice", Email: "a@test.com"})
	})
	if err != nil {
		t.Fatal(err)
	}

	users, err := ego.Query[SimpleEntity](db, ctx).All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestTransactionRollsBackOnError(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	testErr := errors.New("rollback me")
	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		ego.Create(tx, ctx, &SimpleEntity{Name: "Alice", Email: "a@test.com"})
		return testErr
	})
	if !errors.Is(err, testErr) {
		t.Errorf("expected testErr, got %v", err)
	}

	users, err := ego.Query[SimpleEntity](db, ctx).All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users after rollback, got %d", len(users))
	}
}

func TestTransactionRollsBackOnPanic(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	func() {
		defer func() { recover() }()
		ego.Transaction(db, ctx, func(tx *ego.Tx) error {
			ego.Create(tx, ctx, &SimpleEntity{Name: "Alice", Email: "a@test.com"})
			panic("oops")
		})
	}()

	users, err := ego.Query[SimpleEntity](db, ctx).All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users after panic rollback, got %d", len(users))
	}
}

func TestTransactionMultipleOperations(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		if err := ego.Create(tx, ctx, &SimpleEntity{Name: "Alice", Email: "a@test.com"}); err != nil {
			return err
		}
		if err := ego.Create(tx, ctx, &SimpleEntity{Name: "Bob", Email: "b@test.com"}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	users, err := ego.Query[SimpleEntity](db, ctx).All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestTransactionQueryInsideTx(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		if err := ego.Create(tx, ctx, &SimpleEntity{Name: "Alice", Email: "a@test.com"}); err != nil {
			return err
		}

		// Query within the same transaction should see the uncommitted row.
		users, err := ego.Query[SimpleEntity](tx, ctx).All()
		if err != nil {
			return err
		}
		if len(users) != 1 {
			t.Errorf("expected 1 user inside tx, got %d", len(users))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransactionUpdateAndDelete(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	// Create outside transaction
	e := &SimpleEntity{Name: "Alice", Email: "a@test.com", Age: 30}
	if err := ego.Create(db, ctx, e); err != nil {
		t.Fatal(err)
	}

	// Update and delete inside transaction
	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		e.Name = "Alice Updated"
		if err := ego.Update(tx, ctx, e); err != nil {
			return err
		}

		return ego.Delete(tx, ctx, e)
	})
	if err != nil {
		t.Fatal(err)
	}

	users, err := ego.Query[SimpleEntity](db, ctx).All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users after delete in tx, got %d", len(users))
	}
}
