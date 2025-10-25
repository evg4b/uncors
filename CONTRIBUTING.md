# Contributing to UNCORS

First off, thank you for considering contributing to UNCORS! It's people like you that make UNCORS such a great tool.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Commit Message Guidelines](#commit-message-guidelines)

## Code of Conduct

This project and everyone participating in it is governed by the [UNCORS Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When creating a bug report, include as many details as possible using the bug report template.

**Good bug reports include:**

- A clear and descriptive title
- Exact steps to reproduce the problem
- Expected behavior vs actual behavior
- UNCORS version, Go version, and OS
- Configuration file (if applicable)
- Relevant logs or error messages

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- Use a clear and descriptive title
- Provide a detailed description of the proposed enhancement
- Explain why this enhancement would be useful
- Include code examples if applicable

### Pull Requests

1. Fork the repository and create your branch from `main`
2. Follow the [development setup](#development-setup) instructions
3. Make your changes following our [coding standards](#coding-standards)
4. Add tests for your changes
5. Ensure all tests pass
6. Update documentation as needed
7. Submit a pull request using the PR template

## Development Setup

### Prerequisites

- **Go 1.24.1** or later
- **Make** (optional, but recommended)
- **Git**
- **golangci-lint** for code linting
- **gofumpt** for code formatting (optional)

### Setting Up Your Development Environment

1. **Fork and clone the repository:**

```bash
git clone https://github.com/YOUR_USERNAME/uncors.git
cd uncors
```

2. **Install dependencies:**

```bash
go mod download
```

3. **Install development tools:**

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install gofumpt (optional)
go install mvdan.cc/gofumpt@latest
```

4. **Build the project:**

```bash
make build
# or
go build ./...
```

5. **Run tests to verify setup:**

```bash
make test
# or
go test ./...
```

## Development Workflow

### Using Make (Recommended)

The project includes a Makefile with common development tasks:

```bash
# Run all checks (format, test, build)
make check

# Format code
make format

# Run tests
make test

# Run tests with coverage
make test-cover

# Build release binary
make build-release

# Clean generated files
make clean

# Show all available targets
make help
```

### Manual Workflow

```bash
# Format code
gofmt -l -s -w .
gofumpt -l -w .

# Lint code
golangci-lint run

# Run tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out ./...

# Build
go build ./...
```

## Coding Standards

### General Guidelines

- Follow standard Go conventions and idioms
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions small and focused
- Use meaningful variable and function names

### Code Style

The project uses:

- **gofmt** for basic formatting
- **gofumpt** for stricter formatting (optional)
- **golangci-lint** for linting with the configuration in `.golangci.yml`

Run formatting before committing:

```bash
make format
```

### Project Structure

```
uncors/
├── internal/
│   ├── config/       # Configuration loading and validation
│   ├── contracts/    # Interfaces and contracts
│   ├── handler/      # HTTP handlers (proxy, mock, static, script, etc.)
│   ├── helpers/      # Utility functions
│   ├── infra/        # Infrastructure (HTTP client, logger)
│   ├── tui/          # Terminal UI components
│   ├── uncors/       # Main application logic
│   ├── urlparser/    # URL parsing utilities
│   ├── urlreplacer/  # URL replacement logic
│   └── version/      # Version checking
├── testing/          # Test utilities and mocks
├── tests/            # Integration tests
└── docs/             # Documentation
```

### Key Principles

- Place new code in appropriate packages
- Use `internal/` for non-exported packages
- Keep handler logic separate from business logic
- Use dependency injection for testability
- Prefer interfaces for external dependencies

## Testing Guidelines

### Writing Tests

- Write tests for all new functionality
- Aim for high test coverage (80%+ for new code)
- Use table-driven tests where appropriate
- Mock external dependencies using interfaces
- Test both success and error cases

### Test Organization

- Place unit tests in the same package as the code (`_test.go` files)
- Use `testing/mocks` for generated mocks (minimock)
- Place integration tests in `tests/` directory

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with race detector and coverage
go test -race -coverprofile=coverage.out ./...

# Run tests for a specific package
go test ./internal/handler/proxy

# Run a specific test
go test -run TestHandlerName ./internal/handler
```

### Test Coverage

Check coverage after adding tests:

```bash
make test-cover
go tool cover -html=coverage.out
```

## Commit Message Guidelines

### Format

```
<type>: <subject>

<body>

<footer>
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code refactoring without changing functionality
- **test**: Adding or updating tests
- **chore**: Maintenance tasks, dependency updates

### Examples

```
feat: add support for custom CORS headers

Implement custom CORS header configuration in the mapping
section. This allows users to specify custom headers beyond
the default CORS headers.

Closes #123
```

```
fix: handle nil pointer in proxy handler

Add nil check for response body to prevent panic when
upstream server returns empty response.

Fixes #456
```

### Best Practices

- Use the imperative mood ("add feature" not "added feature")
- Keep the subject line under 50 characters
- Capitalize the subject line
- Don't end the subject line with a period
- Separate subject from body with a blank line
- Wrap the body at 72 characters
- Reference issues and pull requests in the footer

## Additional Resources

- [GitHub Flow](https://guides.github.com/introduction/flow/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [UNCORS Wiki](https://github.com/evg4b/uncors/wiki)

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

Thank you for contributing to UNCORS!
