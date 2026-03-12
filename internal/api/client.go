package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	DefaultBaseURL = "https://api.datagen.dev"
	DefaultTimeout = 30 * time.Second
)

// Client is the DataGen API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new DataGen API client
func NewClient(apiKey string) *Client {
	baseURL := os.Getenv("DATAGEN_API_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ==========================================
// GitHub Installation Methods
// ==========================================

// GetGitHubInstallUrl returns the GitHub App installation URL
func (c *Client) GetGitHubInstallUrl() (*InstallUrlResponse, error) {
	body, err := c.doRequest("GET", "/api/cli/github/install-url", nil)
	if err != nil {
		return nil, err
	}

	var resp InstallUrlResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ListGitHubInstallations returns all GitHub App installations
func (c *Client) ListGitHubInstallations() (*ListInstallationsResponse, error) {
	body, err := c.doRequest("GET", "/api/cli/github/installations", nil)
	if err != nil {
		return nil, err
	}

	var resp ListInstallationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ==========================================
// Repository Methods
// ==========================================

// ListAvailableRepos returns all repos available via GitHub App
func (c *Client) ListAvailableRepos() (*ListAvailableReposResponse, error) {
	body, err := c.doRequest("GET", "/api/cli/github/repos", nil)
	if err != nil {
		return nil, err
	}

	var resp ListAvailableReposResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ListConnectedRepos returns all connected repos for the user
func (c *Client) ListConnectedRepos() (*ListConnectedReposResponse, error) {
	body, err := c.doRequest("GET", "/api/cli/github/repos/connected", nil)
	if err != nil {
		return nil, err
	}

	var resp ListConnectedReposResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ConnectRepo connects a repository by full name (owner/repo)
func (c *Client) ConnectRepo(fullName string) (*ConnectRepoResponse, error) {
	body, err := c.doRequest("POST", "/api/cli/github/repos/connect", ConnectRepoRequest{
		FullName: fullName,
	})
	if err != nil {
		return nil, err
	}

	var resp ConnectRepoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// SyncRepo syncs/refreshes agents in a connected repo
func (c *Client) SyncRepo(repoID string) (*SyncRepoResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/api/cli/github/repos/%s/sync", repoID), nil)
	if err != nil {
		return nil, err
	}

	var resp SyncRepoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ==========================================
// Agent Methods
// ==========================================

// ListAgents returns all discovered agents for the user
func (c *Client) ListAgents() (*ListAgentsResponse, error) {
	body, err := c.doRequest("GET", "/api/cli/agents", nil)
	if err != nil {
		return nil, err
	}

	var resp ListAgentsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// GetAgent returns details of a specific agent
func (c *Client) GetAgent(agentID string) (*GetAgentResponse, error) {
	body, err := c.doRequest("GET", fmt.Sprintf("/api/cli/agents/%s", agentID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetAgentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// DeployAgent deploys an agent (creates webhook)
func (c *Client) DeployAgent(agentID string, callbackUrl string, secretNames []string) (*DeployAgentResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/api/cli/agents/%s/deploy", agentID), DeployAgentRequest{
		CallbackUrl: callbackUrl,
		SecretNames: secretNames,
	})
	if err != nil {
		return nil, err
	}

	var resp DeployAgentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// UndeployAgent undeploys an agent (removes webhook)
func (c *Client) UndeployAgent(agentID string) (*UndeployAgentResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/api/cli/agents/%s/undeploy", agentID), nil)
	if err != nil {
		return nil, err
	}

	var resp UndeployAgentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// RunAgent triggers agent execution
func (c *Client) RunAgent(agentID string, payload interface{}) (*RunAgentResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/api/cli/agents/%s/run", agentID), RunAgentRequest{
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}

	var resp RunAgentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ==========================================
// Secret Methods
// ==========================================

// ListSecrets returns the user's secrets (masked values only)
// Reuses the existing mcpGetUserSecretKeys endpoint
func (c *Client) ListSecrets() (*ListSecretsResponse, error) {
	body, err := c.doRequest("GET", "/mcp/user-secrets/keys", nil)
	if err != nil {
		return nil, err
	}

	var resp ListSecretsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// UpsertSecret creates or updates a secret
func (c *Client) UpsertSecret(name, value string) (*UpsertSecretResponse, error) {
	body, err := c.doRequest("POST", "/mcp/cli/secrets/upsert", UpsertSecretRequest{
		Name:  name,
		Value: value,
	})
	if err != nil {
		return nil, err
	}

	var resp UpsertSecretResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ==========================================
// Agent Config Methods
// ==========================================

// GetAgentConfig returns the unified configuration for an agent
func (c *Client) GetAgentConfig(agentID string) (*AgentConfigResponse, error) {
	body, err := c.doRequest("GET", fmt.Sprintf("/api/agents/%s/config", agentID), nil)
	if err != nil {
		return nil, err
	}

	var resp AgentConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// UpdateAgentConfig updates the unified configuration for an agent
func (c *Client) UpdateAgentConfig(agentID string, req UpdateAgentConfigRequest) ([]byte, error) {
	return c.doRequest("POST", fmt.Sprintf("/api/agents/%s/config", agentID), req)
}

// ==========================================
// Agent Schedule Methods
// ==========================================

// ListAgentSchedules returns schedules for an agent
func (c *Client) ListAgentSchedules(agentID string) (*ListSchedulesResponse, error) {
	body, err := c.doRequest("GET", fmt.Sprintf("/api/cli/agents/%s/schedules", agentID), nil)
	if err != nil {
		return nil, err
	}

	var resp ListSchedulesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// CreateAgentSchedule creates a cron schedule for an agent
func (c *Client) CreateAgentSchedule(agentID string, req CreateScheduleRequest) (*CreateScheduleResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/api/cli/agents/%s/schedules", agentID), req)
	if err != nil {
		return nil, err
	}

	var resp CreateScheduleResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// UpdateAgentSchedule updates a schedule (pause/resume)
func (c *Client) UpdateAgentSchedule(agentID, scheduleID string, updates map[string]interface{}) ([]byte, error) {
	return c.doRequest("POST", fmt.Sprintf("/api/cli/agents/%s/schedules/%s", agentID, scheduleID), updates)
}

// DeleteAgentSchedule deletes a schedule
func (c *Client) DeleteAgentSchedule(agentID, scheduleID string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/api/cli/agents/%s/schedules/%s", agentID, scheduleID), nil)
	return err
}

// ListAgentExecutions returns executions for an agent
func (c *Client) ListAgentExecutions(agentID string, limit int) (*ListExecutionsResponse, error) {
	path := fmt.Sprintf("/api/cli/agents/%s/executions", agentID)
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}

	body, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp ListExecutionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ==========================================
// Custom Tool Methods
// ==========================================

// ListCustomTools returns the user's custom tools.
func (c *Client) ListCustomTools(limit int) (*ListCustomToolsResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	body, err := c.doRequest("GET", fmt.Sprintf("/mcp/apps?limit=%d", limit), nil)
	if err != nil {
		return nil, err
	}

	var resp ListCustomToolsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// GetCustomTool returns details of a specific custom tool.
func (c *Client) GetCustomTool(toolUUID string) (*GetCustomToolResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/mcp/apps/%s", toolUUID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetCustomToolResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// DeployCustomTool creates a new custom tool deployment.
func (c *Client) DeployCustomTool(req DeployCustomToolRequest) (*DeployCustomToolResponse, error) {
	body, err := c.doRequest("POST", "/mcp/deployments/standalone", req)
	if err != nil {
		return nil, err
	}

	var resp DeployCustomToolResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// UpdateCustomTool updates an existing custom tool deployment.
func (c *Client) UpdateCustomTool(toolUUID string, req UpdateCustomToolRequest) (*UpdateCustomToolResponse, error) {
	body, err := c.doRequest("PUT", fmt.Sprintf("/mcp/deployment/%s", toolUUID), req)
	if err != nil {
		return nil, err
	}

	var resp UpdateCustomToolResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ValidateCustomTool validates secret and MCP requirements before a run.
func (c *Client) ValidateCustomTool(toolUUID string) (*ValidateCustomToolResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/mcp/apps/%s/validate-auth", toolUUID), map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var resp ValidateCustomToolResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// RunCustomTool triggers an asynchronous custom tool run.
func (c *Client) RunCustomTool(toolUUID string, inputVars map[string]interface{}) (*RunCustomToolResponse, error) {
	body, err := c.doRequest("POST", fmt.Sprintf("/mcp/apps/%s/async", toolUUID), RunCustomToolRequest{
		InputVars: inputVars,
	})
	if err != nil {
		return nil, err
	}

	var resp RunCustomToolResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}
