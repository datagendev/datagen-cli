package api

import "time"

// GitHub Installation types

type Installation struct {
	ID             string    `json:"id"`
	InstallationID int       `json:"installationId"`
	AccountLogin   string    `json:"accountLogin"`
	AccountType    string    `json:"accountType"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ListInstallationsResponse struct {
	Installations []Installation `json:"installations"`
}

type InstallUrlResponse struct {
	InstallUrl string `json:"installUrl"`
	StateToken string `json:"stateToken"`
	Message    string `json:"message"`
}

// Repository types

type Repo struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"`
	Private       bool   `json:"private"`
	HtmlUrl       string `json:"htmlUrl"`
	DefaultBranch string `json:"defaultBranch"`
	IsConnected   bool   `json:"isConnected"`
}

type InstallationWithRepos struct {
	Installation Installation `json:"installation"`
	Repos        []Repo       `json:"repos"`
	Error        string       `json:"error,omitempty"`
}

type ListAvailableReposResponse struct {
	Installations []InstallationWithRepos `json:"installations"`
	Message       string                  `json:"message,omitempty"`
}

type ConnectedRepo struct {
	ID                 string     `json:"id"`
	FullName           string     `json:"fullName"`
	HtmlUrl            string     `json:"htmlUrl"`
	DefaultBranch      string     `json:"defaultBranch"`
	SyncStatus         string     `json:"syncStatus"`
	LastSyncedAt       *time.Time `json:"lastSyncedAt"`
	AgentCount         int        `json:"agentCount"`
	DeployedAgentCount int        `json:"deployedAgentCount"`
}

type ListConnectedReposResponse struct {
	Repos []ConnectedRepo `json:"repos"`
}

type ConnectRepoRequest struct {
	FullName string `json:"fullName"`
}

type AgentSummary struct {
	ID          string `json:"id"`
	AgentName   string `json:"agentName"`
	FilePath    string `json:"filePath"`
	Description string `json:"description,omitempty"`
	IsDeployed  bool   `json:"isDeployed,omitempty"`
	IsMissing   bool   `json:"isMissing,omitempty"`
}

type ConnectRepoResponse struct {
	Success          bool           `json:"success"`
	Repo             ConnectedRepo  `json:"repo"`
	AgentsDiscovered int            `json:"agentsDiscovered"`
	Agents           []AgentSummary `json:"agents"`
}

type SyncRepoResponse struct {
	Success       bool           `json:"success"`
	AgentsFound   int            `json:"agentsFound"`
	NewAgents     int            `json:"newAgents"`
	UpdatedAgents int            `json:"updatedAgents"`
	Agents        []AgentSummary `json:"agents"`
}

// Agent types

type RepoInfo struct {
	FullName string `json:"fullName"`
	HtmlUrl  string `json:"htmlUrl"`
}

type WebhookInfo struct {
	IsActive        bool       `json:"isActive"`
	LastTriggeredAt *time.Time `json:"lastTriggeredAt"`
}

type Agent struct {
	ID          string                 `json:"id"`
	AgentName   string                 `json:"agentName"`
	Description string                 `json:"description,omitempty"`
	FilePath    string                 `json:"filePath"`
	Type        string                 `json:"type,omitempty"`
	IsDeployed  bool                   `json:"isDeployed"`
	IsMissing   bool                   `json:"isMissing"`
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty"`
	Repo        RepoInfo               `json:"repo"`
	Webhook     *WebhookInfo           `json:"webhook,omitempty"`
}

type ListAgentsResponse struct {
	Agents []Agent `json:"agents"`
}

type AgentDetailWebhook struct {
	WebhookToken     string     `json:"webhookToken"`
	WebhookUrl       string     `json:"webhookUrl"`
	IsActive         bool       `json:"isActive"`
	LastTriggeredAt  *time.Time `json:"lastTriggeredAt"`
	RateLimitPerHour int        `json:"rateLimitPerHour"`
}

