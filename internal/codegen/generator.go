package codegen

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/datagendev/datagen-cli/internal/config"
)

//go:embed templates/*
var templatesFS embed.FS

// Template helper functions
var templateFuncs = template.FuncMap{
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
	"replace": func(old, new, s string) string {
		return strings.ReplaceAll(s, old, new)
	},
}

// GenerateProject creates the full project structure
func GenerateProject(cfg *config.DatagenConfig, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{
		filepath.Join(outputDir, "app"),
		filepath.Join(outputDir, ".claude/agents"),
		filepath.Join(outputDir, "scripts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate files
	if err := generateMainPy(cfg, outputDir); err != nil {
		return fmt.Errorf("failed to generate main.py: %w", err)
	}

	if err := generateAgentPy(cfg, outputDir); err != nil {
		return fmt.Errorf("failed to generate agent.py: %w", err)
	}

	if err := generateConfigPy(cfg, outputDir); err != nil {
		return fmt.Errorf("failed to generate config.py: %w", err)
	}

	if err := generateModelsPy(cfg, outputDir); err != nil {
		return fmt.Errorf("failed to generate models.py: %w", err)
	}

	if err := generateInitPy(outputDir); err != nil {
		return fmt.Errorf("failed to generate __init__.py: %w", err)
	}

	if err := generateRequirementsTxt(outputDir); err != nil {
		return fmt.Errorf("failed to generate requirements.txt: %w", err)
	}

	if err := generateDockerfile(outputDir); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	if err := generateEnvExample(cfg, outputDir); err != nil {
		return fmt.Errorf("failed to generate .env.example: %w", err)
	}

	if err := generateProcfile(outputDir); err != nil {
		return fmt.Errorf("failed to generate Procfile: %w", err)
	}

	if err := generateRailwayJSON(outputDir); err != nil {
		return fmt.Errorf("failed to generate railway.json: %w", err)
	}

	if err := generateREADME(cfg, outputDir); err != nil {
		return fmt.Errorf("failed to generate README.md: %w", err)
	}

	return nil
}

func generateMainPy(cfg *config.DatagenConfig, outputDir string) error {
	tmpl, err := template.New("main.py.tmpl").Funcs(templateFuncs).ParseFS(templatesFS, "templates/main.py.tmpl")
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outputDir, "app/main.py"))
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, cfg)
}

