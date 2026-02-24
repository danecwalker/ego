package ego

import (
	"context"
	"database/sql"
	"reflect"
)

// DriverConfig holds database driver configuration returned by dialect constructors.
type DriverConfig struct {
	DriverName string
	DSN        string
	Dialect    Dialect
}

// DB wraps a *sql.DB with ego metadata (dialect, schema registry).
type DB struct {
	sqlDB       *sql.DB
	dial        Dialect
	schemas     map[reflect.Type]*EntitySchema
	middlewares []MiddlewareFunc
}

// Open creates a new DB connection using the provided driver configuration.
func Open(cfg DriverConfig, opts ...Option) (*DB, error) {
	sqlDB, err := sql.Open(cfg.DriverName, cfg.DSN)
	if err != nil {
		return nil, err
	}

	// For SQLite, default to a single open connection to avoid issues
	// with in-memory databases (each connection gets its own database).
	// User-provided options will override this default.
	if cfg.Dialect != nil && cfg.Dialect.Name() == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	}

	// Apply user options (these override defaults above)
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	if o.maxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(o.maxOpenConns)
	}
	if o.maxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(o.maxIdleConns)
	}
	if o.connMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(o.connMaxLifetime)
	}

	// Verify connectivity
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return &DB{
		sqlDB:   sqlDB,
		dial:    cfg.Dialect,
		schemas: make(map[reflect.Type]*EntitySchema),
	}, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.sqlDB.Close()
}

// SqlDB returns the underlying *sql.DB.
func (db *DB) SqlDB() *sql.DB {
	return db.sqlDB
}

// Dialect returns the dialect associated with this database.
func (db *DB) Dialect() Dialect {
	return db.dial
}

// Executor interface methods — these allow *DB to satisfy Executor.

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.sqlDB.ExecContext(ctx, query, args...)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.sqlDB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.sqlDB.QueryRowContext(ctx, query, args...)
}

func (db *DB) dialect() Dialect { return db.dial }

func (db *DB) schemaFor(t reflect.Type) *EntitySchema { return db.schemas[t] }

func (db *DB) registerSchema(t reflect.Type, s *EntitySchema) { db.schemas[t] = s }

func (db *DB) getMiddlewares() []MiddlewareFunc { return db.middlewares }

// Use registers a middleware that will be executed on Create, Update, and
// Delete operations. Middlewares are called in the order they are registered
// (first registered = outermost wrapper, runs first).
func (db *DB) Use(m MiddlewareFunc) {
	db.middlewares = append(db.middlewares, m)
}

// TableExists checks whether the given table name exists in the database.
// The check is dialect-aware; for SQLite it queries sqlite_master.
func (db *DB) TableExists(tableName string) bool {
	var name string
	var err error

	switch db.dial.Name() {
	case "sqlite":
		err = db.sqlDB.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			tableName,
		).Scan(&name)
	default:
		// Generic fallback using information_schema (PostgreSQL, MySQL, etc.)
		err = db.sqlDB.QueryRow(
			"SELECT table_name FROM information_schema.tables WHERE table_name=?",
			tableName,
		).Scan(&name)
	}

	return err == nil && name == tableName
}

// ColumnNames returns the column names for the given table, in ordinal order.
// The implementation is dialect-aware; for SQLite it uses PRAGMA table_info.
func (db *DB) ColumnNames(tableName string) []string {
	var cols []string

	switch db.dial.Name() {
	case "sqlite":
		rows, err := db.sqlDB.Query("PRAGMA table_info(" + db.dial.QuoteIdentifier(tableName) + ")")
		if err != nil {
			return nil
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue *string
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				return nil
			}
			cols = append(cols, name)
		}
	}

	return cols
}