type AgentDetail struct {
	ID          string                 `json:"id"`
	AgentName   string                 `json:"agentName"`
	Description string                 `json:"description,omitempty"`
	FilePath    string                 `json:"filePath"`
	Type        string                 `json:"type,omitempty"`
	IsDeployed  bool                   `json:"isDeployed"`
	IsMissing   bool                   `json:"isMissing"`
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty"`
	EntryPrompt string                 `json:"entryPrompt,omitempty"`
	Repo        RepoInfo               `json:"repo"`
	Webhook     *AgentDetailWebhook    `json:"webhook,omitempty"`
}

type ExecutionSummary struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type GetAgentResponse struct {
	Agent            AgentDetail        `json:"agent"`
	RecentExecutions []ExecutionSummary `json:"recentExecutions,omitempty"`
}

type DeployAgentRequest struct {
	CallbackUrl string   `json:"callbackUrl,omitempty"`
	SecretNames []string `json:"secretNames,omitempty"`
}

type DeployAgentResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	WebhookUrl   string `json:"webhookUrl"`
	WebhookToken string `json:"webhookToken"`
	AgentID      string `json:"agentId"`
	AgentName    string `json:"agentName"`
}

type UndeployAgentResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
}

type RunAgentRequest struct {
	Payload interface{} `json:"payload,omitempty"`
}

type RunAgentResponse struct {
	Success     bool   `json:"success"`
	ExecutionID string `json:"executionId"`
	Status      string `json:"status"`
	StatusUrl   string `json:"statusUrl"`
	Message     string `json:"message"`
}

type Execution struct {
	ID           string                 `json:"id"`
	Status       string                 `json:"status"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	Result       map[string]interface{} `json:"result,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
	SdkSessionID *string                `json:"sdkSessionId,omitempty"`
	PrUrl        string                 `json:"prUrl,omitempty"`
	DurationMs   *int                   `json:"durationMs,omitempty"`
}

type ExecutionOutputResponse struct {
	ExecutionID  string                 `json:"executionId"`
	AgentID      string                 `json:"agentId"`
	AgentName    string                 `json:"agentName"`
	Type         string                 `json:"type,omitempty"`
	Status       string                 `json:"status"`
	SdkSessionID *string                `json:"sdkSessionId,omitempty"`
	Result       map[string]interface{} `json:"result,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	PrUrl        string                 `json:"prUrl,omitempty"`
	AgentBranch  string                 `json:"agentBranch,omitempty"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
	DurationMs   *int                   `json:"durationMs,omitempty"`
}

type ListExecutionsResponse struct {
	Executions []Execution `json:"executions"`
}

// Secret types

type Secret struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	MaskedValue string    `json:"maskedValue"`
	Provider    *string   `json:"provider"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ListSecretsData struct {
	SecretKeys []Secret `json:"secretKeys"`
}

type ListSecretsResponse struct {
	Success bool            `json:"success"`
	Data    ListSecretsData `json:"data"`
}

type UpsertSecretRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UpsertSecretResponse struct {
	Secret  Secret `json:"secret"`
	Created bool   `json:"created"`
}

// Agent Config types

type AgentConfigWebhook struct {
	SecretNames []string `json:"secretNames"`
	PrMode      string   `json:"prMode"`
	IsActive    bool     `json:"isActive"`
}

type AgentConfigNotifications struct {
	EmailEnabled      *bool `json:"emailEnabled"`
	EmailOnSuccess    *bool `json:"emailOnSuccess"`
	EmailOnFailure    *bool `json:"emailOnFailure"`
	EmailReplyEnabled *bool `json:"emailReplyEnabled"`
}

type AgentConfigRecipient struct {
	ID    string  `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
	Role  string  `json:"role"`
}

type AgentConfigResponse struct {
	AgentID       string                    `json:"agentId"`
	AgentName     string                    `json:"agentName"`
	Repo          string                    `json:"repo"`
	EntryPrompt   *string                   `json:"entryPrompt"`
	Webhook       *AgentConfigWebhook       `json:"webhook"`
	Notifications *AgentConfigNotifications `json:"notifications"`
	Recipients    []AgentConfigRecipient    `json:"recipients"`
}

