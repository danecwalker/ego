# ego

**Entity GO** -- A type-safe, code-first ORM for Go.

[![CI](https://github.com/danecwalker/ego/actions/workflows/ci.yml/badge.svg)](https://github.com/danecwalker/ego/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/danecwalker/ego.svg)](https://pkg.go.dev/github.com/danecwalker/ego)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- No struct tags -- fluent `EntityBuilder` API for schema configuration
- Type-safe queries using Go generics
- All relationship types: HasMany, BelongsTo, HasOne, ManyToMany
- Eager loading via `Include()` with efficient IN-clause queries
- Automatic timestamps (CreatedAt, UpdatedAt)
- Convention over configuration (snake_case columns, pluralized tables)
- Transaction support with auto commit/rollback
- Lifecycle hooks (Before/After Create, Update, Delete)
- Middleware chain for cross-cutting concerns
- Raw SQL escape hatch when you need it
- SQLite and PostgreSQL support
- Explicit versioned migrations

## Install

```bash
go get github.com/danecwalker/ego
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/danecwalker/ego"
	"github.com/danecwalker/ego/sqlite"
)

type User struct {
	ego.Model
	Name  string
	Email string
	Age   int
}

func (u *User) Configure(b *ego.EntityBuilder[User]) {
	b.ToTable("users")
	b.Property(&u.Name).HasMaxLength(100).IsRequired()
	b.Property(&u.Email).HasMaxLength(255).IsRequired().IsUnique()
	b.Property(&u.Age).HasDefault(0)
}

func main() {
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := ego.AutoMigrate(db, &User{}); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create
	user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := ego.Create(db, ctx, user); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created user with ID:", user.ID)

	// Query
	found, err := ego.Query[User](db, ctx).
		Where(ego.Col("email").Eq("alice@example.com")).
		First()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Found:", found.Name)

	// Update
	found.Age = 31
	if err := ego.Update(db, ctx, found); err != nil {
		log.Fatal(err)
	}

	// Delete
	if err := ego.Delete(db, ctx, found); err != nil {
		log.Fatal(err)
	}
}
```

## Defining Entities

Entities are plain Go structs. Embed `ego.Model` to get `ID`, `CreatedAt`, and
`UpdatedAt` fields automatically. Configure schema details by implementing a
`Configure` method on the pointer receiver:

```go
type Author struct {
	ego.Model
	Name    string
	Posts   []Post
	Profile *Profile
}

func (a *Author) Configure(b *ego.EntityBuilder[Author]) {
	b.ToTable("authors")
	b.Property(&a.Name).IsRequired().HasMaxLength(200)
	b.HasMany(&a.Posts)
	b.HasOne(&a.Profile)
}

type Post struct {
	ego.Model
	Title    string
	Body     string
	AuthorID int64
	Author   *Author
}

func (p *Post) Configure(b *ego.EntityBuilder[Post]) {
	b.ToTable("posts")
	b.Property(&p.Title).IsRequired()
	b.BelongsTo(&p.Author)
}

type Tag struct {
	ego.Model
	Label string
}

type Article struct {
	ego.Model
	Title string
	Tags  []Tag
}

func (a *Article) Configure(b *ego.EntityBuilder[Article]) {
	b.ToTable("articles")
	b.Property(&a.Title).IsRequired()
	b.ManyToMany(&a.Tags)
}
```

### Property Configuration

- `IsRequired()` -- marks the column as NOT NULL
- `HasMaxLength(n)` -- sets the maximum length for string columns
- `IsUnique()` -- adds a UNIQUE constraint
- `HasDefault(v)` -- sets a default value

### Relationship Types

| Method       | Field Type   | FK Location    |
|-------------|-------------|----------------|
| `HasMany`    | `[]Related`  | Related table  |
| `BelongsTo`  | `*Related`   | Current table  |
| `HasOne`     | `*Related`   | Related table  |
| `ManyToMany` | `[]Related`  | Pivot table    |

Foreign keys are inferred by convention. `HasMany` and `HasOne` use
`lowercase_parent_id` on the related table. `BelongsTo` uses
`lowercase_related_id` on the current table. `ManyToMany` creates a pivot table
named `owner_relateds` with FK columns `owner_id` and `related_id`.

## Querying

```go
ctx := context.Background()

// Fetch all rows
users, err := ego.Query[User](db, ctx).All()

// Fetch the first matching row (returns ego.ErrNotFound if none)
user, err := ego.Query[User](db, ctx).
	Where(ego.Col("email").Eq("alice@example.com")).
	First()

// Multiple conditions (combined with AND)
users, err := ego.Query[User](db, ctx).
	Where(ego.Col("age").Gte(21)).
	Where(ego.Col("age").Lt(65)).
	OrderBy(ego.Col("name").Asc()).
	Limit(10).
	Offset(20).
	All()

// Count
count, err := ego.Query[User](db, ctx).
	Where(ego.Col("age").Gt(18)).
	Count()
```

### Condition Operators

| Method   | SQL       |
|---------|-----------|
| `Eq(v)`  | `= v`     |
| `Ne(v)`  | `!= v`    |
| `Gt(v)`  | `> v`     |
| `Lt(v)`  | `< v`     |
| `Gte(v)` | `>= v`    |
| `Lte(v)` | `<= v`    |
| `Like(v)`| `LIKE v`  |

### Handling Not Found

`First()` returns `ego.ErrNotFound` when no rows match:

```go
user, err := ego.Query[User](db, ctx).
	Where(ego.Col("id").Eq(999)).
	First()
if errors.Is(err, ego.ErrNotFound) {
	// handle missing user
}
```

## Relationships and Eager Loading

Use `Include()` to eager-load related entities. ego loads relationships using
efficient IN-clause queries rather than N+1 selects:

```go
// Load an author with all their posts
author, err := ego.Query[Author](db, ctx).
	Include("Posts").
	First()
fmt.Println(len(author.Posts))

// Load posts with their parent author
posts, err := ego.Query[Post](db, ctx).
	Include("Author").
	All()
for _, p := range posts {
	fmt.Println(p.Title, "by", p.Author.Name)
}

// Multiple includes
author, err := ego.Query[Author](db, ctx).
	Include("Posts").
	Include("Profile").
	First()
```

### ManyToMany with Associate

For ManyToMany relationships, use `ego.Associate` to insert pivot table entries
after creating both sides:

```go
article := &Article{Title: "Go Generics"}
ego.Create(db, ctx, article)

tag1 := &Tag{Label: "golang"}
tag2 := &Tag{Label: "generics"}
ego.Create(db, ctx, tag1)
ego.Create(db, ctx, tag2)

// Insert pivot rows linking article to both tags
ego.Associate(db, ctx, article, tag1, tag2)

// Eager-load the tags
result, err := ego.Query[Article](db, ctx).
	Where(ego.Col("id").Eq(article.ID)).
	Include("Tags").
	First()
fmt.Println(result.Tags) // [{ID:1 Label:golang} {ID:2 Label:generics}]
```

## Transactions

`ego.Transaction` starts a transaction, calls your function, and auto-commits on
success or rolls back on error. If the function panics, the transaction is rolled
back and the panic is re-raised:

```go
err := ego.Transaction(db, ctx, func(tx *ego.Tx) error {
	alice := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := ego.Create(tx, ctx, alice); err != nil {
		return err // triggers rollback
	}

	bob := &User{Name: "Bob", Email: "bob@example.com", Age: 25}
	if err := ego.Create(tx, ctx, bob); err != nil {
		return err // triggers rollback
	}

	return nil // triggers commit
})
```

All CRUD functions (`Create`, `Update`, `Delete`) and `Query` work identically
inside transactions -- pass the `*ego.Tx` instead of `*ego.DB`.

## Lifecycle Hooks

Implement hook interfaces on your entity to run logic before or after database
operations:

| Interface       | Method                              | Runs                  |
|----------------|-------------------------------------|-----------------------|
| `BeforeCreator` | `BeforeCreate(ctx) error`           | Before INSERT         |
| `AfterCreator`  | `AfterCreate(ctx) error`            | After INSERT          |
| `BeforeUpdater` | `BeforeUpdate(ctx) error`           | Before UPDATE         |
| `AfterUpdater`  | `AfterUpdate(ctx) error`            | After UPDATE          |
| `BeforeDeleter` | `BeforeDelete(ctx) error`           | Before DELETE         |
| `AfterDeleter`  | `AfterDelete(ctx) error`            | After DELETE          |

```go
type User struct {
	ego.Model
	Email string
}

func (u *User) BeforeCreate(ctx context.Context) error {
	u.Email = strings.ToLower(u.Email)
	return nil
}
```

## Middleware

Register middleware on the `*ego.DB` to wrap all Create, Update, and Delete
operations. Middleware receives an `*ego.Operation` containing the operation type,
entity, SQL, and arguments:

```go
db.Use(func(next ego.HandlerFunc) ego.HandlerFunc {
	return func(op *ego.Operation) error {
		start := time.Now()
		err := next(op)
		log.Printf("[%s] %s (%v) err=%v", op.Type, op.SQL, time.Since(start), err)
		return err
	}
})
```

Middlewares execute in registration order (first registered = outermost).

## Raw SQL

When the query builder does not cover your use case, drop down to raw SQL:

```go
// Raw query with type-safe scanning
var users []User
err := ego.Raw[User](db, ctx, "SELECT * FROM users WHERE age > ?", 21).Scan(&users)

// Raw exec for INSERT, UPDATE, DELETE, or DDL
result, err := ego.RawExec(db, ctx, "DELETE FROM users WHERE age < ?", 18)
```

`Raw[T]` maps returned columns to struct fields by matching column names to
the entity's schema. Columns not present in the schema are discarded.

## PostgreSQL

Swap the import to switch from SQLite to PostgreSQL:

```go
import (
	"github.com/danecwalker/ego"
	"github.com/danecwalker/ego/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

db, err := ego.Open(postgres.New("postgres://user:pass@localhost/mydb"))
```

Everything else stays the same -- entity definitions, queries, transactions, and
hooks work identically across both dialects.

## Benchmarks

Results on an AMD Ryzen 9 3900X (SQLite in-memory):

```
goos: windows
goarch: amd64
cpu: AMD Ryzen 9 3900X 12-Core Processor
BenchmarkSchemaParseAndRegister-24       524793       2189 ns/op      2064 B/op       28 allocs/op
BenchmarkCreate-24                        42638      27120 ns/op      2058 B/op       48 allocs/op
BenchmarkQueryAll-24                       4101     325235 ns/op     91656 B/op     1658 allocs/op
BenchmarkQueryFirst-24                    52489      24534 ns/op      1673 B/op       48 allocs/op
BenchmarkQueryWhere-24                     5083     242677 ns/op     57296 B/op     1166 allocs/op
BenchmarkUpdate-24                        42999      28459 ns/op      1970 B/op       58 allocs/op
BenchmarkDelete-24                        25384      49881 ns/op      2514 B/op       64 allocs/op
BenchmarkIncludeHasMany-24                 4203     265301 ns/op     50952 B/op     1059 allocs/op
BenchmarkIncludeBelongsTo-24              12210      95995 ns/op     12952 B/op      263 allocs/op
BenchmarkIncludeManyToMany-24              9783     132072 ns/op     18032 B/op      358 allocs/op
```

Run benchmarks yourself:

```bash
go test -bench=. -benchmem -run='^$' .
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on development workflow,
code style, and testing.

## License

[MIT](LICENSE)
