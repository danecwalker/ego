// benchmark_test.go
package ego_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/danecwalker/ego"
	"github.com/danecwalker/ego/sqlite"
)

// setupBenchDB creates an in-memory SQLite database for benchmarks,
// analogous to setupTestDB but accepting *testing.B.
func setupBenchDB(b *testing.B, entities ...any) *ego.DB {
	b.Helper()
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })
	if err := ego.AutoMigrate(db, entities...); err != nil {
		b.Fatal(err)
	}
	return db
}

// BenchmarkSchemaParseAndRegister measures the cost of parsing and building
// an entity schema from a struct definition.
func BenchmarkSchemaParseAndRegister(b *testing.B) {
	for b.Loop() {
		ego.BuildSchema(&SimpleEntity{})
	}
}

// BenchmarkCreate measures single-row insert throughput.
func BenchmarkCreate(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	ctx := context.Background()
	b.ResetTimer()

	for b.Loop() {
		e := &SimpleEntity{Name: "Alice", Email: "alice@example.com", Age: 30}
		if err := ego.Create(db, ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryAll measures querying all rows from a 100-row table.
func BenchmarkQueryAll(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		ego.Create(db, ctx, &SimpleEntity{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i,
		})
	}
	b.ResetTimer()

	for b.Loop() {
		if _, err := ego.Query[SimpleEntity](db, ctx).All(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryFirst measures fetching a single row via First().
func BenchmarkQueryFirst(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		ego.Create(db, ctx, &SimpleEntity{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i,
		})
	}
	b.ResetTimer()

	for b.Loop() {
		if _, err := ego.Query[SimpleEntity](db, ctx).First(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryWhere measures filtered queries using a Where clause.
func BenchmarkQueryWhere(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		ego.Create(db, ctx, &SimpleEntity{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i,
		})
	}
	b.ResetTimer()

	for b.Loop() {
		if _, err := ego.Query[SimpleEntity](db, ctx).
			Where(ego.Col("age").Gt(50)).All(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUpdate measures updating a single entity repeatedly.
func BenchmarkUpdate(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	ctx := context.Background()
	e := &SimpleEntity{Name: "Alice", Email: "alice@example.com", Age: 30}
	ego.Create(db, ctx, e)
	b.ResetTimer()

	for b.Loop() {
		e.Age++
		if err := ego.Update(db, ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDelete measures creating and deleting a row each iteration.
func BenchmarkDelete(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	ctx := context.Background()
	b.ResetTimer()

	for b.Loop() {
		e := &SimpleEntity{Name: "Temp", Email: "temp@example.com", Age: 25}
		if err := ego.Create(db, ctx, e); err != nil {
			b.Fatal(err)
		}
		if err := ego.Delete(db, ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIncludeHasMany measures eager-loading a HasMany relationship.
// Seeds 10 authors with 5 posts each.
func BenchmarkIncludeHasMany(b *testing.B) {
	db := setupBenchDB(b, &Author{}, &Post{}, &Profile{})
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		a := &Author{Name: fmt.Sprintf("Author%d", i)}
		ego.Create(db, ctx, a)
		for j := 0; j < 5; j++ {
			ego.Create(db, ctx, &Post{
				Title:    fmt.Sprintf("Post%d-%d", i, j),
				Body:     "body",
				AuthorID: a.ID,
			})
		}
	}
	b.ResetTimer()

	for b.Loop() {
		if _, err := ego.Query[Author](db, ctx).Include("Posts").All(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIncludeBelongsTo measures eager-loading a BelongsTo relationship.
// Seeds 3 authors and 10 posts spread across them.
func BenchmarkIncludeBelongsTo(b *testing.B) {
	db := setupBenchDB(b, &Author{}, &Post{}, &Profile{})
	ctx := context.Background()
	authors := make([]*Author, 3)
	for i := 0; i < 3; i++ {
		a := &Author{Name: fmt.Sprintf("Author%d", i)}
		ego.Create(db, ctx, a)
		authors[i] = a
	}
	for i := 0; i < 10; i++ {
		ego.Create(db, ctx, &Post{
			Title:    fmt.Sprintf("Post%d", i),
			Body:     "body",
			AuthorID: authors[i%3].ID,
		})
	}
	b.ResetTimer()

	for b.Loop() {
		if _, err := ego.Query[Post](db, ctx).Include("Author").All(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIncludeManyToMany measures eager-loading a ManyToMany relationship.
// Seeds 10 articles with 2 tags each via Associate.
func BenchmarkIncludeManyToMany(b *testing.B) {
	db := setupBenchDB(b, &Article{}, &Tag{})
	ctx := context.Background()
	tags := make([]*Tag, 5)
	for i := 0; i < 5; i++ {
		t := &Tag{Label: fmt.Sprintf("tag%d", i)}
		ego.Create(db, ctx, t)
		tags[i] = t
	}
	for i := 0; i < 10; i++ {
		a := &Article{Title: fmt.Sprintf("Article%d", i)}
		ego.Create(db, ctx, a)
		ego.Associate(db, ctx, a, tags[i%5], tags[(i+1)%5])
	}
	b.ResetTimer()

	for b.Loop() {
		if _, err := ego.Query[Article](db, ctx).Include("Tags").All(); err != nil {
			b.Fatal(err)
		}
	}
}
