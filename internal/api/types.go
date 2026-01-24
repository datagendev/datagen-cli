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
}

type ListExecutionsResponse struct {
	Executions []Execution `json:"executions"`
}

// Error response

type ErrorResponse struct {
	Error string `json:"error"`
}
