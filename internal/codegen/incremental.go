package codegen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/datagendev/datagen-cli/internal/config"
)

// IncrementalAddService adds a new service to existing project files
func IncrementalAddService(cfg *config.DatagenConfig, newService *config.Service, outputDir string) error {
	// Update main.py with new endpoint
	if err := updateMainPy(cfg, newService, outputDir); err != nil {
		return fmt.Errorf("failed to update main.py: %w", err)
	}

	// Update models.py with new models
	if err := updateModelsPy(newService, outputDir); err != nil {
		return fmt.Errorf("failed to update models.py: %w", err)
	}

	// Update .env.example if service has auth
	if newService.Auth != nil || (newService.Webhook != nil && newService.Webhook.SecretEnv != "") {
		if err := updateEnvExample(newService, outputDir); err != nil {
			return fmt.Errorf("failed to update .env.example: %w", err)
		}
	}

	return nil
}

// updateMainPy injects new endpoint handlers into main.py
func updateMainPy(cfg *config.DatagenConfig, newService *config.Service, outputDir string) error {
	mainPath := filepath.Join(outputDir, "app/main.py")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("failed to read main.py: %w", err)
	}

	mainContent := string(content)

	// Check for markers
	if !strings.Contains(mainContent, "=== AGENT LOADING START ===") {
		return fmt.Errorf("missing agent loading markers in main.py - file may have been manually modified")
	}
	if !strings.Contains(mainContent, "=== ENDPOINT HANDLERS START ===") {
		return fmt.Errorf("missing endpoint handlers markers in main.py - file may have been manually modified")
	}

	// 1. Add agent loading
	agentLoadingCode := fmt.Sprintf(`    agent_executors["%s"] = load_agent("%s", "%s")`,
		newService.Name, newService.Name, newService.Prompt)
	mainContent = injectBeforeMarker(mainContent, "    # === AGENT LOADING END ===", agentLoadingCode+"\n")

	// 2. Generate endpoint handler code
	endpointCode, err := generateEndpointCode(newService)
	if err != nil {
		return fmt.Errorf("failed to generate endpoint code: %w", err)
	}

	// 3. Inject endpoint handler before END marker
	mainContent = injectBeforeMarker(mainContent, "# === ENDPOINT HANDLERS END ===", endpointCode+"\n")

	// 4. Update health check services list
	mainContent = updateHealthCheckServices(mainContent, cfg)

	// Write back
	return os.WriteFile(mainPath, []byte(mainContent), 0644)
}

// updateModelsPy appends new models to models.py
func updateModelsPy(newService *config.Service, outputDir string) error {
	modelsPath := filepath.Join(outputDir, "app/models.py")
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		return fmt.Errorf("failed to read models.py: %w", err)
	}

	modelsContent := string(content)

	// Check for marker
	if !strings.Contains(modelsContent, "=== SERVICE MODELS START ===") {
		return fmt.Errorf("missing service models markers in models.py - file may have been manually modified")
	}

	// Generate model code
	modelCode, err := generateModelCode(newService)
	if err != nil {
		return fmt.Errorf("failed to generate model code: %w", err)
	}

	// Inject before END marker
	modelsContent = injectBeforeMarker(modelsContent, "# === SERVICE MODELS END ===", modelCode+"\n")

	// Write back
	return os.WriteFile(modelsPath, []byte(modelsContent), 0644)
}