func generateAgentPy(cfg *config.DatagenConfig, outputDir string) error {
	// Using raw string literal with proper escape for Python f-strings
	content := "\"\"\"Agent loading and execution logic.\"\"\"\n\n" +
		"import json\n" +
		"import logging\n" +
		"from dataclasses import dataclass\n" +
		"from pathlib import Path\n" +
		"from typing import Any, Dict, Optional\n\n" +
		"import frontmatter\n" +
		"from claude_agent_sdk import (\n" +
		"    AssistantMessage,\n" +
		"    ClaudeAgentOptions,\n" +
		"    TextBlock,\n" +
		"    ToolUseBlock,\n" +
		"    query,\n" +
		")\n\n" +
		"from app.config import settings\n\n" +
		"logger = logging.getLogger(__name__)\n\n\n" +
		"def log_event(event: str, **data):\n" +
		"    \"\"\"Emit structured JSON log for easy parsing.\"\"\"\n" +
		"    payload = {\"event\": event, **data}\n" +
		"    logger.info(json.dumps(payload, indent=2, ensure_ascii=False))\n\n\n" +
		"@dataclass\n" +
		"class AgentConfig:\n" +
		"    \"\"\"Configuration loaded from agent.md file.\"\"\"\n\n" +
		"    name: str\n" +
		"    model: str\n" +
		"    system_prompt: str\n" +
		"    allowed_tools: list[str]\n" +
		"    description: Optional[str] = None\n\n" +
		"    @classmethod\n" +
		"    def from_file(cls, path: Path) -> \"AgentConfig\":\n" +
		"        \"\"\"Load agent configuration from markdown file.\"\"\"\n" +
		"        if not path.exists():\n" +
		"            raise FileNotFoundError(f\"Agent file not found: {path}\")\n\n" +
		"        content = path.read_text(encoding=\"utf-8\")\n\n" +
		"        try:\n" +
		"            post = frontmatter.loads(content)\n" +
		"            has_frontmatter = bool(post.metadata)\n" +
		"        except Exception:\n" +
		"            has_frontmatter = False\n" +
		"            post = None\n\n" +
		"        if has_frontmatter and post:\n" +
		"            name = post.metadata.get(\"name\", path.stem)\n" +
		"            model = post.metadata.get(\"model\", \"claude-sonnet-4-5\")\n" +
		"            description = post.metadata.get(\"description\")\n\n" +
		"            tools = post.metadata.get(\"tools\", [])\n" +
		"            if isinstance(tools, str):\n" +
		"                allowed_tools = [t.strip() for t in tools.split(\",\") if t.strip()]\n" +
		"            else:\n" +
		"                allowed_tools = tools if isinstance(tools, list) else []\n\n" +
		"            system_prompt = post.content.strip()\n" +
		"        else:\n" +
		"            name = path.stem\n" +
		"            model = \"claude-sonnet-4-5\"\n" +
		"            description = None\n" +
		"            allowed_tools = [\n" +
		"                \"mcp__Datagen__getToolDetails\",\n" +
		"                \"mcp__Datagen__executeTool\",\n" +
		"            ]\n" +
		"            system_prompt = content.strip()\n\n" +
		"        return cls(\n" +
		"            name=name,\n" +
		"            model=model,\n" +
		"            system_prompt=system_prompt,\n" +
		"            allowed_tools=allowed_tools,\n" +
		"            description=description,\n" +
		"        )\n\n\n" +
		"class AgentExecutor:\n" +
		"    \"\"\"Execute Claude agent with MCP integration.\"\"\"\n\n" +
		"    def __init__(self, agent_config: AgentConfig):\n" +
		"        \"\"\"Initialize executor with agent configuration.\"\"\"\n" +
		"        self.config = agent_config\n" +
		"        self.model = settings.model_name or agent_config.model\n\n" +
		"    def build_mcp_config(self) -> Dict[str, Any]:\n" +
		"        \"\"\"Build MCP server configuration from environment.\"\"\"\n" +
		"        mcp_servers = {}\n\n" +
		"        if settings.datagen_api_key:\n" +
		"            mcp_servers[\"datagen\"] = {\n" +
		"                \"type\": \"http\",\n" +
		"                \"url\": \"https://mcp.datagen.dev/mcp\",\n" +
		"                \"headers\": {\"Authorization\": f\"Bearer {settings.datagen_api_key.strip()}\"},\n" +
		"            }\n" +
		"            log_event(\n" +
		"                \"mcp_config\",\n" +
		"                server=\"datagen\",\n" +
		"                url=\"https://mcp.datagen.dev/mcp\",\n" +
		"                authenticated=True,\n" +
		"            )\n\n" +
		"        return mcp_servers\n\n" +
		"    def _build_options(self) -> ClaudeAgentOptions:\n" +
		"        \"\"\"Compose Claude agent options.\"\"\"\n" +
		"        return ClaudeAgentOptions(\n" +
		"            model=self.model,\n" +
		"            system_prompt=self.config.system_prompt,\n" +
		"            permission_mode=settings.permission_mode,\n" +
		"            mcp_servers=self.build_mcp_config(),\n" +
		"            allowed_tools=self.config.allowed_tools if self.config.allowed_tools else None,\n" +
		"        )\n\n" +
		"    async def stream_execute(self, payload: Dict[str, Any], request_id: str, *, log_success: bool = True):\n" +
		"        \"\"\"Async generator yielding text chunks for streaming responses.\"\"\"\n" +
		"        log_event(\"agent_start\", request_id=request_id, agent=self.config.name)\n" +
		"        user_message = self._format_payload(payload)\n" +
		"        opts = self._build_options()\n\n" +
		"        try:\n" +
		"            async for msg in query(prompt=user_message, options=opts):\n" +
		"                if isinstance(msg, AssistantMessage):\n" +
		"                    for block in msg.content:\n" +
		"                        if isinstance(block, TextBlock):\n" +
		"                            text = block.text\n" +
		"                            log_event(\n" +
		"                                \"agent_chunk\",\n" +
		"                                request_id=request_id,\n" +
		"                                chunk=text[:500],\n" +
		"                                truncated=len(text) > 500,\n" +
		"                            )\n" +
		"                            yield text\n" +
		"                        elif isinstance(block, ToolUseBlock):\n" +
		"                            log_event(\n" +
		"                                \"agent_tool_use\",\n" +
		"                                request_id=request_id,\n" +
		"                                tool=block.name,\n" +
		"                                input=block.input,\n" +
		"                            )\n" +
		"                else:\n" +
		"                    log_event(\"agent_event\", request_id=request_id, msg_type=type(msg).__name__)\n\n" +
		"        except Exception as e:\n" +
		"            log_event(\n" +
		"                \"agent_error\",\n" +
		"                request_id=request_id,\n" +
		"                error=str(e),\n" +
		"                error_type=type(e).__name__,\n" +
		"            )\n" +
		"            raise\n" +
		"        finally:\n" +
		"            if log_success:\n" +
		"                log_event(\"agent_success\", request_id=request_id, result_length=None)\n\n" +
		"    async def execute(self, payload: Dict[str, Any], request_id: str) -> str:\n" +
		"        \"\"\"Execute agent and return concatenated text (non-streaming).\"\"\"\n" +
		"        collected_text: list[str] = []\n" +
		"        async for chunk in self.stream_execute(payload, request_id, log_success=False):\n" +
		"            collected_text.append(chunk)\n\n" +
		"        result = \"\".join(collected_text)\n" +
		"        log_event(\"agent_success\", request_id=request_id, result_length=len(result))\n" +
		"        return result\n\n" +
		"    def _format_payload(self, payload: Dict[str, Any]) -> str:\n" +
		"        \"\"\"Format payload as JSON for the agent.\"\"\"\n" +
		"        return f\"\"\"Here is the input data to process:\n\n" +
		"```json\n" +
		"{json.dumps(payload, indent=2, ensure_ascii=False)}\n" +
		"```\n\n" +
		"Process this data according to your system prompt instructions.\"\"\"\n\n\n" +
		"# Agent executors will be loaded per service\n" +
		"agent_executors = {}\n\n\n" +
		"def load_agent(name: str, prompt_path: str) -> AgentExecutor:\n" +
		"    \"\"\"Load an agent from a prompt file.\"\"\"\n" +
		"    from pathlib import Path\n" +
		"    base_dir = Path(__file__).resolve().parent.parent\n" +
		"    agent_file = base_dir / prompt_path\n" +
		"    agent_config = AgentConfig.from_file(agent_file)\n" +
		"    executor = AgentExecutor(agent_config)\n" +
		"    log_event(\"agent_loaded\", name=name, model=executor.model, file=str(agent_file))\n" +
		"    return executor\n"

	return os.WriteFile(filepath.Join(outputDir, "app/agent.py"), []byte(content), 0644)
}

