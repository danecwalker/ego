# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2026-02-24

### Added

- `ego.Model` base type with ID, CreatedAt, UpdatedAt
- Schema reflection engine with convention-based mapping (snake_case columns, pluralized tables)
- `EntityBuilder[T]` fluent configuration API (no struct tags)
- `PropertyBuilder` with HasMaxLength, IsRequired, IsUnique, HasDefault
- SQLite dialect (`ego/sqlite` package) using modernc.org/sqlite
- PostgreSQL dialect (`ego/postgres` package) with $N placeholders and RETURNING support
- `ego.Open()` with functional options (WithMaxOpenConns, WithMaxIdleConns, WithConnMaxLifetime)
- `ego.AutoMigrate()` for CREATE TABLE IF NOT EXISTS DDL generation
- `ego.Create[T]()` with auto-timestamps and ID population
- `ego.Update[T]()` with auto-UpdatedAt
- `ego.Delete[T]()` by primary key
- `ego.Query[T]()` builder with Where, OrderBy, Limit, Offset, Count, First, All
- `ego.Col()` for type-safe conditions (Eq, Gt, Lt, Gte, Lte, Ne, Like)
- HasMany, BelongsTo, HasOne, ManyToMany relationship support
- Eager loading via `Include()` with separate IN-clause queries
- `ego.Associate()` for ManyToMany pivot table inserts
- `ego.Transaction()` with auto commit/rollback and panic recovery
- Lifecycle hooks: BeforeCreate, AfterCreate, BeforeUpdate, AfterUpdate, BeforeDelete, AfterDelete
- Middleware chain for cross-cutting concerns
- `ego.Raw[T]()` and `ego.RawExec()` for raw SQL escape hatch
- Explicit versioned migrations via `ego/migrate` package
