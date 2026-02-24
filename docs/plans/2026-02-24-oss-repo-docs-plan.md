# OSS Repository Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add standard open-source repository files (README, LICENSE, CONTRIBUTING, CHANGELOG, .gitignore, GitHub Actions CI, issue/PR templates) and a comprehensive benchmark suite to the ego Go ORM.

**Architecture:** Create files at repo root and under `.github/`. Benchmarks go in `benchmark_test.go` in the root package, reusing existing test entity types and `setupTestDB` helper from `crud_test.go`.

**Tech Stack:** Go testing/benchmarking (`testing.B`), GitHub Actions, Markdown

---

### Task 1: LICENSE

**Files:**
- Create: `LICENSE`

**Step 1: Create MIT license file**

```text
MIT License

Copyright (c) 2026 Dan Wilson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

**Step 2: Commit**

```bash
git add LICENSE
git commit -m "chore: add MIT license"
```

---

### Task 2: .gitignore

**Files:**
- Create: `.gitignore`

**Step 1: Create Go standard .gitignore**

```text
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of go coverage
*.out

# Go workspace
go.work
go.work.sum

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
```

**Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore"
```

---

### Task 3: CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

**Step 1: Create contributing guide**

The file should cover:
- Prerequisites (Go 1.22+)
- Getting started (clone, `go test ./...`)
- Development workflow (branch, code, test, PR)
- Code style (gofmt, go vet, no struct tags)
- Testing expectations (all new features need tests, use `setupTestDB` pattern)
- PR guidelines (clear description, one concern per PR)

**Step 2: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add contributing guide"
```

---

### Task 4: CHANGELOG.md

**Files:**
- Create: `CHANGELOG.md`

**Step 1: Create changelog**

Document v0.1.0 as the initial release with all current features:
- Core: Model base type, schema reflection, EntityBuilder fluent API
- CRUD: Create, Update, Delete with auto-timestamps
- Query: Type-safe builder with Where, OrderBy, Limit, Offset, Count, First, All
- Relationships: HasMany, BelongsTo, HasOne, ManyToMany with eager loading
- Transactions: Callback-based with auto commit/rollback
- Hooks: Before/After Create/Update/Delete lifecycle hooks
- Middleware: Operation wrapping for cross-cutting concerns
- Raw SQL: Raw query and exec escape hatch
- Migrations: AutoMigrate DDL + explicit versioned migrations
- Dialects: SQLite, PostgreSQL

Format: Keep a Changelog (https://keepachangelog.com/)

**Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: add changelog for v0.1.0"
```

---

### Task 5: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Create CI workflow**

```yaml
name: CI

on:
  push:
    branches: [master]
  pull_request:
    branches: [master]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.22', '1.23', 'stable']

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}

      - name: Vet
        run: go vet ./...

      - name: Build
        run: go build ./...

      - name: Test
        run: go test -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        if: matrix.go-version == 'stable'
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions workflow for test and vet"
```

---

### Task 6: GitHub Issue and PR Templates

**Files:**
- Create: `.github/ISSUE_TEMPLATE/bug_report.md`
- Create: `.github/ISSUE_TEMPLATE/feature_request.md`
- Create: `.github/PULL_REQUEST_TEMPLATE.md`

**Step 1: Create bug report template**

```markdown
---
name: Bug Report
about: Report a bug in ego
title: ''
labels: bug
---

## Description
A clear description of the bug.

## Steps to Reproduce
1. ...
2. ...

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- Go version:
- ego version:
- Database: SQLite / PostgreSQL
- OS:

## Code Sample
```go
// Minimal reproducing code
```
```

**Step 2: Create feature request template**

```markdown
---
name: Feature Request
about: Suggest an idea for ego
title: ''
labels: enhancement
---

## Problem
What problem does this solve?

## Proposed Solution
How should ego handle this?

## Alternatives Considered
What else did you consider?

## Additional Context
Any other context (code samples, links, etc).
```

**Step 3: Create PR template**

```markdown
## What
Brief description of changes.

## Why
What problem does this solve?

## How
Key implementation details.

## Checklist
- [ ] Tests added/updated
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] No new dependencies unless discussed
```

**Step 4: Commit**

```bash
git add .github/
git commit -m "chore: add GitHub issue and PR templates"
```

---

### Task 7: Benchmark Suite

**Files:**
- Create: `benchmark_test.go`

