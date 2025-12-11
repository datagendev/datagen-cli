# DataGen CLI

A command-line tool for generating production-ready FastAPI boilerplate for deploying Claude Code agents with DataGen MCP integration.

## Features

- 🎯 **Interactive Setup** - Answer simple questions to configure your project
- 🔧 **Multiple Endpoint Types** - Support for webhooks, synchronous APIs, and streaming endpoints
- 🔐 **Built-in Auth** - API key, bearer token, and HMAC signature verification
- 📝 **Type-Safe** - Generates Pydantic models from your schema definitions
- 🚀 **Deploy Ready** - Railway deployment configuration included
- 🎨 **Flexible** - Customize auth, tools, timeouts, and more per endpoint

## Installation

### From Source

```bash
git clone https://github.com/datagendev/datagen-cli
cd datagen-cli
go build -o datagen
sudo mv datagen /usr/local/bin/
```

### Quick Test

```bash
./datagen --help
```

## Usage

### 1. Start a New Project

```bash
datagen start
```

This will interactively guide you through:
- Configuring API key environment variables
- Creating services/endpoints
- Choosing endpoint types (webhook, api, streaming)
- Defining input/output schemas
- Setting up authentication
- Selecting allowed DataGen tools

**Options:**
- `--output`, `-o` - Directory to save the configuration file (default: current directory)

**Example:**
```bash
# Save config to a specific directory
datagen start --output ./my-project
```

### 2. Build the Project

After completing the interactive setup, you'll have a `datagen.toml` file. Generate the boilerplate code with:

```bash
datagen build
```

**Options:**
- `--output`, `-o` - Directory for generated project files (default: current directory)
- `--config`, `-c` - Path to datagen.toml configuration file (default: datagen.toml)

**Examples:**
```bash
# Generate in current directory
datagen build

# Generate in a specific directory
datagen build --output ./my-project

# Use config from different location
datagen build --output ./output --config ./my-project/datagen.toml
```

**Note:** Using `--output` is recommended to avoid polluting your source directory during testing.

This creates:
```
.
├── app/
│   ├── __init__.py
│   ├── main.py           # FastAPI application with all endpoints
│   ├── agent.py          # Agent loading and execution
│   ├── config.py         # Configuration management
│   └── models.py         # Pydantic models
├── .claude/agents/       # Your agent prompt files
├── Dockerfile
├── requirements.txt
├── .env.example
├── Procfile
├── railway.json
└── README.md
```

### 3. Deploy to Railway

```bash
datagen deploy railway
```

**Options:**
- `--output`, `-o` - Directory containing the project to deploy (default: current directory)

**Example:**
```bash
# Deploy from current directory
datagen deploy railway

# Deploy from specific directory
datagen deploy railway --output ./my-project
```

## Endpoint Types

### Webhook (Async Background Processing)

Perfect for fire-and-forget operations:
- Accepts payload and queues background task
- Returns immediately with request ID
- Optional HMAC signature verification
- Retry policies with exponential backoff

**Example use case:** Receiving webhook from Stripe, HubSpot, or custom services

### API (Synchronous)

For operations that return results:
- Waits for agent to complete
- Returns structured response
- Configurable timeout
- Rate limiting support
- Define output schema for type safety

**Example use case:** Chat endpoints, enrichment APIs, data transformation

### Streaming (Server-Sent Events)

For real-time streaming responses:
- Server-Sent Events (SSE) support
- Streams agent output as it's generated
- Configurable buffer size
- JSON or text format options

**Example use case:** Chat interfaces, real-time generation, progressive updates

## Configuration Example

Here's what a `datagen.toml` might look like:

```toml
datagen_api_key_env = "DATAGEN_API_KEY"
claude_api_key_env = "ANTHROPIC_API_KEY"

[[service]]
name = "enrichment"
type = "webhook"
webhook_path = "/webhook/signup"
description = "Receives signup event and triggers enrichment"
prompt = ".claude/agents/enrichment.md"

[service.allowed_tools]
searchTools = false
executeTools = true
executeCode = false
getToolDetails = true

[[service.input_schema.fields]]
name = "email"
type = "str"
required = true

[service.webhook]
signature_verification = "hmac_sha256"
signature_header = "X-Signature"
secret_env = "HMAC_SECRET"
retry_enabled = true
max_retries = 3
backoff_strategy = "exponential"

[service.auth]
type = "api_key"
header = "X-API-Key"
env_var = "WEBHOOK_SECRET"

[[service]]
name = "chat"
type = "api"
api_path = "/api/chat"
description = "Synchronous chat endpoint"
prompt = ".claude/agents/chat.md"

[[service.input_schema.fields]]
name = "message"
type = "str"
required = true

[[service.output_schema.fields]]
name = "response"
type = "str"
required = true

[service.api]
response_format = "json"
timeout = 30
rate_limit_enabled = false

[service.auth]
type = "bearer_token"
env_var = "API_TOKEN"
```

## Commands

### `datagen start`

Interactive project setup. Creates `datagen.toml` configuration file.

**Options:**
- Supports multiple services in one project
- Each service can have different endpoint types
- Conditional prompts based on endpoint type

### `datagen build`

Generate FastAPI boilerplate from `datagen.toml`.

**Generated files:**
- FastAPI application with type-safe endpoints
- Agent loading and execution logic
- Configuration management with Pydantic
- Dockerfile and deployment configs
- Comprehensive README

### `datagen deploy [platform]`

Deploy to cloud platform. Currently supports:
- `railway` - Deploy to Railway

## Development

### Prerequisites

- Go 1.20+
- Python 3.13+ (for running generated projects)

### Build from Source

```bash
git clone https://github.com/datagendev/datagen-cli
cd datagen-cli
go mod download
go build -o datagen
```

### Run Tests

```bash
go test ./...
```

## Generated Project Usage

After running `datagen build`, your generated project is ready to use:

```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Set up environment variables
cp .env.example .env
# Edit .env with your actual API keys

# Run locally
uvicorn app.main:app --reload

# Visit API docs
open http://localhost:8000/docs
```

## Authentication Types

### API Key

```toml
[service.auth]
type = "api_key"
header = "X-API-Key"
env_var = "MY_API_KEY"
```

### Bearer Token

```toml
[service.auth]
type = "bearer_token"
env_var = "MY_TOKEN"
```

### HMAC Signature (Webhooks)

```toml
[service.webhook]
signature_verification = "hmac_sha256"
signature_header = "X-Signature"
secret_env = "HMAC_SECRET"
```

## Schema Types

Supported field types for input and output schemas:

- `str` - String
- `int` - Integer
- `float` - Float/Decimal
- `bool` - Boolean
- `list` - List/Array
- `dict` - Dictionary/Object
- `any` - Any type

## Examples

See the reference implementation at [my-agent-project](../my-agent-project) for a complete example.

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License

## Support

- **Issues**: Report bugs via GitHub Issues
- **Documentation**: See generated README.md in projects
- **Reference**: Check [my-agent-project](../my-agent-project) for working example

## Credits

Built for deploying Claude Code agents with:
- [FastAPI](https://fastapi.tiangolo.com/)
- [Claude Agent SDK](https://github.com/anthropics/anthropic-sdk-python)
- [DataGen MCP](https://datagen.dev/)
- [Cobra](https://cobra.dev/) - CLI framework
- [Survey](https://github.com/AlecAivazis/survey) - Interactive prompts