type RecipientAdd struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Role  string `json:"role"`
}

type RecipientsUpdate struct {
	Add    []RecipientAdd `json:"add,omitempty"`
	Remove []string       `json:"remove,omitempty"`
}

type UpdateAgentConfigRequest struct {
	EntryPrompt   *string                `json:"entryPrompt,omitempty"`
	Webhook       map[string]interface{} `json:"webhook,omitempty"`
	Notifications map[string]interface{} `json:"notifications,omitempty"`
	Recipients    *RecipientsUpdate      `json:"recipients,omitempty"`
}

// Schedule types

type ScheduleInfo struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	CronExpression string     `json:"cronExpression"`
	Timezone       string     `json:"timezone"`
	IsActive       bool       `json:"isActive"`
	NextRunAt      *time.Time `json:"nextRunAt"`
	LastRunAt      *time.Time `json:"lastRunAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type ListSchedulesResponse struct {
	Schedules []ScheduleInfo `json:"schedules"`
}

type CreateScheduleRequest struct {
	Cron     string `json:"cron"`
	Timezone string `json:"timezone,omitempty"`
	Name     string `json:"name,omitempty"`
}

type CreateScheduleResponse struct {
	Schedule ScheduleInfo `json:"schedule"`
}

// Custom tool types

type CustomToolSummary struct {
	DeploymentUUID     string                 `json:"deployment_uuid"`
	FlowName           string                 `json:"flow_name"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description"`
	RequiredInputVars  []string               `json:"required_input_vars,omitempty"`
	RequiredOutputVars []string               `json:"required_output_vars,omitempty"`
	FinalCode          *string                `json:"final_code,omitempty"`
	DefaultInputVars   map[string]interface{} `json:"default_input_vars,omitempty"`
	CreatedAt          string                 `json:"created_at,omitempty"`
	UpdatedAt          string                 `json:"updated_at,omitempty"`
	DeploymentType     *int                   `json:"deployment_type,omitempty"`
	IsCodePublic       *bool                  `json:"is_code_public,omitempty"`
	IsOwner            *bool                  `json:"is_owner,omitempty"`
}

type ListCustomToolsResponse struct {
	Success bool                `json:"success"`
	Data    []CustomToolSummary `json:"data"`
	Total   int                 `json:"total,omitempty"`
	Page    int                 `json:"page,omitempty"`
	Size    int                 `json:"size,omitempty"`
}

type MCPConfigSummary struct {
	Name       string                 `json:"name"`
	Config     map[string]interface{} `json:"config,omitempty"`
	TemplateID string                 `json:"template_id,omitempty"`
	IsActive   bool                   `json:"is_active,omitempty"`
}

type CustomToolDetail struct {
	DeploymentUUID     string                 `json:"deployment_uuid"`
	FlowName           string                 `json:"flow_name"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description"`
	RequiredInputVars  []string               `json:"required_input_vars,omitempty"`
	RequiredOutputVars []string               `json:"required_output_vars,omitempty"`
	FinalCode          *string                `json:"final_code,omitempty"`
	InputSchema        map[string]interface{} `json:"input_schema,omitempty"`
	OutputSchema       map[string]interface{} `json:"output_schema,omitempty"`
	DefaultInputVars   map[string]interface{} `json:"default_input_vars,omitempty"`
	CreatedAt          string                 `json:"created_at,omitempty"`
	UpdatedAt          string                 `json:"updated_at,omitempty"`
	ExpectedTools      []string               `json:"expected_tools,omitempty"`
	AdditionalImports  []string               `json:"additional_imports,omitempty"`
	RequiredSecrets    []string               `json:"required_secrets,omitempty"`
	MCPConfigs         []MCPConfigSummary     `json:"mcp_configs,omitempty"`
	DeploymentType     *int                   `json:"deployment_type,omitempty"`
	IsCodePublic       *bool                  `json:"is_code_public,omitempty"`
	IsOwner            *bool                  `json:"is_owner,omitempty"`
	OwnerDisplayName   string                 `json:"owner_display_name,omitempty"`
}

