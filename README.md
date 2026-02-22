# DataGen CLI

Deploy and manage AI agents from your GitHub repos. Connect a repo, discover agents, and ship them as live webhook endpoints — all from the terminal.

## Installation

### One-line (macOS/Linux)

```bash
curl -fsSL https://cli.datagen.dev/install.sh | sh
```

Mirror:

```bash
curl -fsSL https://raw.githubusercontent.com/datagendev/datagen-cli/main/install.sh | sh
```

Installs the latest release to `/usr/local/bin` if writable, otherwise to `~/.local/bin`.

**Options:**
- `DATAGEN_VERSION` — pin a specific release (e.g. `v0.2.0`)
- `DATAGEN_INSTALL_DIR` — custom install directory

```bash
curl -fsSL https://cli.datagen.dev/install.sh | env DATAGEN_VERSION=v0.2.0 sh
```

### Windows (PowerShell)

```powershell
irm https://cli.datagen.dev/install.ps1 | iex
```

Mirror:

```powershell
irm https://raw.githubusercontent.com/datagendev/datagen-cli/main/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\datagen` and adds it to your user PATH.

**Options:**
- `$env:DATAGEN_VERSION` — pin a specific release (e.g. `v0.3.1`)
- `$env:DATAGEN_INSTALL_DIR` — custom install directory

```powershell
$env:DATAGEN_VERSION="v0.3.1"; irm https://cli.datagen.dev/install.ps1 | iex
```

Or download manually from the [Releases page](https://github.com/datagendev/datagen-cli/releases) and place `datagen.exe` somewhere in your PATH.

### From Source

```bash
git clone https://github.com/datagendev/datagen-cli
cd datagen-cli
go build -o datagen
sudo mv datagen /usr/local/bin/
```

Verify:

```bash
datagen --help
```

## End-to-End Workflow

### 1. Login

```bash
datagen login
```

Saves your DataGen API key to your shell profile (`~/.zshrc`). Restart your terminal or `source ~/.zshrc` after running.

### 2. Configure MCP (Optional)

```bash
datagen mcp
```

Adds the DataGen MCP server to local tool configs (Claude Code, Codex, Gemini).

### 3. Connect GitHub

```bash
datagen github connect
```

Opens your browser to install the DataGen GitHub App. Once installed, DataGen scans your repos for agents defined in `.claude/agents/*.md`.

Check status and connected repos:

```bash
datagen github status
datagen github repos
datagen github connected
```

Connect a specific repo:

```bash
datagen github connect-repo owner/my-repo
```

Re-sync agents from a repo:

```bash
datagen github sync <repo-id>
```

### 4. Deploy an Agent

List discovered agents:

```bash
datagen agents list
datagen agents list --repo my-repo    # filter by repo
datagen agents list --deployed        # show only deployed agents
```

View agent details:

```bash
datagen agents show <agent-id>
```

Deploy — creates a webhook endpoint for the agent:

```bash
datagen agents deploy <agent-id>
```

### 5. Run and Monitor

Trigger an agent execution:

```bash
datagen agents run <agent-id>
datagen agents run <agent-id> --payload '{"key": "value"}'
```

View execution logs:

```bash
datagen agents logs <agent-id>
datagen agents logs <agent-id> --limit 20
```

### 6. Configure

View or update agent configuration:

```bash
# View current config
datagen agents config <agent-id>

# Set entry prompt
datagen agents config <agent-id> --set-prompt "You are a helpful assistant"

# Attach secrets and set PR mode
datagen agents config <agent-id> --secrets KEY1,KEY2 --pr-mode create_pr

# Add a recipient for notifications
datagen agents config <agent-id> --add-recipient user@example.com:OWNER

# Configure notification preferences
datagen agents config <agent-id> --notify-success true --notify-failure true
```

**Config flags:**

| Flag | Description |
|------|-------------|
| `--set-prompt` | Set the entry prompt text |
| `--clear-prompt` | Clear the entry prompt |
| `--secrets` | Comma-separated secret names for webhook |
| `--pr-mode` | `create_pr`, `auto_merge`, or `skip` |
| `--add-recipient` | Add recipient as `email[:role]` |
| `--remove-recipient` | Remove recipient by ID |
| `--notify-success` | Email on success: `true`, `false`, or `default` |
| `--notify-failure` | Email on failure: `true`, `false`, or `default` |
| `--notify-reply` | Email reply-to-resume: `true`, `false`, or `default` |

### 7. Schedule

Set up cron schedules for automated runs:

```bash
# List schedules
datagen agents schedule <agent-id>

# Create a daily 9am schedule (Eastern time)
datagen agents schedule <agent-id> --cron "0 9 * * *" --timezone "America/New_York" --name "daily-9am"

# Pause / resume / delete
datagen agents schedule <agent-id> --pause <schedule-id>
datagen agents schedule <agent-id> --resume <schedule-id>
datagen agents schedule <agent-id> --delete <schedule-id>
```

### 8. Manage Secrets

Store API keys and secrets that agents can access at runtime:

```bash
datagen secrets list
datagen secrets set OPENAI_API_KEY=sk-...
datagen secrets set MY_SECRET              # prompts for value
```

## Undeploy

Remove an agent's webhook endpoint:

```bash
datagen agents undeploy <agent-id>
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `datagen login` | Save your DataGen API key |
| `datagen mcp` | Configure DataGen MCP in local tools |
| `datagen github connect` | Install GitHub App and connect repos |
| `datagen github repos` | List available repositories |
| `datagen github connected` | List connected repositories |
| `datagen github connect-repo` | Connect a specific repository |
| `datagen github sync` | Re-sync agents from a repository |
| `datagen github status` | Check GitHub connection status |
| `datagen agents list` | List discovered agents |
| `datagen agents show` | Show agent details and recent executions |
| `datagen agents deploy` | Deploy an agent (creates webhook endpoint) |
| `datagen agents undeploy` | Remove an agent deployment |
| `datagen agents run` | Trigger agent execution |
| `datagen agents logs` | View execution history |
| `datagen agents config` | View or update agent configuration |
| `datagen agents schedule` | Manage cron schedules |
| `datagen secrets list` | List stored secrets (masked) |
| `datagen secrets set` | Create or update a secret |

## Development

```bash
go build -o datagen        # Build
go test ./...              # Test
make release               # Cross-compile for all platforms
```
