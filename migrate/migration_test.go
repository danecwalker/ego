package migrate_test

import (
	"testing"

	"github.com/danecwalker/ego"
	"github.com/danecwalker/ego/migrate"
	"github.com/danecwalker/ego/sqlite"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) Version() string { return "20260224_001" }

func (m *CreateUsersTable) Up(s *migrate.Schema) {
	s.CreateTable("users", func(t *migrate.Table) {
		t.BigInt("id").PrimaryKey().AutoIncrement()
		t.String("name", 255).NotNull()
		t.String("email", 255).NotNull().Unique()
		t.Int("age").Default("0")
		t.Timestamps()
	})
}

func (m *CreateUsersTable) Down(s *migrate.Schema) {
	s.DropTable("users")
}

func TestMigrateUp(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	err := migrate.Run(db, migrate.Up, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}
	if !db.TableExists("users") {
		t.Error("expected users table to exist")
	}
}

func TestMigrateDown(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	migrate.Run(db, migrate.Up, &CreateUsersTable{})
	err := migrate.Run(db, migrate.Down, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}
	if db.TableExists("users") {
		t.Error("expected users table to be dropped")
	}
}

func TestMigrateTracksVersion(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	migrate.Run(db, migrate.Up, &CreateUsersTable{})

	// Running same migration again should be a no-op
	err := migrate.Run(db, migrate.Up, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMigrateMultiple(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	err := migrate.Run(db, migrate.Up, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}
	if !db.TableExists("users") {
		t.Error("expected users table")
	}
}

func TestMigrateUpCreatesCorrectColumns(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	err := migrate.Run(db, migrate.Up, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}

	cols := db.ColumnNames("users")
	expected := []string{"id", "name", "email", "age", "created_at", "updated_at"}
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d: %v", len(expected), len(cols), cols)
	}
	for i, name := range expected {
		if cols[i] != name {
			t.Errorf("column %d: expected %q, got %q", i, name, cols[i])
		}
	}
}

func TestMigrateDownSkipsUnapplied(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	// Running Down on a migration that was never applied should be a no-op
	err := migrate.Run(db, migrate.Down, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}
	if db.TableExists("users") {
		t.Error("expected users table to not exist")
	}
}

func TestMigrateCreatesTrackingTable(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	err := migrate.Run(db, migrate.Up, &CreateUsersTable{})
	if err != nil {
		t.Fatal(err)
	}
	if !db.TableExists("schema_migrations") {
		t.Error("expected schema_migrations tracking table to exist")
	}
}

func TestMigrateMultipleMigrations(t *testing.T) {
	db, _ := ego.Open(sqlite.New(":memory:"))
	defer db.Close()

	type CreatePostsTable struct{}

	posts := &createPostsMigration{}

	err := migrate.Run(db, migrate.Up, &CreateUsersTable{}, posts)
	if err != nil {
		t.Fatal(err)
	}
	if !db.TableExists("users") {
		t.Error("expected users table")
	}
	if !db.TableExists("posts") {
		t.Error("expected posts table")
	}

	// Down should reverse in order
	err = migrate.Run(db, migrate.Down, &CreateUsersTable{}, posts)
	if err != nil {
		t.Fatal(err)
	}
	if db.TableExists("users") {
		t.Error("expected users table to be dropped")
	}
	if db.TableExists("posts") {
		t.Error("expected posts table to be dropped")
	}
}

// createPostsMigration is a helper migration for testing multiple migrations.
type createPostsMigration struct{}

func (m *createPostsMigration) Version() string { return "20260224_002" }

func (m *createPostsMigration) Up(s *migrate.Schema) {
	s.CreateTable("posts", func(t *migrate.Table) {
		t.BigInt("id").PrimaryKey().AutoIncrement()
		t.String("title", 255).NotNull()
		t.Text("body")
		t.Bool("published").Default("0")
		t.Timestamps()
	})
}

func (m *createPostsMigration) Down(s *migrate.Schema) {
	s.DropTable("posts")
}
