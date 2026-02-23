# ego — Entity Framework for Go

**Date**: 2026-02-24
**Module**: `github.com/danewilson/ego`
**Status**: Design approved

## Overview

ego is a full-featured ORM for Go inspired by .NET EF Core and Drizzle, designed to be Go-idiomatic. It provides code-first schema definition, type-safe query building with generics, automatic and explicit migrations, relationship management, transactions, lifecycle hooks, and middleware.

## Target Databases

- PostgreSQL (production)
- SQLite (development/testing)

## Architecture

Monolithic module with a `Dialect` interface for database-specific SQL generation.

```
ego/               - core: DB, Query, Create, Update, Delete, schema, migrations
ego/dialect/       - Dialect interface definition
ego/postgres/      - PostgreSQL dialect
ego/sqlite/        - SQLite dialect
ego/migrate/       - Migration engine
ego/internal/      - Internal utilities (reflection, SQL building)
```

## Schema Definition

Models are plain Go structs — no struct tags. Configuration via fluent EntityBuilder interface (like EF Core's `IEntityTypeConfiguration`).

```go
type User struct {
    ego.Model
    Name    string
    Email   string
    Age     int
    Posts   []Post
    Profile *Profile
    Tags    []Tag
}

func (u *User) Configure(b ego.EntityBuilder[User]) {
    b.ToTable("users")
    b.Property(&u.Name).HasMaxLength(255).IsRequired()
    b.Property(&u.Email).HasMaxLength(255).IsRequired().IsUnique()
    b.Property(&u.Age).HasDefault(0)
    b.HasMany(&u.Posts).WithForeignKey(&Post{}.UserID)
    b.HasOne(&u.Profile)
    b.ManyToMany(&u.Tags)
}
```

Key points:
- `ego.Model` provides `ID`, `CreatedAt`, `UpdatedAt` fields
- `ego.Model` is optional — any struct with a primary key property works
- Configuration uses field pointers for compile-time safety
- Reflection at registration time only, results cached
- Convention-based defaults (table name = plural lowercase, FK = TypeNameID)

## Query API

Type-safe builder pattern with Go generics. Free functions rather than methods on context.

```go
db, err := ego.Open(postgres.New("postgres://localhost/mydb"))

// CREATE
user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
err = ego.Create(db, ctx, user)

// READ
users, err := ego.Query[User](db, ctx).
    Where(ego.Col("age").Gt(18)).
    Where(ego.Col("name").Like("A%")).
    OrderBy(ego.Col("name").Asc()).
    Limit(10).
    Offset(20).
    Include("Posts").
    Include("Profile").
    All()

user, err := ego.Query[User](db, ctx).
    Where(ego.Col("id").Eq(1)).
    First()

// UPDATE
user.Name = "Alice Updated"
err = ego.Update(db, ctx, user)

// DELETE
err = ego.Delete(db, ctx, user)

// RAW SQL
var results []User
err = ego.Raw[User](db, ctx, "SELECT * FROM users WHERE age > $1", 18).Scan(&results)
```

## Transactions

Callback pattern for automatic commit/rollback:

```go
err := ego.Transaction(db, ctx, func(tx ego.Tx) error {
    user := &User{Name: "Alice", Email: "alice@example.com"}
    if err := ego.Create(tx, ctx, user); err != nil {
        return err
    }
    post := &Post{Title: "Hello", Body: "World", UserID: user.ID}
    return ego.Create(tx, ctx, post)
})
```

`ego.Tx` satisfies the same interface as `ego.DB` so all CRUD functions work with both.

## Lifecycle Hooks

Optional interfaces on model structs:

```go
func (u *User) BeforeCreate(ctx context.Context) error {
    u.Email = strings.ToLower(u.Email)
    return nil
}

func (u *User) AfterCreate(ctx context.Context) error {
    log.Printf("User created: %d", u.ID)
    return nil
}
```

Supported hooks: `BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, `AfterDelete`.

## Middleware

Handler chain pattern for cross-cutting concerns:

```go
db.Use(ego.Logger(log.Default()))
db.Use(ego.SlowQueryLog(100 * time.Millisecond))
db.Use(func(next ego.Handler) ego.Handler {
    return func(op ego.Operation) error {
        start := time.Now()
        err := next(op)
        return err
    }
})
```

## Migrations

### Auto-migration (development)

```go
err := ego.AutoMigrate(db, ctx, &User{}, &Post{}, &Profile{}, &Tag{})
```

### Explicit migrations (production)

```go
type AddUsersTable struct{}

func (m *AddUsersTable) Up(s ego.Schema) {
    s.CreateTable("users", func(t ego.Table) {
        t.BigInt("id").PrimaryKey().AutoIncrement()
        t.String("name", 255).NotNull()
        t.String("email", 255).NotNull().Unique()
        t.Int("age").Default(0)
        t.Timestamps()
    })
}

func (m *AddUsersTable) Down(s ego.Schema) {
    s.DropTable("users")
}

err := ego.Migrate(db, ctx, ego.Up)
err := ego.Migrate(db, ctx, ego.Down)
```

## Connection Pooling

Delegates to `database/sql` pool management:

```go
db, err := ego.Open(postgres.New("postgres://localhost/mydb"),
    ego.WithMaxOpenConns(25),
    ego.WithMaxIdleConns(5),
    ego.WithConnMaxLifetime(5 * time.Minute),
)
```

## Relationships

| Type | Definition | Convention |
|------|-----------|------------|
| One-to-Many | `b.HasMany(&u.Posts).WithForeignKey(&Post{}.UserID)` | FK on child |
| Many-to-One | `b.BelongsTo(&p.User)` | FK on self |
| One-to-One | `b.HasOne(&u.Profile)` | FK on related |
| Many-to-Many | `b.ManyToMany(&u.Tags)` | Auto pivot table `user_tags` |

## Design Principles

1. **Go-idiomatic**: Use interfaces, generics, `context.Context`, explicit error returns
2. **No struct tags**: All configuration via fluent API
3. **Convention over configuration**: Sensible defaults, override when needed
4. **Type-safe**: Generics for compile-time safety, no `interface{}` in public API
5. **No code generation**: Pure runtime reflection (at startup) + generics
6. **Standard library friendly**: Built on `database/sql`, uses `context.Context`
7. **Testable**: Interface-based design, SQLite dialect for testing
