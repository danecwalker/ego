# Contributing to ego

Thank you for your interest in contributing to ego! This guide will help you
get started.

## Prerequisites

- **Go 1.22+** (see [go.dev/dl](https://go.dev/dl/))
- Git

## Getting Started

1. Fork and clone the repository:

   ```bash
   git clone https://github.com/<your-fork>/ego.git
   cd ego
   ```

2. Verify everything works:

   ```bash
   go test ./...
   ```

## Development Workflow

1. Create a branch from `master`:

   ```bash
   git checkout -b my-feature master
   ```

2. Write tests first, then implement your feature or fix.

3. Before committing, run the full check:

   ```bash
   go vet ./... && go test ./...
   ```

4. Commit with a clear, descriptive message.

5. Open a pull request against `master`.

## Code Style

- Format all code with `gofmt` (or `goimports`).
- Run `go vet ./...` and fix any warnings.
- **No struct tags.** ego uses a fluent `EntityBuilder` API for configuration
  instead of struct tags. Define column metadata through `PropertyBuilder`
  methods like `HasMaxLength`, `IsRequired`, and `HasDefault`.

## Testing

- All new features and bug fixes must include tests.
- Use in-memory SQLite for tests via the `setupTestDB` helper pattern found
  throughout the test suite. This keeps tests fast and self-contained.
- Run the full suite with:

  ```bash
  go test ./...
  ```

## Pull Request Guidelines

- Provide a clear description of what your PR does and why.
- Keep each PR focused on a single concern.
- All tests must pass before a PR will be reviewed.
- If your change affects public API, update or add relevant documentation.

## Questions?

Open an issue if you have questions or need help getting started.
