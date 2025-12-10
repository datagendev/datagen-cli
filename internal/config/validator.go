package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateConfig checks if the configuration is valid
func ValidateConfig(cfg *DatagenConfig, configDir string) error {
	// Check required API key env vars
	if cfg.DatagenAPIKeyEnv == "" {
		return fmt.Errorf("datagen_api_key_env is required")
	}
	if cfg.ClaudeAPIKeyEnv == "" {
		return fmt.Errorf("claude_api_key_env is required")
	}

	// Check that at least one service is defined
	if len(cfg.Services) == 0 {
		return fmt.Errorf("at least one service must be defined")
	}

	// Validate each service
	for i, svc := range cfg.Services {
		if err := validateService(&svc, i, configDir); err != nil {
			return fmt.Errorf("service[%d] (%s): %w", i, svc.Name, err)
		}
	}

	return nil
}

func validateService(svc *Service, index int, configDir string) error {
	// Check required fields
	if svc.Name == "" {
		return fmt.Errorf("name is required")
	}
	if svc.Type == "" {
		return fmt.Errorf("type is required")
	}
	if svc.Description == "" {
		return fmt.Errorf("description is required")
	}
	if svc.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	// Validate type
	validTypes := map[string]bool{"webhook": true, "api": true, "streaming": true}
	if !validTypes[svc.Type] {
		return fmt.Errorf("invalid type '%s', must be one of: webhook, api, streaming", svc.Type)
	}

	// Check that prompt file exists (resolve relative to config directory)
	promptPath := svc.Prompt
	if !filepath.IsAbs(promptPath) {
		promptPath = filepath.Join(configDir, promptPath)
	}
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		return fmt.Errorf("prompt file not found: %s", svc.Prompt)
	}

	// Validate paths based on type
	switch svc.Type {
	case "webhook":
		if svc.WebhookPath == "" {
			return fmt.Errorf("webhook_path is required for webhook type")
		}
		if !strings.HasPrefix(svc.WebhookPath, "/") {
			return fmt.Errorf("webhook_path must start with /")
		}
		if svc.Webhook != nil {
			if err := validateWebhookConfig(svc.Webhook); err != nil {
				return fmt.Errorf("webhook config: %w", err)
			}
		}
	case "api":
		if svc.APIPath == "" {
			return fmt.Errorf("api_path is required for api type")
		}
		if !strings.HasPrefix(svc.APIPath, "/") {
			return fmt.Errorf("api_path must start with /")
		}
		if svc.API != nil {
			if err := validateAPIConfig(svc.API); err != nil {
				return fmt.Errorf("api config: %w", err)
			}
		}
	case "streaming":
		if svc.APIPath == "" {
			return fmt.Errorf("api_path is required for streaming type")
		}
		if !strings.HasPrefix(svc.APIPath, "/") {
			return fmt.Errorf("api_path must start with /")
		}
		if svc.Streaming != nil {
			if err := validateStreamingConfig(svc.Streaming); err != nil {
				return fmt.Errorf("streaming config: %w", err)
			}
		}
	}

	// Validate input schema fields (if any)
	for _, field := range svc.InputSchema.Fields {
		if err := validateField(&field); err != nil {
			return fmt.Errorf("input_schema field '%s': %w", field.Name, err)
		}
	}

	// Validate output schema for API endpoints
	if svc.Type == "api" && svc.OutputSchema != nil && len(svc.OutputSchema.Fields) > 0 {
		for _, field := range svc.OutputSchema.Fields {
			if err := validateField(&field); err != nil {
				return fmt.Errorf("output_schema field '%s': %w", field.Name, err)
			}
		}
	}

	// Validate auth if present
	if svc.Auth != nil {
		if err := validateAuth(svc.Auth); err != nil {
			return fmt.Errorf("auth config: %w", err)
		}
	}

	return nil
}

func validateField(field *Field) error {
	if field.Name == "" {
		return fmt.Errorf("field name is required")
	}
	validTypes := map[string]bool{
		"str": true, "int": true, "float": true, "bool": true,
		"list": true, "dict": true, "any": true,
	}
	if !validTypes[field.Type] {
		return fmt.Errorf("invalid type '%s', must be one of: str, int, float, bool, list, dict, any", field.Type)
	}
	return nil
}

func validateAuth(auth *Auth) error {
	validTypes := map[string]bool{"api_key": true, "bearer_token": true, "oauth": true, "none": true}
	if !validTypes[auth.Type] {
		return fmt.Errorf("invalid auth type '%s', must be one of: api_key, bearer_token, oauth, none", auth.Type)
	}
	if auth.Type != "none" && auth.EnvVar == "" {
		return fmt.Errorf("env_var is required when auth type is not 'none'")
	}
	return nil
}

func validateWebhookConfig(wh *WebhookConfig) error {
	if wh.SignatureVerification != "" {
		validTypes := map[string]bool{"hmac_sha256": true, "custom": true, "none": true}
		if !validTypes[wh.SignatureVerification] {
			return fmt.Errorf("invalid signature_verification '%s'", wh.SignatureVerification)
		}
		if wh.SignatureVerification == "hmac_sha256" {
			if wh.SignatureHeader == "" {
				return fmt.Errorf("signature_header is required for hmac_sha256")
			}
			if wh.SecretEnv == "" {
				return fmt.Errorf("secret_env is required for hmac_sha256")
			}
		}
	}
	if wh.RetryEnabled && wh.MaxRetries <= 0 {
		return fmt.Errorf("max_retries must be > 0 when retry_enabled is true")
	}
	return nil
}

func validateAPIConfig(api *APIConfig) error {
	if api.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}
	if api.RateLimitEnabled && api.RateLimitRPM <= 0 {
		return fmt.Errorf("rate_limit_rpm must be > 0 when rate_limit_enabled is true")
	}
	validFormats := map[string]bool{"json": true, "text": true, "custom": true}
	if !validFormats[api.ResponseFormat] {
		return fmt.Errorf("invalid response_format '%s'", api.ResponseFormat)
	}
	return nil
}

func validateStreamingConfig(stream *StreamingConfig) error {
	validFormats := map[string]bool{"default": true, "json": true, "custom": true}
	if !validFormats[stream.Format] {
		return fmt.Errorf("invalid format '%s'", stream.Format)
	}
	if stream.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be > 0")
	}
	return nil
}
