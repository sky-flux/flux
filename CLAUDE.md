# flux

Pure Go implementation of the FSRS v6 spaced repetition scheduling algorithm.

## Project Structure

```
├── *.go                  # Core package: Card, Scheduler, Rating, State, Algorithm
├── optimizer/            # Subpackage: parameter training and optimal retention
├── examples/             # Runnable demos (basic, optimizer, reschedule)
├── testdata/             # py-fsrs alignment and optimizer baseline data
├── scripts/              # Python scripts for generating test data
└── .github/workflows/    # CI, Release, CodeQL
```

## Build & Test

```bash
go test ./...                                    # Unit tests
go test -cover ./... -coverprofile=coverage.out   # With coverage
go test -tags integration ./optimizer/            # Integration tests
go test -bench=. -benchmem -run='^$' ./...        # Benchmarks
go vet ./...                                      # Vet
```

## Key Conventions

- **Zero external dependencies** — stdlib only. Do not add third-party modules.
- **100% test coverage** enforced in CI for `.` and `./optimizer/` packages.
- **Tests first** — write tests before implementation.
- **DisableFuzzing: true** in tests for deterministic, reproducible results.
- All outputs cross-validated against py-fsrs reference implementation.

## Commit Convention

Format: `<gitmoji> <description>`

Uses [gitmoji](https://gitmoji.dev/) convention. Full reference at https://gitmoji.dev/. Common ones for this project:

| Emoji | When to Use                              |
|-------|------------------------------------------|
| ✨    | Introduce new features                   |
| 🐛    | Fix a bug                                |
| ⚡    | Improve performance                      |
| ♻️    | Refactor code                            |
| ✅    | Add, update, or pass tests               |
| 📝    | Add or update documentation              |
| 🔧    | Add or update configuration files        |
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

Rules:
- Every commit must start with a gitmoji emoji.
- Keep subject line under 72 characters.
- **No AI attribution** — never include `Co-Authored-By` or any AI-related metadata in commits.

## Code Style

- Run `gofmt -w .` before committing.
- Run `go vet ./...` to catch common mistakes.
- golangci-lint config in `.golangci.yml`.
- No TODO/FIXME comments in committed code.