// updateEnvExample appends new environment variables to .env.example
func updateEnvExample(newService *config.Service, outputDir string) error {
	envPath := filepath.Join(outputDir, ".env.example")
	content, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("failed to read .env.example: %w", err)
	}

	envContent := string(content)
	newVars := []string{}

	// Add auth env vars
	if newService.Auth != nil && newService.Auth.EnvVar != "" {
		varName := strings.ToUpper(newService.Auth.EnvVar)
		if !strings.Contains(envContent, varName+"=") {
			newVars = append(newVars, fmt.Sprintf("\n# %s authentication", newService.Name))
			newVars = append(newVars, fmt.Sprintf("%s=your_%s_key_here", varName, strings.ToLower(newService.Name)))
		}
	}

	// Add webhook secret env vars
	if newService.Webhook != nil && newService.Webhook.SecretEnv != "" {
		varName := strings.ToUpper(newService.Webhook.SecretEnv)
		if !strings.Contains(envContent, varName+"=") {
			newVars = append(newVars, fmt.Sprintf("\n# %s webhook signature verification", newService.Name))
			newVars = append(newVars, fmt.Sprintf("%s=your_webhook_secret_here", varName))
		}
	}

	if len(newVars) > 0 {
		envContent += "\n" + strings.Join(newVars, "\n") + "\n"
		return os.WriteFile(envPath, []byte(envContent), 0644)
	}

	return nil
}

// injectBeforeMarker inserts code before a marker line
func injectBeforeMarker(content, marker, codeToInject string) string {
	markerIndex := strings.Index(content, marker)
	if markerIndex == -1 {
		return content
	}

	before := content[:markerIndex]
	after := content[markerIndex:]
	return before + codeToInject + after
}

// updateHealthCheckServices updates the services list in health check endpoint
func updateHealthCheckServices(content string, cfg *config.DatagenConfig) string {
	// Find health check section
	healthStart := strings.Index(content, `"services": [`)
	if healthStart == -1 {
		return content
	}

	healthEnd := strings.Index(content[healthStart:], `],`)
	if healthEnd == -1 {
		return content
	}

	// Generate new services list
	var serviceNames []string
	for _, svc := range cfg.Services {
		serviceNames = append(serviceNames, fmt.Sprintf(`"%s"`, svc.Name))
	}
	newServicesList := strings.Join(serviceNames, ", ")

	// Replace
	before := content[:healthStart]
	after := content[healthStart+healthEnd+2:] // +2 to skip past the "],"
	return before + `"services": [` + newServicesList + `],` + after
}

