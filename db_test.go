package ego_test

import (
	"testing"

	"github.com/danewilson/ego"
	"github.com/danewilson/ego/sqlite"
)

func TestOpenWithSQLite(t *testing.T) {
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer db.Close()
	if db == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestOpenWithOptions(t *testing.T) {
	db, err := ego.Open(sqlite.New(":memory:"),
		ego.WithMaxOpenConns(5),
		ego.WithMaxIdleConns(2),
	)
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer db.Close()
}

func TestOpenPingsDatabase(t *testing.T) {
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		t.Fatalf("expected successful ping: %v", err)
	}
	db.Close()
}