func generateConfigPy(cfg *config.DatagenConfig, outputDir string) error {
	tmpl, err := template.New("config.py.tmpl").Funcs(templateFuncs).ParseFS(templatesFS, "templates/config.py.tmpl")
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outputDir, "app/config.py"))
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, cfg)
}

func generateModelsPy(cfg *config.DatagenConfig, outputDir string) error {
	tmpl, err := template.New("models.py.tmpl").Funcs(templateFuncs).ParseFS(templatesFS, "templates/models.py.tmpl")
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outputDir, "app/models.py"))
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, cfg)
}

func generateInitPy(outputDir string) error {
	content := `"""FastAPI application package."""
`
	return os.WriteFile(filepath.Join(outputDir, "app/__init__.py"), []byte(content), 0644)
}

func generateRequirementsTxt(outputDir string) error {
	content := `# FastAPI and server
fastapi~=0.115.0
uvicorn[standard]~=0.32.0

# Anthropic and agent SDK
anthropic~=0.39.0
claude-agent-sdk~=0.1.0

# DataGen SDK
datagen-python-sdk~=0.1.0

# HTTP client
httpx~=0.27.0

# Data validation
pydantic~=2.10.0
pydantic-settings~=2.6.0

# Markdown parsing
python-frontmatter~=1.1.0
pyyaml~=6.0.2
`
	return os.WriteFile(filepath.Join(outputDir, "requirements.txt"), []byte(content), 0644)
}