// generateEndpointCode generates the endpoint handler code for a single service
func generateEndpointCode(svc *config.Service) (string, error) {
	// Create a mini-template with just the endpoint handler
	tmplStr := `
{{if eq .Type "webhook"}}
# Webhook endpoint: {{.Name}}
{{if .Auth}}
async def verify_{{.Name}}_auth({{if eq .Auth.Type "api_key"}}{{.Auth.Header | lower | replace "-" "_"}}: str | None = Header(None, alias="{{.Auth.Header}}"){{else if eq .Auth.Type "bearer_token"}}authorization: str | None = Header(None){{end}}):
    """Verify authentication for {{.Name}} endpoint."""
    {{if eq .Auth.Type "api_key"}}
    expected_key = getattr(settings, "{{.Auth.EnvVar | lower}}", None)
    if not expected_key:
        return  # Auth optional if not configured
    if {{.Auth.Header | lower | replace "-" "_"}} is None:
        raise HTTPException(status_code=401, detail="API key required")
    if {{.Auth.Header | lower | replace "-" "_"}} != expected_key:
        raise HTTPException(status_code=401, detail="Invalid API key")
    {{else if eq .Auth.Type "bearer_token"}}
    expected_token = getattr(settings, "{{.Auth.EnvVar | lower}}", None)
    if not expected_token:
        return  # Auth optional if not configured
    if authorization is None:
        raise HTTPException(status_code=401, detail="Bearer token required")
    if not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Invalid authorization format")
    token = authorization[7:]
    if token != expected_token:
        raise HTTPException(status_code=401, detail="Invalid bearer token")
    {{end}}
{{end}}

{{if and .Webhook .Webhook.SignatureVerification (eq .Webhook.SignatureVerification "hmac_sha256")}}
def verify_{{.Name}}_signature(request: Request, body: bytes):
    """Verify HMAC signature for {{.Name}} webhook."""
    signature = request.headers.get("{{.Webhook.SignatureHeader}}")
    if not signature:
        raise HTTPException(status_code=401, detail="Missing signature")

    secret = getattr(settings, "{{.Webhook.SecretEnv | lower}}", None)
    if not secret:
        return  # Verification optional if secret not configured

    expected = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    if not hmac.compare_digest(signature, expected):
        raise HTTPException(status_code=401, detail="Invalid signature")
{{end}}

async def {{.Name}}_task(payload: {{.GetInputModelName}}, request_id: str):
    """Background task for {{.Name}}."""
    try:
        executor = agent_executors["{{.Name}}"]
        await executor.execute(payload.model_dump(), request_id)
    except Exception as e:
        log_event(
            "background_task_error",
            request_id=request_id,
            service="{{.Name}}",
            error=str(e),
            error_type=type(e).__name__,
        )

@app.post("{{.WebhookPath}}")
async def {{.GetFunctionName}}(
    request: Request,
    payload: {{.GetInputModelName}},
    background_tasks: BackgroundTasks,
    {{if .Auth}}_: None = Depends(verify_{{.Name}}_auth),{{end}}
):
    """
    {{.Description}}

    Type: Webhook (async background processing)
    """
    request_id = request.state.request_id

    {{if and .Webhook .Webhook.SignatureVerification (eq .Webhook.SignatureVerification "hmac_sha256")}}
    body = await request.body()
    verify_{{.Name}}_signature(request, body)
    {{end}}

    log_event("webhook_queued", request_id=request_id, service="{{.Name}}")
    background_tasks.add_task({{.Name}}_task, payload, request_id)

    return {"status": "accepted", "request_id": request_id, "message": "Processing in background"}

{{else if eq .Type "api"}}
# API endpoint: {{.Name}}
{{if .Auth}}
async def verify_{{.Name}}_auth({{if eq .Auth.Type "api_key"}}{{.Auth.Header | lower | replace "-" "_"}}: str | None = Header(None, alias="{{.Auth.Header}}"){{else if eq .Auth.Type "bearer_token"}}authorization: str | None = Header(None){{end}}):
    """Verify authentication for {{.Name}} endpoint."""
    {{if eq .Auth.Type "api_key"}}
    expected_key = getattr(settings, "{{.Auth.EnvVar | lower}}", None)
    if not expected_key:
        return  # Auth optional if not configured
    if {{.Auth.Header | lower | replace "-" "_"}} is None:
        raise HTTPException(status_code=401, detail="API key required")
    if {{.Auth.Header | lower | replace "-" "_"}} != expected_key:
        raise HTTPException(status_code=401, detail="Invalid API key")
    {{else if eq .Auth.Type "bearer_token"}}
    expected_token = getattr(settings, "{{.Auth.EnvVar | lower}}", None)
    if not expected_token:
        return  # Auth optional if not configured
    if authorization is None:
        raise HTTPException(status_code=401, detail="Bearer token required")
    if not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Invalid authorization format")
    token = authorization[7:]
    if token != expected_token:
        raise HTTPException(status_code=401, detail="Invalid bearer token")
    {{end}}
{{end}}

@app.post("{{.APIPath}}"{{if .OutputSchema}}, response_model={{.GetOutputModelName}}{{end}})
async def {{.GetFunctionName}}(
    request: Request,
    payload: {{.GetInputModelName}},
    {{if .Auth}}_: None = Depends(verify_{{.Name}}_auth),{{end}}
):
    """
    {{.Description}}

    Type: API (synchronous)
    {{if .API}}Timeout: {{.API.Timeout}}s{{end}}
    """
    request_id = request.state.request_id

    try:
        executor = agent_executors["{{.Name}}"]
        result = await executor.execute(payload.model_dump(), request_id)
        {{if .OutputSchema}}
        # TODO: Parse result into {{.GetOutputModelName}}
        return {{.GetOutputModelName}}(result=result)
        {{else}}
        return {"status": "completed", "request_id": request_id, "result": result}
        {{end}}
    except Exception as e:
        log_event("api_error", request_id=request_id, service="{{.Name}}", error=str(e))
        raise HTTPException(status_code=500, detail="Agent execution failed")

{{else if eq .Type "streaming"}}
# Streaming endpoint: {{.Name}}
{{if .Auth}}
async def verify_{{.Name}}_auth({{if eq .Auth.Type "api_key"}}{{.Auth.Header | lower | replace "-" "_"}}: str | None = Header(None, alias="{{.Auth.Header}}"){{else if eq .Auth.Type "bearer_token"}}authorization: str | None = Header(None){{end}}):
    """Verify authentication for {{.Name}} endpoint."""
    {{if eq .Auth.Type "api_key"}}
    expected_key = getattr(settings, "{{.Auth.EnvVar | lower}}", None)
    if not expected_key:
        return  # Auth optional if not configured
    if {{.Auth.Header | lower | replace "-" "_"}} is None:
        raise HTTPException(status_code=401, detail="API key required")
    if {{.Auth.Header | lower | replace "-" "_"}} != expected_key:
        raise HTTPException(status_code=401, detail="Invalid API key")
    {{else if eq .Auth.Type "bearer_token"}}
    expected_token = getattr(settings, "{{.Auth.EnvVar | lower}}", None)
    if not expected_token:
        return  # Auth optional if not configured
    if authorization is None:
        raise HTTPException(status_code=401, detail="Bearer token required")
    if not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Invalid authorization format")
    token = authorization[7:]
    if token != expected_token:
        raise HTTPException(status_code=401, detail="Invalid bearer token")
    {{end}}
{{end}}

@app.post("{{.APIPath}}")
async def {{.GetFunctionName}}(
    request: Request,
    payload: {{.GetInputModelName}},
    {{if .Auth}}_: None = Depends(verify_{{.Name}}_auth),{{end}}
):
    """
    {{.Description}}

    Type: Streaming (SSE)
    """
    request_id = request.state.request_id

    async def event_generator():
        try:
            executor = agent_executors["{{.Name}}"]
            async for chunk in executor.stream_execute(payload.model_dump(), request_id):
                {{if and .Streaming (eq .Streaming.Format "json")}}
                import json
                yield f"data: {json.dumps({'text': chunk})}\n\n"
                {{else}}
                yield f"data: {chunk}\n\n"
                {{end}}
            yield "event: done\ndata: [DONE]\n\n"
        except Exception as e:
            log_event("streaming_error", request_id=request_id, service="{{.Name}}", error=str(e))
            yield f"event: error\ndata: {str(e)}\n\n"

    headers = {"X-Request-ID": request_id}
    return StreamingResponse(event_generator(), media_type="text/event-stream", headers=headers)

{{end}}`

	tmpl, err := template.New("endpoint").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, svc); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateModelCode generates the Pydantic model code for a single service
