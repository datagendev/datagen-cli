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

func TestUpdateClaudeConfig_WritesTopLevelMCPServers(t *testing.T) {
	t.Setenv("HOME", "/Users/testuser")

	input := `{
  "cachedChangelog": "x",
  "cachedGrowthBookFeatures": {
    "persimmon_marble_flag": "N/A"
  }
}`

	out, changed, err := UpdateClaudeConfig(input, "k123")
	if err != nil {
		t.Fatalf("UpdateClaudeConfig() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}
	// Ensure "headers" is last within the datagen server object.
	idxType := strings.Index(out, `"type": "http"`)
	idxURL := strings.Index(out, `"url": "`+DatagenMCPURL+`"`)
	idxHeaders := strings.Index(out, `"headers": {`)
	if idxType == -1 || idxURL == -1 || idxHeaders == -1 {
		t.Fatalf("expected datagen server fields present, got:\n%s", out)
	}
	if !(idxType < idxURL && idxURL < idxHeaders) {
		t.Fatalf("expected field order type < url < headers, got indices type=%d url=%d headers=%d\n%s", idxType, idxURL, idxHeaders, out)
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := root["cachedChangelog"]; !ok {
		t.Fatalf("expected cachedChangelog preserved")
	}
	if _, ok := root["/Users/testuser"]; ok {
		t.Fatalf("did not expect per-home section to be created")
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		t.Fatalf("expected top-level mcpServers present")
	}
	if _, ok := servers["datagen"]; !ok {
		t.Fatalf("expected datagen server added")
	}

	features, _ := root["cachedGrowthBookFeatures"].(map[string]any)
	if features == nil || features["persimmon_marble_flag"] != "N/A" {
		t.Fatalf("expected cachedGrowthBookFeatures preserved")
	}
}
