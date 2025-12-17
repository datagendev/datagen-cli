package mcpconfig

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpdateCodexConfig_PreservesOtherFeatures(t *testing.T) {
	input := `[features]
some_other_flag = false

[mcp_servers.other]
url = "https://example.com/mcp"
`

	out, changed, err := UpdateCodexConfig(input, "", true, "DATAGEN_API_KEY")
	if err != nil {
		t.Fatalf("UpdateCodexConfig() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if !strings.Contains(out, "some_other_flag = false") {
		t.Fatalf("expected other feature preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "rmcp_client = true") {
		t.Fatalf("expected rmcp_client true, got:\n%s", out)
	}
	if !strings.Contains(out, "[mcp_servers.datagen]") {
		t.Fatalf("expected datagen table, got:\n%s", out)
	}
	if !strings.Contains(out, `env_http_headers = { "x-api-key" = "DATAGEN_API_KEY" }`) {
		t.Fatalf("expected env_http_headers, got:\n%s", out)
	}
}

func TestUpdateCodexConfig_ReplacesExistingDatagenTable(t *testing.T) {
	input := `[mcp_servers.datagen]
url = "https://old.example/mcp"
http_headers = { "x-api-key" = "old" }
`

	out, _, err := UpdateCodexConfig(input, "newkey", false, "")
	if err != nil {
		t.Fatalf("UpdateCodexConfig() error = %v", err)
	}
	if strings.Contains(out, "old.example") || strings.Contains(out, `"old"`) {
		t.Fatalf("expected old settings replaced, got:\n%s", out)
	}
	if !strings.Contains(out, `http_headers = { "x-api-key" = "newkey" }`) {
		t.Fatalf("expected new http_headers, got:\n%s", out)
	}
}

func TestUpdateClaudeConfig_RemovesTopLevelMcpServers(t *testing.T) {
	t.Setenv("HOME", "/Users/testuser")

	input := `{
  "cachedChangelog": "x",
  "mcpServers": {
    "playwright": {
      "type": "stdio"
    }
  },
  "/Users/testuser": {
    "mcpServers": {
      "Existing": {
        "type": "http",
        "url": "https://example.com"
      }
    }
  }
}`

	out, changed, err := UpdateClaudeConfig(input, "k123")
	if err != nil {
		t.Fatalf("UpdateClaudeConfig() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := root["cachedChangelog"]; !ok {
		t.Fatalf("expected cachedChangelog preserved")
	}
	if _, ok := root["mcpServers"]; ok {
		t.Fatalf("expected top-level mcpServers removed")
	}
	sec, _ := root["/Users/testuser"].(map[string]any)
	if sec == nil {
		t.Fatalf("expected home section present")
	}
	servers, _ := sec["mcpServers"].(map[string]any)
	if servers == nil {
		t.Fatalf("expected home mcpServers present")
	}
	if _, ok := servers["Existing"]; !ok {
		t.Fatalf("expected Existing server preserved")
	}
	if _, ok := servers["playwright"]; !ok {
		t.Fatalf("expected top-level servers merged into home section")
	}
	if _, ok := servers["Datagen"]; !ok {
		t.Fatalf("expected Datagen server added")
	}
}

func TestUpdateMCPJSONConfig_AddsDatagen(t *testing.T) {
	input := `{
  "mcpServers": {
    "dropbox": {
      "command": "npx",
      "args": ["-y", "@klavis/mcp-dropbox-server"],
      "env": {"DROPBOX_SERVER_URL": "x"}
    }
  }
}`

	out, changed, err := UpdateMCPJSONConfig(input, "k123")
	if err != nil {
		t.Fatalf("UpdateMCPJSONConfig() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if !strings.Contains(out, `"datagen"`) || !strings.Contains(out, DatagenMCPURL) {
		t.Fatalf("expected datagen server added, got:\n%s", out)
	}
}
