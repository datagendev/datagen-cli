package prompts

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/datagendev/datagen-cli/internal/config"
)

// CollectServiceConfig interactively collects configuration for a service
func CollectServiceConfig() (*config.Service, error) {
	svc := &config.Service{
		InputSchema: config.Schema{Fields: []config.Field{}},
	}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name (lowercase, no spaces):",
		Help:    "E.g., enrichment, chat, generate",
	}, &svc.Name, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Endpoint type selection
	endpointType := ""
	if err := survey.AskOne(&survey.Select{
		Message: "What type of endpoint do you want to create?",
		Options: []string{"webhook", "api", "streaming"},
		Description: func(value string, index int) string {
			switch value {
			case "webhook":
				return "Async background processing (fire-and-forget)"
			case "api":
				return "Synchronous API call (returns result)"
			case "streaming":
				return "Server-sent events (SSE) streaming"
			default:
				return ""
			}
		},
	}, &endpointType, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	svc.Type = endpointType

	// Path based on type
	var path string
	if endpointType == "webhook" {
		if err := survey.AskOne(&survey.Input{
			Message: "Webhook path:",
			Default: fmt.Sprintf("/webhook/%s", svc.Name),
			Help:    "E.g., /webhook/signup",
		}, &path, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
		svc.WebhookPath = path
	} else {
		if err := survey.AskOne(&survey.Input{
			Message: "API path:",
			Default: fmt.Sprintf("/api/%s", svc.Name),
			Help:    "E.g., /api/chat or /stream/generate",
		}, &path, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
		svc.APIPath = path
	}

	// Description
	if err := survey.AskOne(&survey.Input{
		Message: "Description:",
		Help:    "Brief description of what this endpoint does",
	}, &svc.Description, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Prompt file path
	if err := survey.AskOne(&survey.Input{
		Message: "Agent prompt file path:",
		Default: fmt.Sprintf(".claude/agents/%s.md", svc.Name),
		Help:    "Path to the agent markdown file",
	}, &svc.Prompt, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Input schema fields
	fmt.Println("\n📋 Define input schema fields (press Enter with empty name to finish):")
	if err := collectSchemaFields(&svc.InputSchema); err != nil {
		return nil, err
	}

	// Output schema fields (only for API endpoints)
	if endpointType == "api" {
		addOutput := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Define output schema?",
			Default: true,
			Help:    "Specify the structure of the response data",
		}, &addOutput); err != nil {
			return nil, err
		}

		if addOutput {
			svc.OutputSchema = &config.Schema{Fields: []config.Field{}}
			fmt.Println("\n📤 Define output schema fields (press Enter with empty name to finish):")
			if err := collectSchemaFields(svc.OutputSchema); err != nil {
				return nil, err
			}
		}
	}

	// Allowed tools
	if err := collectAllowedTools(&svc.AllowedTools); err != nil {
		return nil, err
	}

	// Type-specific configuration
	switch endpointType {
	case "webhook":
		if err := collectWebhookConfig(svc); err != nil {
			return nil, err
		}
	case "api":
		if err := collectAPIConfig(svc); err != nil {
			return nil, err
		}
	case "streaming":
		if err := collectStreamingConfig(svc); err != nil {
			return nil, err
		}
	}

	// Auth configuration
	if err := collectAuthConfig(svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func collectSchemaFields(schema *config.Schema) error {
	for {
		var fieldName string
		if err := survey.AskOne(&survey.Input{
			Message: "Field name (or press Enter to finish):",
		}, &fieldName); err != nil {
			return err
		}

		if fieldName == "" {
			break
		}

		field := config.Field{Name: fieldName}

		// Field type
		if err := survey.AskOne(&survey.Select{
			Message: fmt.Sprintf("Type for '%s':", fieldName),
			Options: []string{"str", "int", "float", "bool", "list", "dict", "any"},
			Default: "str",
		}, &field.Type); err != nil {
			return err
		}

		// Required?
		if err := survey.AskOne(&survey.Confirm{
			Message: "Required?",
			Default: true,
		}, &field.Required); err != nil {
			return err
		}

		// Default value (optional)
		var defaultVal string
		if err := survey.AskOne(&survey.Input{
			Message: "Default value (optional, press Enter to skip):",
		}, &defaultVal); err != nil {
			return err
		}
		if defaultVal != "" {
			field.Default = defaultVal
		}

		schema.Fields = append(schema.Fields, field)
	}

	return nil
}

func collectAllowedTools(tools *config.AllowedTools) error {
	selected := []string{}
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Select allowed DataGen tools:",
		Options: []string{"searchTools", "executeTools", "executeCode", "getToolDetails"},
		Default: []string{"executeTools", "getToolDetails"},
	}, &selected); err != nil {
		return err
	}

	for _, tool := range selected {
		switch tool {
		case "searchTools":
			tools.SearchTools = true
		case "executeTools":
			tools.ExecuteTools = true
		case "executeCode":
			tools.ExecuteCode = true
		case "getToolDetails":
			tools.GetToolDetails = true
		}
	}

	return nil
}

func collectWebhookConfig(svc *config.Service) error {
	svc.Webhook = &config.WebhookConfig{}

	// Signature verification
	var sigType string
	if err := survey.AskOne(&survey.Select{
		Message: "Signature verification method:",
		Options: []string{"hmac_sha256", "custom", "none"},
		Default: "hmac_sha256",
		Help:    "How to verify webhook authenticity",
	}, &sigType); err != nil {
		return err
	}
	svc.Webhook.SignatureVerification = sigType

	if sigType == "hmac_sha256" {
		if err := survey.AskOne(&survey.Input{
			Message: "Signature header name:",
			Default: "X-Signature",
		}, &svc.Webhook.SignatureHeader); err != nil {
			return err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "Secret environment variable name:",
			Default: "HMAC_SECRET",
		}, &svc.Webhook.SecretEnv); err != nil {
			return err
		}
	}

	// Retry policy
	if err := survey.AskOne(&survey.Confirm{
		Message: "Enable retry policy?",
		Default: false,
	}, &svc.Webhook.RetryEnabled); err != nil {
		return err
	}

	if svc.Webhook.RetryEnabled {
		if err := survey.AskOne(&survey.Input{
			Message: "Max retries:",
			Default: "3",
		}, &svc.Webhook.MaxRetries); err != nil {
			return err
		}

		var strategy string
		if err := survey.AskOne(&survey.Select{
			Message: "Backoff strategy:",
			Options: []string{"exponential", "linear"},
			Default: "exponential",
		}, &strategy); err != nil {
			return err
		}
		svc.Webhook.BackoffStrategy = strategy
	}

	return nil
}

func collectAPIConfig(svc *config.Service) error {
	svc.API = &config.APIConfig{}

	// Response format
	if err := survey.AskOne(&survey.Select{
		Message: "Response format:",
		Options: []string{"json", "text", "custom"},
		Default: "json",
	}, &svc.API.ResponseFormat); err != nil {
		return err
	}

	// Timeout
	var timeoutStr string
	if err := survey.AskOne(&survey.Input{
		Message: "Timeout (seconds):",
		Default: "30",
	}, &timeoutStr); err != nil {
		return err
	}
	fmt.Sscanf(timeoutStr, "%d", &svc.API.Timeout)

	// Rate limiting
	if err := survey.AskOne(&survey.Confirm{
		Message: "Enable rate limiting?",
		Default: false,
	}, &svc.API.RateLimitEnabled); err != nil {
		return err
	}

	if svc.API.RateLimitEnabled {
		var rpmStr string
		if err := survey.AskOne(&survey.Input{
			Message: "Requests per minute:",
			Default: "60",
		}, &rpmStr); err != nil {
			return err
		}
		fmt.Sscanf(rpmStr, "%d", &svc.API.RateLimitRPM)
	}

	return nil
}

func collectStreamingConfig(svc *config.Service) error {
	svc.Streaming = &config.StreamingConfig{}

	// Format
	if err := survey.AskOne(&survey.Select{
		Message: "SSE format:",
		Options: []string{"default", "json", "custom"},
		Default: "default",
	}, &svc.Streaming.Format); err != nil {
		return err
	}

	// Buffer size
	var bufferStr string
	if err := survey.AskOne(&survey.Input{
		Message: "Buffer size (bytes):",
		Default: "8192",
	}, &bufferStr); err != nil {
		return err
	}
	fmt.Sscanf(bufferStr, "%d", &svc.Streaming.BufferSize)

	return nil
}

func collectAuthConfig(svc *config.Service) error {
	var authType string
	if err := survey.AskOne(&survey.Select{
		Message: "Authentication method:",
		Options: []string{"api_key", "bearer_token", "oauth", "none"},
		Default: "api_key",
	}, &authType); err != nil {
		return err
	}

	if authType == "none" {
		return nil
	}

	svc.Auth = &config.Auth{Type: authType}

	// Header name
	defaultHeader := "X-API-Key"
	if authType == "bearer_token" {
		defaultHeader = "Authorization"
	}
	if err := survey.AskOne(&survey.Input{
		Message: "Header name:",
		Default: defaultHeader,
	}, &svc.Auth.Header); err != nil {
		return err
	}

	// Environment variable
	defaultEnv := strings.ToUpper(svc.Name) + "_API_KEY"
	if err := survey.AskOne(&survey.Input{
		Message: "Environment variable name:",
		Default: defaultEnv,
	}, &svc.Auth.EnvVar); err != nil {
		return err
	}

	return nil
}

// CollectRootConfig collects the root configuration (API keys, etc.)
func CollectRootConfig() (string, string, error) {
	var datagenKey, claudeKey string

	if err := survey.AskOne(&survey.Input{
		Message: "DataGen API key environment variable name:",
		Default: "DATAGEN_API_KEY",
	}, &datagenKey, survey.WithValidator(survey.Required)); err != nil {
		return "", "", err
	}

	// Claude API key env - hardcoded to match Anthropic SDK convention
	claudeKey = "ANTHROPIC_API_KEY"

	return datagenKey, claudeKey, nil
}
