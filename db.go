package ego

import (
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
	sqlDB   *sql.DB
	dialect Dialect
	schemas map[reflect.Type]*EntitySchema
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
		dialect: cfg.Dialect,
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
	return db.dialect
}
