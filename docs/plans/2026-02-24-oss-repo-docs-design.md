# ego OSS Repository Documentation Design

**Goal:** Add standard open-source repository files to make ego a polished, contributor-friendly Go library.

**Audience:** Go developers familiar with ORMs, evaluating ego vs GORM/Ent/sqlx/sqlc.

**License:** MIT, copyright Dan Wilson 2026.

---

## Files to Create

| File | Purpose |
|------|---------|
| `README.md` | Badges, overview, install, examples, API reference, benchmarks |
| `LICENSE` | MIT license |
| `CONTRIBUTING.md` | Setup, testing, PR guidelines |
| `CHANGELOG.md` | Version history starting with v0.1.0 |
| `.gitignore` | Go standard ignores |
| `.github/workflows/ci.yml` | Go test + vet on push/PR |
| `.github/ISSUE_TEMPLATE/bug_report.md` | Bug report template |
| `.github/ISSUE_TEMPLATE/feature_request.md` | Feature request template |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR checklist template |
| `benchmark_test.go` | Full benchmark suite |

## README Structure

1. Project name + tagline + badges (Go Reference, CI, License)
2. Features list (no struct tags, generics, type-safe queries, relationships, migrations, etc.)
3. Quick Start (install, define entity, open DB, migrate, CRUD)
4. Full API examples (Query builder, relationships, transactions, hooks, middleware, raw SQL)
5. Benchmarks section (how to run + sample output)
6. Comparison table vs GORM/Ent/sqlx
7. Contributing link
8. License

## Benchmarks

File: `benchmark_test.go` using in-memory SQLite.

| Benchmark | Measures |
|-----------|----------|
| `BenchmarkSchemaParseAndRegister` | One-time schema reflection cost |
| `BenchmarkCreate` | Single INSERT with timestamps |
| `BenchmarkQueryAll` | SELECT all (100 pre-seeded rows) |
| `BenchmarkQueryFirst` | SELECT LIMIT 1 |
| `BenchmarkQueryWhere` | SELECT with WHERE condition |
| `BenchmarkUpdate` | UPDATE single row |
| `BenchmarkDelete` | DELETE single row |
| `BenchmarkIncludeHasMany` | Eager load HasMany (10 parents, 5 children each) |
| `BenchmarkIncludeBelongsTo` | Eager load BelongsTo (10 entities) |
| `BenchmarkIncludeManyToMany` | Eager load ManyToMany via pivot |

## GitHub Actions CI

- Trigger: push to master, pull requests
- Matrix: Go 1.22.x, 1.23.x, latest
- Steps: checkout, setup-go, go vet, go build, go test -race