**Context:** The benchmark file lives in `package ego_test` alongside the existing test files. It reuses entity types already defined in other test files (`SimpleEntity` from `crud_test.go`, `Author`/`Post`/`Profile` from `relationship_test.go`, `Article`/`Tag` from `relationship_test.go`). The `setupTestDB` helper from `crud_test.go` is also available.

**Step 1: Write benchmark file**

The file must define a `setupBenchDB` helper (like `setupTestDB` but takes `*testing.B` and uses `b.Helper()`/`b.Fatal()`). Then implement these benchmarks:

```go
package ego_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/danewilson/ego"
	"github.com/danewilson/ego/sqlite"
)

// setupBenchDB creates an in-memory SQLite DB for benchmarks.
func setupBenchDB(b *testing.B, entities ...any) *ego.DB {
	b.Helper()
	db, err := ego.Open(sqlite.New(":memory:"))
	if err != nil {
		b.Fatal(err)
	}
	if err := ego.AutoMigrate(db, entities...); err != nil {
		b.Fatal(err)
	}
	return db
}

func BenchmarkSchemaParseAndRegister(b *testing.B) {
	// Measures the one-time cost of reflecting a struct into a schema.
	for b.Loop() {
		ego.BuildSchema(&SimpleEntity{})
	}
}

func BenchmarkCreate(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	defer db.Close()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		e := &SimpleEntity{Name: "bench", Email: "bench@test.com", Age: 25}
		if err := ego.Create(db, ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryAll(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	defer db.Close()
	ctx := context.Background()

	// Seed 100 rows.
	for i := 0; i < 100; i++ {
		ego.Create(db, ctx, &SimpleEntity{
			Name:  fmt.Sprintf("user_%d", i),
			Email: fmt.Sprintf("user_%d@test.com", i),
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

func BenchmarkQueryFirst(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	defer db.Close()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		ego.Create(db, ctx, &SimpleEntity{
			Name:  fmt.Sprintf("user_%d", i),
			Email: fmt.Sprintf("user_%d@test.com", i),
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

func BenchmarkQueryWhere(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	defer db.Close()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		ego.Create(db, ctx, &SimpleEntity{
			Name:  fmt.Sprintf("user_%d", i),
			Email: fmt.Sprintf("user_%d@test.com", i),
			Age:   20 + i,
		})
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := ego.Query[SimpleEntity](db, ctx).
			Where(ego.Col("age").Gt(50)).
			All(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdate(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	defer db.Close()
	ctx := context.Background()

	e := &SimpleEntity{Name: "bench", Email: "bench@test.com", Age: 25}
	ego.Create(db, ctx, e)

	b.ResetTimer()
	for b.Loop() {
		e.Age++
		if err := ego.Update(db, ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	db := setupBenchDB(b, &SimpleEntity{})
	defer db.Close()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		e := &SimpleEntity{Name: "bench", Email: "bench@test.com", Age: 25}
		ego.Create(db, ctx, e)
		if err := ego.Delete(db, ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIncludeHasMany(b *testing.B) {
	db := setupBenchDB(b, &Author{}, &Post{}, &Profile{})
	defer db.Close()
	ctx := context.Background()

	// Seed 10 authors with 5 posts each.
	for i := 0; i < 10; i++ {
		a := &Author{Name: fmt.Sprintf("author_%d", i)}
		ego.Create(db, ctx, a)
		for j := 0; j < 5; j++ {
			ego.Create(db, ctx, &Post{
				Title:    fmt.Sprintf("post_%d_%d", i, j),
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

func BenchmarkIncludeBelongsTo(b *testing.B) {
	db := setupBenchDB(b, &Author{}, &Post{}, &Profile{})
	defer db.Close()
	ctx := context.Background()

	// Seed 10 posts belonging to 3 authors.
	authors := make([]*Author, 3)
	for i := range authors {
		authors[i] = &Author{Name: fmt.Sprintf("author_%d", i)}
		ego.Create(db, ctx, authors[i])
	}
	for i := 0; i < 10; i++ {
		ego.Create(db, ctx, &Post{
			Title:    fmt.Sprintf("post_%d", i),
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

func BenchmarkIncludeManyToMany(b *testing.B) {
	db := setupBenchDB(b, &Article{}, &Tag{})
	defer db.Close()
	ctx := context.Background()

	// Seed 5 tags and 10 articles, each with 2-3 tags.
	tags := make([]*Tag, 5)
	for i := range tags {
		tags[i] = &Tag{Label: fmt.Sprintf("tag_%d", i)}
		ego.Create(db, ctx, tags[i])
	}
	for i := 0; i < 10; i++ {
		a := &Article{Title: fmt.Sprintf("article_%d", i)}
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
```

