# Contributing to Flux

Thank you for your interest in contributing to Flux! Every contribution matters, whether it is a bug report, feature request, documentation improvement, or code change. This guide will help you get started.

## Table of Contents

- [Development Setup](#development-setup)
- [Code Style](#code-style)
- [Testing](#testing)
- [Pull Request Workflow](#pull-request-workflow)
- [Issue Guidelines](#issue-guidelines)
- [Commit Convention](#commit-convention)

## Development Setup

### Prerequisites

- **Go 1.26+** is required. Install it from [go.dev](https://go.dev/dl/).
- Git

### Getting Started

1. Fork the repository on GitHub.
2. Clone your fork:

   ```bash
   git clone https://github.com/<your-username>/flux.git
   cd flux
   ```

3. Verify everything works:

   ```bash
   go test ./...
   ```

## Code Style

All code must pass the standard Go formatting and vetting tools:

- **Format**: Run `gofmt` on all `.go` files. Code that is not properly formatted will not be accepted.

  ```bash
  gofmt -w .
  ```

- **Vet**: Run `go vet` to catch common mistakes.

  ```bash
  go vet ./...
  ```

Please ensure both commands pass cleanly before submitting a pull request.

## Testing

### Unit Tests

Flux requires **100% test coverage**. Run the full test suite with coverage:

```bash
go test -cover ./... -coverprofile=coverage.out
```

Review the coverage report to identify any untested code paths:

```bash
go tool cover -func=coverage.out
```

### Integration Tests

Integration tests for the optimizer package are gated behind a build tag. Run them with:

```bash
go test -tags integration ./optimizer/
```

### Testing Checklist

- All existing tests must pass.
- New code must include tests that maintain 100% coverage.
- Integration tests must pass when applicable.

## Pull Request Workflow

1. **Fork** the repository and create a new branch from `main`:

   ```bash
   git checkout -b feat/my-new-feature
   ```

2. **Write tests** for the changes you plan to make.
3. **Implement** your changes.
4. **Ensure 100% coverage** by running the full test suite:

   ```bash
   go test -cover ./... -coverprofile=coverage.out
   ```

5. **Run formatting and vetting**:

   ```bash
   gofmt -w .
   go vet ./...
   ```

6. **Commit** your changes following the [commit convention](#commit-convention).
7. **Push** your branch and open a pull request against `main`.

A maintainer will review your PR. Please be responsive to feedback and make requested changes promptly.

## Issue Guidelines

When opening an issue, please:

- Use the appropriate issue template if one is available.
- Include your **Go version** (`go version`).
- Include your **operating system** and architecture.
- Provide a clear description of the problem or feature request.
- For bugs, include steps to reproduce the issue and the expected vs. actual behavior.
- For feature requests, explain the use case and why it would benefit the project.

## Commit Convention

Flux uses the [gitmoji](https://gitmoji.dev/) convention. Each commit message must start with a gitmoji emoji:

```
<gitmoji> <short description>
```

### Common Gitmoji

| Emoji | Description                              |
|-------|------------------------------------------|
| ✨    | Introduce new features                   |
| 🐛    | Fix a bug                                |
| 🚑    | Critical hotfix                          |
| ⚡    | Improve performance                      |
| ♻️    | Refactor code                            |
| ✅    | Add, update, or pass tests               |
| 📝    | Add or update documentation              |
| 🔧    | Add or update configuration files        |
| 🔨    | Add or update development scripts        |
| ⬆️    | Upgrade dependencies                     |
| 👷    | Add or update CI build system            |
| 💚    | Fix CI build                             |
| 🚨    | Fix compiler / linter warnings           |
| 🎉    | Begin a project / initial commit         |
| 🔖    | Release / version tags                   |
| 🔒    | Fix security or privacy issues           |
| 🚚    | Move or rename resources                 |
| 🔥    | Remove code or files                     |
| 🩹    | Simple fix for a non-critical issue      |
| 🎨    | Improve structure / format of the code   |
| 💡    | Add or update comments in source code    |
| 🏷️    | Add or update types                      |
| 💥    | Introduce breaking changes               |

For the full list, see https://gitmoji.dev/.

### Examples

```
✨ add gradient clipping to optimizer
🐛 resolve nil pointer in tensor reshape
⚡ vectorize matrix multiplication kernel
✅ add coverage for edge cases in loss functions
📝 update API reference for v0.8
♻️ simplify computation graph traversal
👷 add CodeQL security scanning workflow
⬆️ upgrade codeql-action v3 → v4
💚 fix CI matrix cancellation on partial failure
```

Keep the subject line under 72 characters. Use the body of the commit message for additional context when necessary.

---

Thank you for contributing to Flux!
