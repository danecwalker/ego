// integration_test.go
package ego_test

import (
	"context"
	"errors"
	"testing"

	"github.com/danewilson/ego"
)

func TestFullCRUDWorkflow(t *testing.T) {
	db := setupTestDB(t, &UserEntity{})
	ctx := context.Background()

	// Create
	user := &UserEntity{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := ego.Create(db, ctx, user); err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 {
		t.Fatal("expected ID")
	}

	// Read
	fetched, err := ego.Query[UserEntity](db, ctx).
		Where(ego.Col("id").Eq(user.ID)).First()
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Email != "alice@example.com" {
		t.Errorf("expected alice@example.com, got %s", fetched.Email)
	}

	// Update
	fetched.Age = 31
	if err := ego.Update(db, ctx, fetched); err != nil {
		t.Fatal(err)
	}

	// Verify update
	updated, _ := ego.Query[UserEntity](db, ctx).
		Where(ego.Col("id").Eq(user.ID)).First()
	if updated.Age != 31 {
		t.Errorf("expected age 31, got %d", updated.Age)
	}

	// Delete
	if err := ego.Delete(db, ctx, updated); err != nil {
		t.Fatal(err)
	}

	// Verify delete
	_, err = ego.Query[UserEntity](db, ctx).
		Where(ego.Col("id").Eq(user.ID)).First()
	if err != ego.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTransactionWorkflow(t *testing.T) {
	db := setupTestDB(t, &UserEntity{})
	ctx := context.Background()

	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		ego.Create(tx, ctx, &UserEntity{Name: "A", Email: "a@test.com", Age: 1})
		ego.Create(tx, ctx, &UserEntity{Name: "B", Email: "b@test.com", Age: 2})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	count, _ := ego.Query[UserEntity](db, ctx).Count()
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestQueryWithMultipleConditions(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	// Multiple Where clauses (AND)
	users, err := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("age").Gt(25)).
		Where(ego.Col("age").Lt(35)).
		OrderBy(ego.Col("name").Asc()).
		All()
	if err != nil {
		t.Fatal(err)
	}
	// Alice(30), Anna(28) — Bob(25) excluded by >25, Charlie(35) excluded by <35
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if len(users) > 0 && users[0].Name != "Alice" {
		t.Errorf("expected Alice first (alphabetical), got %s", users[0].Name)
	}
}

func TestRelationshipWorkflow(t *testing.T) {
	db := setupTestDB(t, &Author{}, &Post{}, &Profile{})
	ctx := context.Background()

	// Create author with posts and profile
	author := &Author{Name: "Jane"}
	ego.Create(db, ctx, author)

	ego.Create(db, ctx, &Post{Title: "First Post", Body: "Hello", AuthorID: author.ID})
	ego.Create(db, ctx, &Post{Title: "Second Post", Body: "World", AuthorID: author.ID})
	ego.Create(db, ctx, &Profile{Bio: "Developer", AuthorID: author.ID})

	// Query with multiple includes
	result, err := ego.Query[Author](db, ctx).
		Where(ego.Col("id").Eq(author.ID)).
		Include("Posts").
		Include("Profile").
		First()
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(result.Posts))
	}
	if result.Profile == nil {
		t.Fatal("expected profile to be loaded")
	}
	if result.Profile.Bio != "Developer" {
		t.Errorf("expected 'Developer', got %q", result.Profile.Bio)
	}
}

func TestHooksAndTransactionIntegration(t *testing.T) {
	db := setupTestDB(t, &HookedEntity{})
	ctx := context.Background()

	// Hooks should work inside transactions
	err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		e := &HookedEntity{Email: "UPPER@TEST.COM"}
		return ego.Create(tx, ctx, e)
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify the BeforeCreate hook lowercased the email
	result, _ := ego.Query[HookedEntity](db, ctx).First()
	if result.Email != "upper@test.com" {
		t.Errorf("expected lowercase email, got %q", result.Email)
	}
}

func TestTransactionRollbackLeavesNoTrace(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	// Create one entity outside transaction
	ego.Create(db, ctx, &SimpleEntity{Name: "Permanent"})

	// Failed transaction
	ego.Transaction(db, ctx, func(tx *ego.Tx) error {
		ego.Create(tx, ctx, &SimpleEntity{Name: "Temporary"})
		return errors.New("abort")
	})

	// Only the permanent entity should exist
	count, _ := ego.Query[SimpleEntity](db, ctx).Count()
	if count != 1 {
		t.Errorf("expected 1 entity, got %d", count)
	}

	result, _ := ego.Query[SimpleEntity](db, ctx).First()
	if result.Name != "Permanent" {
		t.Errorf("expected 'Permanent', got %q", result.Name)
	}
}
