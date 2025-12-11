# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**datagen-cli** is a Go-based CLI tool that generates production-ready FastAPI boilerplate for deploying Claude Code agents with DataGen MCP integration. It uses Cobra for CLI commands, Survey for interactive prompts, and Go templates for code generation.

## Build and Development Commands

```bash
# Build the binary
go build -o datagen
# or
make build

# Run tests
go test ./...
# or
make test

# Install locally for testing
sudo mv datagen /usr/local/bin/
# or
make install

# Build for multiple platforms
make release

# Quick test during development
./datagen --help
./datagen start --output ./test-project
./datagen build --output ./output --config ./test-project/datagen.toml
./datagen deploy railway --output ./output
```

## Architecture

### Three-Phase Workflow

1. **Interactive Setup** (`datagen start`): Prompts user with Survey library, creates `datagen.toml` config
2. **Code Generation** (`datagen build`): Parses TOML config, validates, generates FastAPI project from templates
3. **Deployment** (`datagen deploy`): Deploys generated project to Railway (or other platforms)

### Key Components

#### Command Layer (`cmd/`)
- **root.go**: Cobra root command registration
- **start.go**: Interactive setup flow using Survey prompts
- **build.go**: Config loading and project generation orchestration
- **deploy.go**: Deployment logic (Railway integration)

#### Configuration Layer (`internal/config/`)
- **types.go**: Core data structures for `datagen.toml` configuration
  - `DatagenConfig`: Root config with services array
  - `Service`: Individual endpoint configuration (webhook/api/streaming)
  - `Schema`: Input/output field definitions
  - Type-specific configs: `WebhookConfig`, `APIConfig`, `StreamingConfig`
- **parser.go**: TOML parsing using BurntSushi/toml
  - `LoadConfig()`: Reads TOML, passes configDir to validator for relative path resolution
  - `SaveConfig()`: Writes config back to TOML
- **validator.go**: Configuration validation
  - Validates prompt file paths **relative to config directory** (not CWD)
  - Validates required fields, types, and endpoint-specific configs
  - Empty input schemas are valid (services without input parameters)

#### Code Generation Layer (`internal/codegen/`)
- **generator.go**: Main code generation logic
  - Uses `//go:embed templates/*` for embedded templates
  - `GenerateProject()`: Orchestrates full project generation with outputDir parameter
  - Template functions: `lower`, `upper`, `replace(old, new, s)` - note parameter order for pipe syntax
  - All file paths use `filepath.Join(outputDir, ...)` to avoid source directory pollution
- **templates/**: Go text/template files for FastAPI code
  - `main.py.tmpl`: FastAPI app with all endpoints (single template handles all three types)
  - `models.py.tmpl`: Pydantic models from schemas
  - `config.py.tmpl`: Environment variable configuration
  - Uses conditionals: `{{if eq .Type "webhook"}}...{{else if eq .Type "api"}}...{{end}}`

#### Interactive Prompts (`internal/prompts/`)
- **interactive.go**: Survey-based interactive questions
  - Conditional prompts based on endpoint type selection
  - Schema field collection with type validation
  - Auth and tool configuration prompts

### Important Implementation Details

#### Path Resolution
- **Prompt files**: Resolved relative to config directory, not CWD
  - `LoadConfig()` passes `filepath.Dir(configPath)` to validator
  - `validateService()` uses `filepath.Join(configDir, promptPath)` for relative paths
- **Output files**: All use `filepath.Join(outputDir, ...)` to support `--output` flag
- **Config file**: `--config` flag allows specifying config path outside CWD

#### Template Function: `replace`
Custom function with signature `func(old, new, s string)` to work with Go template pipe syntax:
```go
// Template usage: {{.Auth.Header | lower | replace "-" "_"}}
// Converts "X-API-Key" -> "x-api-key" -> "x_api_key"
"replace": func(old, new, s string) string {
    return strings.ReplaceAll(s, old, new)
}
```
Parameter order matters: piped value comes last, so `replace` takes `(old, new, s)` not `(s, old, new)`.

#### Endpoint Types
Three distinct types with different configurations:
- **webhook**: Async background processing, HMAC verification, retry policies
- **api**: Synchronous calls, output schemas, timeouts, rate limiting
- **streaming**: SSE streaming, buffer configuration

Each type has:
- Dedicated config struct (`WebhookConfig`, `APIConfig`, `StreamingConfig`)
- Conditional validation in `validator.go`
- Conditional template sections in `main.py.tmpl`
- Type-specific path field (`WebhookPath` vs `APIPath`)

## Code Generation Testing

To avoid polluting the source directory during testing:

```bash
# Create test config in separate directory
mkdir test-project
./datagen start --output ./test-project

# Generate output in separate directory
./datagen build --output ./test-output --config ./test-project/datagen.toml

# Deploy from separate directory
./datagen deploy railway --output ./test-output

# Source directory stays clean - no app/, .claude/, etc.
```

### Command Flags

**`datagen start`**
- `--output`, `-o` - Directory to save datagen.toml (default: current directory)

**`datagen build`**
- `--output`, `-o` - Directory for generated files (default: current directory)
- `--config`, `-c` - Path to datagen.toml (default: datagen.toml)

**`datagen deploy [platform]`**
- `--output`, `-o` - Directory containing project to deploy (default: current directory)

## Common Modifications

### Adding New Template Functions
Add to `templateFuncs` in `internal/codegen/generator.go`:
```go
var templateFuncs = template.FuncMap{
    "lower": strings.ToLower,
    "yourFunc": func(params...) returnType {
        // implementation
    },
}
```

### Adding New Endpoint Types
1. Add type to `Service.Type` validation in `validator.go`
2. Add type-specific config struct to `types.go`
3. Add conditional validation in `validateService()`
4. Add template conditionals in `templates/main.py.tmpl`
5. Update prompts in `internal/prompts/interactive.go`

### Modifying Configuration Schema
1. Update structs in `internal/config/types.go`
2. Add validation in `internal/config/validator.go`
3. Update prompts in `internal/prompts/interactive.go`
4. Update templates in `internal/codegen/templates/`

## Generated Project Structure

The `datagen build` command generates this structure:
```
output-dir/
├── app/
│   ├── __init__.py
│   ├── main.py          # FastAPI app with all endpoints
│   ├── agent.py         # Claude Agent SDK integration
│   ├── config.py        # Env var configuration
│   └── models.py        # Pydantic models
├── .claude/agents/      # Agent prompt markdown files
├── Dockerfile
├── requirements.txt
├── .env.example
├── Procfile             # Railway deployment
├── railway.json
└── README.md
```

## Field Type Validation

Supported schema field types (validated in `validator.go`):
- `str`, `int`, `float`, `bool`, `list`, `dict`, `any`

These map to Pydantic types in generated `models.py`.
