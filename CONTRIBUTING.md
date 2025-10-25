# Contributing to UNCORS

Thanks for your interest in contributing! This is a pet project, so contributions are welcome but kept simple.

## How to Contribute

### Reporting Bugs

Found a bug? Open an issue with:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Your UNCORS version, OS, and config (if relevant)

### Suggesting Features

Have an idea? Open an issue describing:

- The problem you're trying to solve
- Your proposed solution
- Why it would be useful

### Pull Requests

1. Fork and create a branch from `main`
2. Make your changes
3. Add tests if applicable
4. Make sure tests pass: `make test`
5. Submit a PR

## Development Setup

**Requirements:**

- Go 1.24.1+
- Make (optional)
- golangci-lint for linting

**Quick start:**

```bash
git clone https://github.com/YOUR_USERNAME/uncors.git
cd uncors
go mod download
make test  # or: go test ./...
```

## Development Commands

```bash
make check       # Run all checks
make test        # Run tests
make test-cover  # Test coverage
make format      # Format code
make build       # Build binary
```

Or use Go commands directly:

```bash
go test ./...
go build ./...
golangci-lint run
```

## Code Style

- Follow standard Go conventions
- Run `make format` before committing
- Code is linted with `golangci-lint` (see [.golangci.yml](.golangci.yml))
- Keep it simple and readable

## Testing

- Add tests for new features
- Place tests in `_test.go` files next to the code
- Run `make test` to verify everything works

## Commit Messages

Keep them simple:

```
feat: add new feature
fix: fix the bug
docs: update documentation
refactor: improve code structure
test: add tests
```

Reference issues when relevant: `Fixes #123` or `Closes #456`

## Questions?

Open an issue or check the [wiki](https://github.com/evg4b/uncors/wiki).
