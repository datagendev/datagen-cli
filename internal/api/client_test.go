package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDeployCustomTool(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/mcp/deployments/standalone" {
			t.Fatalf("path = %s, want /mcp/deployments/standalone", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("Decode(request) error = %v", err)
		}
		_ = json.NewEncoder(w).Encode(DeployCustomToolResponse{
			Success: true,
			Data: DeployCustomToolData{
				DeploymentUUID: "tool-123",
				Status:         "deployed",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.BaseURL = server.URL
	client.HTTPClient = server.Client()

	resp, err := client.DeployCustomTool(DeployCustomToolRequest{
		Name:              "demo_tool",
		FinalCode:         "print('hi')",
		AdditionalImports: []string{"requests"},
		RequiredSecrets:   []string{"OPENAI_API_KEY"},
		DeploymentType:    0,
	})
	if err != nil {
		t.Fatalf("DeployCustomTool() error = %v", err)
	}
	if resp.Data.DeploymentUUID != "tool-123" {
		t.Fatalf("DeployCustomTool() deployment_uuid = %q, want tool-123", resp.Data.DeploymentUUID)
	}
	if got := captured["name"]; got != "demo_tool" {
		t.Fatalf("request name = %v, want demo_tool", got)
	}
	if got := captured["additional_imports"]; !reflect.DeepEqual(got, []interface{}{"requests"}) {
		t.Fatalf("request additional_imports = %v, want [requests]", got)
	}
	if got := captured["required_secrets"]; !reflect.DeepEqual(got, []interface{}{"OPENAI_API_KEY"}) {
		t.Fatalf("request required_secrets = %v, want [OPENAI_API_KEY]", got)
	}
}

func TestUpdateCustomToolOnlySendsProvidedFields(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/mcp/deployment/tool-123" {
			t.Fatalf("path = %s, want /mcp/deployment/tool-123", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("Decode(request) error = %v", err)
		}
		_ = json.NewEncoder(w).Encode(UpdateCustomToolResponse{
			Success: true,
			Data:    CustomToolDetail{DeploymentUUID: "tool-123"},
		})
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.BaseURL = server.URL
	client.HTTPClient = server.Client()

	description := "updated description"
	_, err := client.UpdateCustomTool("tool-123", UpdateCustomToolRequest{
		Description: &description,
	})
	if err != nil {
		t.Fatalf("UpdateCustomTool() error = %v", err)
	}

	if len(captured) != 1 {
		t.Fatalf("request field count = %d, want 1 (%v)", len(captured), captured)
	}
	if got := captured["description"]; got != description {
		t.Fatalf("request description = %v, want %s", got, description)
	}
}

func TestValidateAndRunCustomTool(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/mcp/apps/tool-123/validate-auth":
			if r.Method != http.MethodPost {
				t.Fatalf("validate method = %s, want POST", r.Method)
			}
			_ = json.NewEncoder(w).Encode(ValidateCustomToolResponse{
				Success: true,
				Data: ValidateCustomToolResponseData{
					IsValid: true,
				},
			})
		case "/mcp/apps/tool-123/async":
			if r.Method != http.MethodPost {
				t.Fatalf("run method = %s, want POST", r.Method)
			}
			var captured RunCustomToolRequest
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("Decode(run request) error = %v", err)
			}
			if got := captured.InputVars["url"]; got != "https://example.com" {
				t.Fatalf("run input url = %v, want https://example.com", got)
			}
			_ = json.NewEncoder(w).Encode(RunCustomToolResponse{
				Success: true,
				Data: RunCustomToolResponseData{
					RunUUID: "run-123",
					Status:  "pending",
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.BaseURL = server.URL
	client.HTTPClient = server.Client()

	validateResp, err := client.ValidateCustomTool("tool-123")
	if err != nil {
		t.Fatalf("ValidateCustomTool() error = %v", err)
	}
	if !validateResp.Data.IsValid {
		t.Fatalf("ValidateCustomTool() is_valid = false, want true")
	}

	runResp, err := client.RunCustomTool("tool-123", map[string]interface{}{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("RunCustomTool() error = %v", err)
	}
	if runResp.Data.RunUUID != "run-123" {
		t.Fatalf("RunCustomTool() run_uuid = %q, want run-123", runResp.Data.RunUUID)
	}
	if callCount != 2 {
		t.Fatalf("HTTP calls = %d, want 2", callCount)
	}
}
