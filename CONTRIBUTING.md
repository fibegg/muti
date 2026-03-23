# Contributing to muti

Thanks for your interest in contributing!

## Development Setup

1. **Clone the repo:**
   ```bash
   git clone https://github.com/fibegg/muti.git
   cd muti
   ```

2. **Build:**
   ```bash
   make build
   ```

3. **Run tests:**
   ```bash
   make test
   ```

4. **Lint (requires [golangci-lint](https://golangci-lint.run/usage/install/)):**
   ```bash
   make lint
   ```

## Adding a New Language

1. Add the tree-sitter grammar dependency:
   ```bash
   go get github.com/tree-sitter/tree-sitter-<lang>@latest
   ```

2. Add an entry to [`internal/language/registry.go`](internal/language/registry.go)

3. The existing operators work automatically — tree-sitter node types are universal enough.

## Adding a New Operator

1. Create a new file in [`internal/mutation/`](internal/mutation/)
2. Implement the `Operator` interface (`Name()` and `Apply()`)
3. Register it in the `All()` function in [`operator.go`](internal/mutation/operator.go)

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make changes and add tests
4. Run `make all` (lint + test + build)
5. Commit with a descriptive message
6. Push and open a pull request

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation
- `test:` test changes
- `ci:` CI/CD changes
- `chore:` maintenance
