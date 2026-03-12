package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildDeployCustomToolRequest(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "tool.py")
	if err := os.WriteFile(scriptPath, []byte("import os\nimport requests\nfrom bs4 import BeautifulSoup\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(tool.py) error = %v", err)
	}

	req, err := buildDeployCustomToolRequest("demo_tool", deployToolOptions{
		FilePath:      scriptPath,
		Description:   "Demo tool",
		Outputs:       "result,status",
		ExpectedTools: "mcp_linear_list_projects,custom_helper",
		Secrets:       "OPENAI_API_KEY",
	})
	if err != nil {
		t.Fatalf("buildDeployCustomToolRequest() error = %v", err)
	}

	if req.Name != "demo_tool" {
		t.Fatalf("req.Name = %q, want demo_tool", req.Name)
	}
	if !reflect.DeepEqual(req.AdditionalImports, []string{"bs4", "requests"}) {
		t.Fatalf("req.AdditionalImports = %v, want [bs4 requests]", req.AdditionalImports)
	}
	if !reflect.DeepEqual(req.OutputVarsList, []string{"result", "status"}) {
		t.Fatalf("req.OutputVarsList = %v, want [result status]", req.OutputVarsList)
	}
	if !reflect.DeepEqual(req.MCPToolNames, []string{"mcp_linear_list_projects"}) {
		t.Fatalf("req.MCPToolNames = %v, want [mcp_linear_list_projects]", req.MCPToolNames)
	}
}

func TestBuildUpdateCustomToolRequestPartial(t *testing.T) {
	req, hasChanges, err := buildUpdateCustomToolRequest(toolsUpdateCmd, updateToolOptions{
		Description:    "Updated",
		HasDescription: true,
	})
	if err != nil {
		t.Fatalf("buildUpdateCustomToolRequest() error = %v", err)
	}
	if !hasChanges {
		t.Fatalf("hasChanges = false, want true")
	}
	if req.Description == nil || *req.Description != "Updated" {
		t.Fatalf("req.Description = %v, want Updated", req.Description)
	}
	if req.FinalCode != nil {
		t.Fatalf("req.FinalCode = %v, want nil", req.FinalCode)
	}
}

func TestParseRunInputFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.json")
	if err := os.WriteFile(inputPath, []byte(`{"url":"https://example.com","count":2}`), 0o644); err != nil {
		t.Fatalf("WriteFile(input.json) error = %v", err)
	}

	got, err := parseRunInput("", inputPath)
	if err != nil {
		t.Fatalf("parseRunInput() error = %v", err)
	}
	if got["url"] != "https://example.com" {
		t.Fatalf("parseRunInput() url = %v, want https://example.com", got["url"])
	}
}

func TestRunToolsRunValidatesBeforeExecuting(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("DATAGEN_API_KEY", "test-key")

	requestPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.Path)
		switch r.URL.Path {
		case "/mcp/apps/tool-123/validate-auth":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"is_valid": true,
				},
			})
		case "/mcp/apps/tool-123/async":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode(run body) error = %v", err)
			}
			if inputVars, ok := body["input_vars"].(map[string]interface{}); !ok || inputVars["url"] != "https://example.com" {
				t.Fatalf("input_vars = %v, want url=https://example.com", body["input_vars"])
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"run_uuid": "run-123",
					"status":   "pending",
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("DATAGEN_API_BASE_URL", server.URL)

	resetToolGlobals()
	toolInput = `{"url":"https://example.com"}`
	defer resetToolGlobals()

	var stdout bytes.Buffer
	toolsRunCmd.SetOut(&stdout)
	toolsRunCmd.SetErr(&stdout)

	if err := runToolsRun(toolsRunCmd, []string{"tool-123"}); err != nil {
		t.Fatalf("runToolsRun() error = %v", err)
	}

	wantPaths := []string{"/mcp/apps/tool-123/validate-auth", "/mcp/apps/tool-123/async"}
	if !reflect.DeepEqual(requestPaths, wantPaths) {
		t.Fatalf("requestPaths = %v, want %v", requestPaths, wantPaths)
	}
}

func TestRunToolsListMissingAPIKey(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("DATAGEN_API_KEY", "")

	err := runToolsList(toolsListCmd, nil)
	if err == nil {
		t.Fatalf("runToolsList() error = nil, want missing API key error")
	}
	if !strings.Contains(err.Error(), "DATAGEN_API_KEY not found") {
		t.Fatalf("runToolsList() error = %q, want DATAGEN_API_KEY not found", err)
	}
}

func resetToolGlobals() {
	toolCode = ""
	toolFile = ""
	toolDescription = ""
	toolSchema = ""
	toolSchemaFile = ""
	toolDefaults = ""
	toolDefaultsFile = ""
	toolOutputs = ""
	toolExpectedTools = ""
	toolImports = ""
	toolNoAutoImports = false
	toolMCPServers = ""
	toolSecrets = ""
	toolPublic = false
	toolInput = ""
	toolInputFile = ""
}
