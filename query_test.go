// query_test.go
package ego_test

import (
	"context"
	"testing"

	"github.com/danewilson/ego"
)

func seedUsers(t *testing.T, db *ego.DB) {
	t.Helper()
	ctx := context.Background()
	for _, u := range []SimpleEntity{
		{Name: "Alice", Email: "alice@test.com", Age: 30},
		{Name: "Bob", Email: "bob@test.com", Age: 25},
		{Name: "Charlie", Email: "charlie@test.com", Age: 35},
		{Name: "Anna", Email: "anna@test.com", Age: 28},
	} {
		if err := ego.Create(db, ctx, &u); err != nil {
			t.Fatal(err)
		}
	}
}

func TestQueryAll(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	users, err := ego.Query[SimpleEntity](db, ctx).All()
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("expected 4 users, got %d", len(users))
	}
}

func TestQueryFirst(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	user, err := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("name").Eq("Alice")).
		First()
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user.Name)
	}
}

func TestQueryWhereGt(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	users, err := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("age").Gt(28)).
		All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 { // Alice(30), Charlie(35)
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestQueryWhereLike(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	users, err := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("name").Like("A%")).
		All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 { // Alice, Anna
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestQueryOrderBy(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	users, err := ego.Query[SimpleEntity](db, ctx).
		OrderBy(ego.Col("age").Asc()).
		All()
	if err != nil {
		t.Fatal(err)
	}
	if users[0].Name != "Bob" {
		t.Errorf("expected Bob first (age 25), got %s", users[0].Name)
	}
}

func TestQueryLimitOffset(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	users, err := ego.Query[SimpleEntity](db, ctx).
		OrderBy(ego.Col("age").Asc()).
		Limit(2).
		Offset(1).
		All()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestQueryCount(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	count, err := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("age").Gt(25)).
		Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestQueryFirstNoResultReturnsError(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	ctx := context.Background()

	_, err := ego.Query[SimpleEntity](db, ctx).
		Where(ego.Col("name").Eq("Nobody")).
		First()
	if err != ego.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
