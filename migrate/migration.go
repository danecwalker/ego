package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/danewilson/ego"
)

// Direction indicates whether to apply or rollback migrations.
type Direction int

const (
	// Up applies migrations (creates tables, adds columns, etc.).
	Up Direction = iota
	// Down rolls back migrations (drops tables, removes columns, etc.).
	Down
)

// Migration defines an explicit migration with Up and Down methods.
type Migration interface {
	// Version returns a unique version string that identifies this migration.
	Version() string
	// Up defines the forward migration (e.g., create tables).
	Up(s *Schema)
	// Down defines the reverse migration (e.g., drop tables).
	Down(s *Schema)
}

// Run executes one or more migrations in the given direction.
// It creates a schema_migrations tracking table and skips already-applied migrations.
func Run(db *ego.DB, dir Direction, migrations ...Migration) error {
	ctx := context.Background()
	sqlDB := db.SqlDB()

	// 1. Create schema_migrations tracking table if not exists.
	createTrackingSQL := "CREATE TABLE IF NOT EXISTS " +
		db.Dialect().QuoteIdentifier("schema_migrations") +
		" (" +
		db.Dialect().QuoteIdentifier("version") + " TEXT PRIMARY KEY, " +
		db.Dialect().QuoteIdentifier("applied_at") + " DATETIME" +
		")"
	if _, err := sqlDB.ExecContext(ctx, createTrackingSQL); err != nil {
		return fmt.Errorf("migrate: failed to create tracking table: %w", err)
	}

	// 2. For each migration, apply or roll back based on direction.
	for _, m := range migrations {
		version := m.Version()

		switch dir {
		case Up:
			if err := applyUp(ctx, db, sqlDB, m, version); err != nil {
				return err
			}
		case Down:
			if err := applyDown(ctx, db, sqlDB, m, version); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyUp applies a single migration in the forward direction.
func applyUp(ctx context.Context, db *ego.DB, sqlDB *sql.DB, m Migration, version string) error {
	// Check if already applied.
	applied, err := isApplied(ctx, db, sqlDB, version)
	if err != nil {
		return err
	}
	if applied {
		return nil // skip
	}

	// Build the schema (collects DDL statements).
	s := &Schema{db: db}
	m.Up(s)

	// Execute all DDL statements.
	for _, stmt := range s.statements {
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: version %s up failed: %w\nSQL: %s", version, err, stmt)
		}
	}

	// Record version in schema_migrations.
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (%s, %s)",
		db.Dialect().QuoteIdentifier("schema_migrations"),
		db.Dialect().QuoteIdentifier("version"),
		db.Dialect().QuoteIdentifier("applied_at"),
		db.Dialect().Placeholder(1),
		db.Dialect().Placeholder(2),
	)
	if _, err := sqlDB.ExecContext(ctx, insertSQL, version, time.Now().UTC()); err != nil {
		return fmt.Errorf("migrate: failed to record version %s: %w", version, err)
	}

	return nil
}

// applyDown rolls back a single migration.
func applyDown(ctx context.Context, db *ego.DB, sqlDB *sql.DB, m Migration, version string) error {
	// Check if applied; skip if not.
	applied, err := isApplied(ctx, db, sqlDB, version)
	if err != nil {
		return err
	}
	if !applied {
		return nil // skip
	}

	// Build the schema (collects DDL statements).
	s := &Schema{db: db}
	m.Down(s)

	// Execute all DDL statements.
	for _, stmt := range s.statements {
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: version %s down failed: %w\nSQL: %s", version, err, stmt)
		}
	}

	// Remove version from schema_migrations.
	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		db.Dialect().QuoteIdentifier("schema_migrations"),
		db.Dialect().QuoteIdentifier("version"),
		db.Dialect().Placeholder(1),
	)
	if _, err := sqlDB.ExecContext(ctx, deleteSQL, version); err != nil {
		return fmt.Errorf("migrate: failed to remove version %s: %w", version, err)
	}

	return nil
}

// isApplied checks whether a migration version has already been recorded.
func isApplied(ctx context.Context, db *ego.DB, sqlDB *sql.DB, version string) (bool, error) {
	querySQL := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = %s",
		db.Dialect().QuoteIdentifier("schema_migrations"),
		db.Dialect().QuoteIdentifier("version"),
		db.Dialect().Placeholder(1),
	)
	var count int
	if err := sqlDB.QueryRowContext(ctx, querySQL, version).Scan(&count); err != nil {
		return false, fmt.Errorf("migrate: failed to check version %s: %w", version, err)
	}
	return count > 0, nil
}