type GetCustomToolResponse struct {
	Success bool             `json:"success"`
	Data    CustomToolDetail `json:"data"`
}

type DeployCustomToolRequest struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	FinalCode         string                 `json:"final_code"`
	InputSchema       map[string]interface{} `json:"input_schema,omitempty"`
	OutputVarsList    []string               `json:"output_vars_list,omitempty"`
	ExpectedTools     []string               `json:"expected_tools,omitempty"`
	AdditionalImports []string               `json:"additional_imports,omitempty"`
	DeploymentType    int                    `json:"deployment_type"`
	DefaultInputVars  map[string]interface{} `json:"default_input_vars,omitempty"`
	MCPServerNames    []string               `json:"mcp_server_names,omitempty"`
	MCPToolNames      []string               `json:"mcp_tool_names,omitempty"`
	RequiredSecrets   []string               `json:"required_secrets,omitempty"`
}

type DeployCustomToolData struct {
	DeploymentUUID    string `json:"deployment_uuid"`
	CodeExecutionUUID string `json:"code_execution_uuid,omitempty"`
	Name              string `json:"name,omitempty"`
	Status            string `json:"status,omitempty"`
	Message           string `json:"message,omitempty"`
}

type DeployCustomToolResponse struct {
	Success bool                 `json:"success"`
	Data    DeployCustomToolData `json:"data"`
}

type UpdateCustomToolRequest struct {
	Name              *string                `json:"name,omitempty"`
	Description       *string                `json:"description,omitempty"`
	FinalCode         *string                `json:"final_code,omitempty"`
	InputSchema       map[string]interface{} `json:"input_schema,omitempty"`
	DefaultInputVars  map[string]interface{} `json:"default_input_vars,omitempty"`
	AdditionalImports []string               `json:"additional_imports,omitempty"`
	ExpectedTools     []string               `json:"expected_tools,omitempty"`
	RequiredSecrets   []string               `json:"required_secrets,omitempty"`
	DeploymentType    *int                   `json:"deployment_type,omitempty"`
	MCPServerNames    []string               `json:"mcp_server_names,omitempty"`
}

type UpdateCustomToolResponse struct {
	Success bool             `json:"success"`
	Data    CustomToolDetail `json:"data"`
}

type ValidateCustomToolMissingRequirements struct {
	OAuthProviders       []string `json:"oauth_providers,omitempty"`
	EnvironmentVariables []string `json:"environment_variables,omitempty"`
	Secrets              []string `json:"secrets,omitempty"`
}

type ValidateCustomToolResponseData struct {
	IsValid                bool                                  `json:"is_valid"`
	IsReady                bool                                  `json:"is_ready"`
	MissingRequirements    ValidateCustomToolMissingRequirements `json:"missing_requirements"`
	ConfiguredRequirements ValidateCustomToolMissingRequirements `json:"configured_requirements"`
	MissingSecrets         []string                              `json:"missing_secrets,omitempty"`
	NextSteps              []string                              `json:"next_steps,omitempty"`
	Message                string                                `json:"message,omitempty"`
}

type ValidateCustomToolResponse struct {
	Success bool                           `json:"success"`
	Data    ValidateCustomToolResponseData `json:"data"`
}

type RunCustomToolRequest struct {
	InputVars map[string]interface{} `json:"input_vars,omitempty"`
}

type RunCustomToolResponseData struct {
	RunUUID       string `json:"run_uuid,omitempty"`
	ExecutionUUID string `json:"execution_uuid,omitempty"`
	Status        string `json:"status,omitempty"`
	Message       string `json:"message,omitempty"`
}

type RunCustomToolResponse struct {
	Success bool                      `json:"success"`
	Data    RunCustomToolResponseData `json:"data"`
}

// Error response

type ErrorResponse struct {
	Error string `json:"error"`
}
