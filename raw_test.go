// raw_test.go
package ego_test

import (
	"context"
	"testing"

	"github.com/danewilson/ego"
)

func TestRawQueryScanResults(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	var results []SimpleEntity
	err := ego.Raw[SimpleEntity](db, ctx, "SELECT * FROM simple_entities WHERE age > ?", 25).
		Scan(&results)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3, got %d", len(results))
	}
}

func TestRawExec(t *testing.T) {
	db := setupTestDB(t, &SimpleEntity{})
	seedUsers(t, db)
	ctx := context.Background()

	result, err := ego.RawExec(db, ctx, "DELETE FROM simple_entities WHERE age < ?", 28)
	if err != nil {
		t.Fatal(err)
	}
	affected, _ := result.RowsAffected()
	if affected != 1 { // Bob age 25
		t.Errorf("expected 1 affected, got %d", affected)
	}
}