func generateDockerfile(outputDir string) error {
	content := `# Use Python 3.13 slim image
FROM python:3.13-slim

# Create a non-root user with home directory
RUN groupadd -r appuser && useradd -r -g appuser -m -d /home/appuser appuser

# Set working directory
WORKDIR /app

# Ensure appuser can write to home directory
RUN mkdir -p /home/appuser && chown -R appuser:appuser /home/appuser

# Copy requirements first for better caching
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port (Railway will set PORT env var)
EXPOSE 8000

# Start the application using PORT environment variable
CMD uvicorn app.main:app --host 0.0.0.0 --port ${PORT:-8000}
`
	return os.WriteFile(filepath.Join(outputDir, "Dockerfile"), []byte(content), 0644)
}

func generateEnvExample(cfg *config.DatagenConfig, outputDir string) error {
	content := fmt.Sprintf(`# Required
%s=your-anthropic-api-key-here
%s=your-datagen-api-key-here

# Optional
MODEL_NAME=claude-sonnet-4-5
LOG_LEVEL=INFO
PORT=8000
PERMISSION_MODE=bypassPermissions
`, cfg.ClaudeAPIKeyEnv, cfg.DatagenAPIKeyEnv)

	// Add service-specific env vars
	for _, svc := range cfg.Services {
		if svc.Auth != nil && svc.Auth.EnvVar != "" {
			content += fmt.Sprintf("\n# Auth for %s service\n%s=your-secret-here\n", svc.Name, svc.Auth.EnvVar)
		}
		if svc.Webhook != nil && svc.Webhook.SecretEnv != "" {
			content += fmt.Sprintf("%s=your-hmac-secret-here\n", svc.Webhook.SecretEnv)
		}
	}

	return os.WriteFile(filepath.Join(outputDir, ".env.example"), []byte(content), 0644)
}

func generateProcfile(outputDir string) error {
	content := `web: uvicorn app.main:app --host 0.0.0.0 --port $PORT
`
	return os.WriteFile(filepath.Join(outputDir, "Procfile"), []byte(content), 0644)
}

func generateRailwayJSON(outputDir string) error {
	content := `{
  "$schema": "https://railway.com/railway.schema.json",
  "build": {
    "builder": "DOCKERFILE",
    "dockerfilePath": "Dockerfile"
  },
  "deploy": {
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 10
  }
}
`
	return os.WriteFile(filepath.Join(outputDir, "railway.json"), []byte(content), 0644)
}

func generateREADME(cfg *config.DatagenConfig, outputDir string) error {
	content := "# DataGen Agent Project\n\n"
	content += "Generated by DataGen CLI\n\n"
	content += "## Services\n\n"

	for _, svc := range cfg.Services {
		content += fmt.Sprintf("### %s (%s)\n", svc.Name, svc.Type)
		content += fmt.Sprintf("- **Path**: %s\n", svc.GetPath())
		content += fmt.Sprintf("- **Description**: %s\n", svc.Description)
		content += fmt.Sprintf("- **Prompt**: %s\n\n", svc.Prompt)
	}

	content += "## Quick Start\n\n"
	content += "1. Create a virtual environment:\n"
	content += "   ```bash\n"
	content += "   python -m venv venv\n"
	content += "   source venv/bin/activate\n"
	content += "   ```\n\n"
	content += "2. Install dependencies:\n"
	content += "   ```bash\n"
	content += "   pip install -r requirements.txt\n"
	content += "   ```\n\n"
	content += "3. Set up environment variables:\n"
	content += "   ```bash\n"
	content += "   cp .env.example .env\n"
	content += "   # Edit .env with your API keys\n"
	content += "   ```\n\n"
	content += "4. Run locally:\n"
	content += "   ```bash\n"
	content += "   uvicorn app.main:app --reload\n"
	content += "   ```\n\n"
	content += "5. Deploy to Railway:\n"
	content += "   ```bash\n"
	content += "   datagen deploy railway\n"
	content += "   ```\n\n"
	content += "## API Documentation\n\n"
	content += "Once running, visit http://localhost:8000/docs for interactive API documentation.\n"

	return os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(content), 0644)
}