**Step 2: Run benchmarks to verify they work**

Run: `go test -bench=. -benchmem -count=1 -run=^$ ./...`
Expected: All benchmarks complete without error. Note the output for use in the README.

**Step 3: Commit**

```bash
git add benchmark_test.go
git commit -m "test: add comprehensive benchmark suite"
```

---

### Task 8: README.md

**Files:**
- Create: `README.md`

**Context:** This is the main project page. It must be accurate to the actual API as implemented. Reference the existing test files for correct usage patterns. The README should use code examples that compile — pull patterns directly from `integration_test.go` and `relationship_test.go`.

**Step 1: Write README**

Structure:
1. **Header:** `# ego` with tagline "Entity GO — A type-safe, code-first ORM for Go" and badges (Go Reference, CI, License)
2. **Features:** Bullet list of key features (no struct tags, generics, type-safe queries, all relationship types, transactions, hooks, middleware, raw SQL, migrations, SQLite + PostgreSQL)
3. **Install:** `go get github.com/danewilson/ego`
4. **Quick Start:** Complete working example showing: define entity with `ego.Model` embedding + `Configure` method, open SQLite DB, AutoMigrate, Create, Query with Where, Update, Delete
5. **Defining Entities:** Show the `Configure(*EntityBuilder[T])` pattern with Property, HasMany, BelongsTo, HasOne, ManyToMany
6. **Querying:** All, First, Where, OrderBy, Limit, Offset, Count
7. **Relationships & Eager Loading:** Include("Posts"), Include("Author"), ManyToMany with Associate
8. **Transactions:** `ego.Transaction(db, ctx, func(tx *ego.Tx) error { ... })`
9. **Lifecycle Hooks:** BeforeCreator, AfterCreator, etc. with example
10. **Middleware:** `db.Use(func(next ego.HandlerFunc) ego.HandlerFunc { ... })`
11. **Raw SQL:** `ego.Raw[T]` and `ego.RawExec`
12. **PostgreSQL:** Show postgres dialect usage
13. **Benchmarks:** How to run (`go test -bench=. -benchmem`) with sample output from Task 7
14. **Contributing:** Link to CONTRIBUTING.md
15. **License:** MIT

**Important API details to get right (from reading the source):**
- Free functions: `ego.Create(db, ctx, entity)`, not `db.Create(ctx, entity)`
- `ego.Query[T](db, ctx).Where(...).All()`
- `ego.Col("name").Eq(value)` for conditions
- `ego.Col("name").Asc()` for ordering
- `ego.Transaction(db, ctx, fn)` returns error
- `ego.AutoMigrate(db, entities...)` takes variadic `any`
- `ego.Open(sqlite.New(":memory:"))` returns `(*ego.DB, error)`
- Entities implement `Configure(b *ego.EntityBuilder[T])` — note: method on pointer receiver
- `ego.Associate(db, ctx, owner, related1, related2)` for M:N pivot inserts
- Import paths: `github.com/danewilson/ego`, `github.com/danewilson/ego/sqlite`, `github.com/danewilson/ego/postgres`

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add comprehensive README with examples and benchmarks"
```

---

### Task 9: Final Verification

**Step 1: Run full test suite**

```bash
go vet ./...
go build ./...
go test ./... -count=1
```

Expected: All pass, including new benchmarks.

**Step 2: Run benchmarks**

```bash
go test -bench=. -benchmem -count=1 -run=^$ .
```

Expected: All benchmarks complete. Copy output for README if needed.

**Step 3: Verify all files present**

```bash
ls -la LICENSE .gitignore CONTRIBUTING.md CHANGELOG.md README.md
ls -la .github/workflows/ci.yml .github/ISSUE_TEMPLATE/ .github/PULL_REQUEST_TEMPLATE.md
ls -la benchmark_test.go
```

**Step 4: Commit any final README updates with benchmark numbers**

```bash
git add README.md
git commit -m "docs: update README with benchmark results"
```
