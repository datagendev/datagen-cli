# Repository Guidelines

## Project Structure & Module Organization

- `main.go` is the CLI entrypoint.
- `cmd/` contains Cobra subcommands (`start`, `build`, `add`, `deploy`) and flag wiring.
- `internal/config/` defines `datagen.toml` types, parsing, and validation.
- `internal/codegen/` holds embedded templates plus project generation and incremental injection.
- `internal/prompts/` contains Survey-driven interactive questions.
- Root `datagen.toml` is a sample config for local smoke tests. For deeper architecture details, see `CLAUDE.md`.

## Build, Test, and Development Commands

- `make build` / `go build -o datagen`: build the local binary.
- `make install`: installs to `/usr/local/bin` (requires sudo).
- `make test` / `go test ./...`: run all tests.
- `make dev`: build then print CLI help.
- `make release`: cross-compile platform binaries.

Quick manual flow:

```bash
./datagen start --output ./test-project
./datagen build --output ./test-output --config ./test-project/datagen.toml
./datagen deploy railway --output ./test-output
```

## Coding Style & Naming Conventions

- Run `gofmt` on all Go files; use idiomatic Go naming and keep packages small.
- Put new CLI wiring in `cmd/` and core logic in `internal/`. Export identifiers only when needed across packages.
- Template files live in `internal/codegen/templates/` and use the `.tmpl` suffix. Preserve marker comments in generated templates (they power `datagen add` incremental updates).

## Testing Guidelines

- Tests use Go’s standard `testing` package. Name files `*_test.go` and place them next to the package under test.
- Prefer table-driven tests for config validation and codegen helpers. Add minimal fixtures in-package.

## Commit & Pull Request Guidelines

- Use conventional commits consistent with history: `feat:`, `fix:`, `docs:`, `chore:` + short imperative subject (e.g., `feat: Add new streaming option`).
- PRs should describe user-facing behavior, note impacted commands/flags, link related issues if any, and include a brief local verification note (for example, “ran `make test` and built a sample project”).
