package ego

import "time"

type options struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
}

// Option configures the DB connection pool.
type Option func(*options)

// WithMaxOpenConns sets the maximum number of open connections to the database.
func WithMaxOpenConns(n int) Option {
	return func(o *options) { o.maxOpenConns = n }
}

// WithMaxIdleConns sets the maximum number of idle connections in the pool.
func WithMaxIdleConns(n int) Option {
	return func(o *options) { o.maxIdleConns = n }
}

// WithConnMaxLifetime sets the maximum amount of time a connection may be reused.
func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *options) { o.connMaxLifetime = d }
}