func generateModelCode(svc *config.Service) (string, error) {
	tmplStr := `# Models for {{.Name}} service
class {{.GetInputModelName}}(BaseModel):
    """Input model for {{.Name}} endpoint."""
    {{range .InputSchema.Fields}}
    {{.Name}}: {{if eq .Type "str"}}str{{else if eq .Type "int"}}int{{else if eq .Type "float"}}float{{else if eq .Type "bool"}}bool{{else if eq .Type "list"}}List[Any]{{else if eq .Type "dict"}}Dict[str, Any]{{else}}Any{{end}}{{if not .Required}} | None = None{{end}}{{if .Default}} = "{{.Default}}"{{end}}
    {{end}}

{{if .OutputSchema}}
class {{.GetOutputModelName}}(BaseModel):
    """Output model for {{.Name}} endpoint."""
    {{range .OutputSchema.Fields}}
    {{.Name}}: {{if eq .Type "str"}}str{{else if eq .Type "int"}}int{{else if eq .Type "float"}}float{{else if eq .Type "bool"}}bool{{else if eq .Type "list"}}List[Any]{{else if eq .Type "dict"}}Dict[str, Any]{{else}}Any{{end}}{{if not .Required}} | None = None{{end}}{{if .Default}} = "{{.Default}}"{{end}}
    {{end}}
{{end}}
`

	tmpl, err := template.New("models").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, svc); err != nil {
		return "", err
	}

	return buf.String(), nil
}
