package config

// DatagenConfig represents the full datagen.toml configuration
type DatagenConfig struct {
	DatagenAPIKeyEnv string    `toml:"datagen_api_key_env"`
	ClaudeAPIKeyEnv  string    `toml:"claude_api_key_env"`
	Services         []Service `toml:"service"`
}

// Service represents a single service/endpoint configuration
type Service struct {
	Name         string       `toml:"name"`
	Type         string       `toml:"type"` // webhook, api, streaming
	Description  string       `toml:"description"`
	Prompt       string       `toml:"prompt"`
	AllowedTools AllowedTools `toml:"allowed_tools"`
	InputSchema  Schema       `toml:"input_schema"`
	OutputSchema *Schema      `toml:"output_schema,omitempty"` // Only for API endpoints
	Auth         *Auth        `toml:"auth,omitempty"`

	// Type-specific configurations
	Webhook   *WebhookConfig   `toml:"webhook,omitempty"`
	API       *APIConfig       `toml:"api,omitempty"`
	Streaming *StreamingConfig `toml:"streaming,omitempty"`

	// Paths (mutually exclusive based on type)
	WebhookPath string `toml:"webhook_path,omitempty"`
	APIPath     string `toml:"api_path,omitempty"`
}

// AllowedTools defines which DataGen tools the agent can use
type AllowedTools struct {
	SearchTools    bool `toml:"searchTools"`
	ExecuteTools   bool `toml:"executeTools"`
	ExecuteCode    bool `toml:"executeCode"`
	GetToolDetails bool `toml:"getToolDetails"`
}

// Schema defines input or output data structure
type Schema struct {
	Name   string  `toml:"name,omitempty"`
	Fields []Field `toml:"fields"`
}

// Field represents a single field in a schema
type Field struct {
	Name     string `toml:"name"`
	Type     string `toml:"type"` // str, int, float, bool, list, dict
	Required bool   `toml:"required"`
	Default  string `toml:"default,omitempty"`
}

// Auth defines authentication configuration
type Auth struct {
	Type   string `toml:"type"` // api_key, bearer_token, oauth, none
	Header string `toml:"header,omitempty"`
	EnvVar string `toml:"env_var,omitempty"`
}

// WebhookConfig contains webhook-specific configuration
type WebhookConfig struct {
	SignatureVerification string `toml:"signature_verification,omitempty"` // hmac_sha256, custom, none
	SignatureHeader       string `toml:"signature_header,omitempty"`
	SecretEnv             string `toml:"secret_env,omitempty"`
	RetryEnabled          bool   `toml:"retry_enabled"`
	MaxRetries            int    `toml:"max_retries,omitempty"`
	BackoffStrategy       string `toml:"backoff_strategy,omitempty"` // exponential, linear
}

// APIConfig contains API-specific configuration
type APIConfig struct {
	ResponseFormat   string `toml:"response_format"`     // json, text, custom
	Timeout          int    `toml:"timeout"`             // seconds
	RateLimitEnabled bool   `toml:"rate_limit_enabled"`
	RateLimitRPM     int    `toml:"rate_limit_rpm,omitempty"` // requests per minute
}

// StreamingConfig contains streaming-specific configuration
type StreamingConfig struct {
	Format     string `toml:"format"`      // default, json, custom
	BufferSize int    `toml:"buffer_size"` // bytes
}

// GetPath returns the appropriate path based on endpoint type
func (s *Service) GetPath() string {
	switch s.Type {
	case "webhook":
		return s.WebhookPath
	case "api", "streaming":
		return s.APIPath
	default:
		return ""
	}
}

// GetFunctionName returns a snake_case function name from the service name
func (s *Service) GetFunctionName() string {
	// Simple snake_case conversion - you can enhance this
	return s.Name + "_handler"
}

// GetModelName returns the Pydantic model name for input
func (s *Service) GetInputModelName() string {
	return toPascalCase(s.Name) + "Input"
}

// GetOutputModelName returns the Pydantic model name for output
func (s *Service) GetOutputModelName() string {
	return toPascalCase(s.Name) + "Output"
}

// GetTaskName returns the background task function name
func (s *Service) GetTaskName() string {
	return s.Name + "_task"
}

// Helper function to convert to PascalCase
func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	// Simple implementation - capitalize first letter
	// In a real implementation, you'd want proper snake_case -> PascalCase conversion
	result := []rune(s)
	result[0] = toUpper(result[0])
	return string(result)
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}
