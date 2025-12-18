package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/datagendev/datagen-cli/internal/agents"
	"github.com/datagendev/datagen-cli/internal/config"
	"github.com/datagendev/datagen-cli/internal/prompts"
	"github.com/spf13/cobra"
)

var startOutputDir string
var startAdvanced bool
var startAgent string
var startMode string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Project setup",
	Long:  `Start a new DataGen project (defaults-first) from an existing .claude/agents/*.md agent, or use --advanced for the full interactive flow.`,
	Run:   runStart,
}

func init() {
	startCmd.Flags().StringVarP(&startOutputDir, "output", "o", ".", "Output directory for project configuration")
	startCmd.MarkFlagDirname("output")
	startCmd.Flags().BoolVar(&startAdvanced, "advanced", false, "Use the full interactive flow to create services and agent files")
	startCmd.Flags().StringVar(&startAgent, "agent", "", "Agent to deploy (agent name or filename under .claude/agents)")
	startCmd.Flags().StringVar(&startMode, "mode", "", "Deployment mode: webhook or api")
}

func runStart(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Welcome to DataGen CLI!")
	fmt.Println("Let's set up your agent project.")
	fmt.Println()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(startOutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if startAdvanced {
		runStartAdvanced()
		return
	}

	if err := runStartFromExistingAgents(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Tip: create an agent file under %s or run 'datagen start --advanced'\n", filepath.Join(".claude", "agents"))
		os.Exit(1)
	}
}

func runStartAdvanced() {
	// Collect root configuration
	datagenKey, claudeKey, err := prompts.CollectRootConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := &config.DatagenConfig{
		DatagenAPIKeyEnv: datagenKey,
		ClaudeAPIKeyEnv:  claudeKey,
		Services:         []config.Service{},
	}

	// Collect services
	for {
		fmt.Println("\n📦 Configure a service:")
		svc, err := prompts.CollectServiceConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		cfg.Services = append(cfg.Services, *svc)

		// Ask if user wants to add another service
		addAnother := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Add another service?",
			Default: false,
		}, &addAnother); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if !addAnother {
			break
		}
	}

	// Create .claude/agents directory
	agentsDir := filepath.Join(startOutputDir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agents directory: %v\n", err)
		os.Exit(1)
	}

	// Create agent prompt files for each service
	fmt.Println("\n📝 Creating agent prompt files...")
	for _, svc := range cfg.Services {
		if err := createAgentPromptFile(startOutputDir, &svc); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create prompt file for %s: %v\n", svc.Name, err)
		} else {
			fmt.Printf("  ✓ Created %s\n", svc.Prompt)
		}
	}

	// Save configuration to output directory
	configPath := filepath.Join(startOutputDir, "datagen.toml")
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(configPath)
	fmt.Printf("\n✅ Configuration saved to %s\n", absPath)
	fmt.Println("\n📝 Next steps:")
	if startOutputDir != "." {
		fmt.Printf("  1. cd %s\n", startOutputDir)
		fmt.Println("  2. Review and edit datagen.toml if needed")
		fmt.Println("  3. Customize your agent prompt files in .claude/agents/")
		fmt.Println("  4. Run 'datagen build' to generate the boilerplate code")
		fmt.Println("  5. Test locally, then run 'datagen deploy railway' to deploy")
	} else {
		fmt.Println("  1. Review and edit datagen.toml if needed")
		fmt.Println("  2. Customize your agent prompt files in .claude/agents/")
		fmt.Println("  3. Run 'datagen build' to generate the boilerplate code")
		fmt.Println("  4. Test locally, then run 'datagen deploy railway' to deploy")
	}
}

func runStartFromExistingAgents() error {
	sourceAgentsDir := filepath.Join(".claude", "agents")
	if _, err := os.Stat(sourceAgentsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no .claude agents directory found in current directory: %s", sourceAgentsDir)
		}
		return err
	}

	found, err := agents.Discover(sourceAgentsDir)
	if err != nil {
		return err
	}

	selectable := make([]agents.Agent, 0, len(found))
	for _, a := range found {
		if a.Kind == agents.KindDatagenOnly || a.Kind == agents.KindNoMCP {
			selectable = append(selectable, a)
		}
	}
	if len(selectable) == 0 {
		return fmt.Errorf("no selectable agents found in %s (only 'tools: [datagen]' or no tools are supported)", sourceAgentsDir)
	}

	sort.Slice(selectable, func(i, j int) bool {
		return strings.ToLower(filepath.Base(selectable[i].Path)) < strings.ToLower(filepath.Base(selectable[j].Path))
	})

	selected, err := chooseAgent(selectable, startAgent)
	if err != nil {
		return err
	}

	mode, err := chooseMode(startMode)
	if err != nil {
		return err
	}

	// Root configuration defaults to env var names.
	datagenKey, claudeKey, err := prompts.CollectRootConfig()
	if err != nil {
		return err
	}

	// Service config derived from the agent file.
	rawName := selected.Name
	if rawName == "" {
		rawName = strings.TrimSuffix(filepath.Base(selected.Path), filepath.Ext(selected.Path))
	}
	serviceName := config.NormalizeServiceName(rawName)

	description := strings.TrimSpace(selected.Description)
	if description == "" {
		description = fmt.Sprintf("Deploy agent %s", rawName)
	}

	promptRel := filepath.ToSlash(filepath.Join(".claude", "agents", filepath.Base(selected.Path)))

	// Ensure the selected agent exists in the output directory (copy when --output != ".").
	destAgentsDir := filepath.Join(startOutputDir, ".claude", "agents")
	if err := os.MkdirAll(destAgentsDir, 0755); err != nil {
		return fmt.Errorf("create agents dir: %w", err)
	}
	destAgentPath := filepath.Join(destAgentsDir, filepath.Base(selected.Path))
	if !samePath(selected.Path, destAgentPath) {
		if _, err := os.Stat(destAgentPath); err == nil {
			return fmt.Errorf("agent file already exists in output dir: %s", destAgentPath)
		}
		if err := copyFile(selected.Path, destAgentPath); err != nil {
			return fmt.Errorf("copy agent to output dir: %w", err)
		}
	}

	svc := config.Service{
		Name:        serviceName,
		Type:        mode,
		Description: description,
		Prompt:      promptRel,
		InputSchema: config.Schema{Fields: []config.Field{}},
		Auth: &config.Auth{
			Type:   "api_key",
			Header: "X-API-Key",
			EnvVar: config.NormalizeEnvVarName(serviceName) + "_API_KEY",
		},
	}

	if selected.Kind == agents.KindDatagenOnly {
		svc.AllowedTools = config.AllowedTools{
			ExecuteTools:   true,
			GetToolDetails: true,
		}
	}

	switch mode {
	case "webhook":
		svc.WebhookPath = fmt.Sprintf("/webhook/%s", serviceName)
		svc.Webhook = &config.WebhookConfig{
			SignatureVerification: "hmac_sha256",
			SignatureHeader:       "X-Signature",
			SecretEnv:             config.NormalizeEnvVarName(serviceName) + "_HMAC_SECRET",
			RetryEnabled:          false,
		}
	case "api":
		svc.APIPath = fmt.Sprintf("/api/%s", serviceName)
		svc.API = &config.APIConfig{
			ResponseFormat:   "json",
			Timeout:          30,
			RateLimitEnabled: false,
		}
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}

	cfg := &config.DatagenConfig{
		DatagenAPIKeyEnv: datagenKey,
		ClaudeAPIKeyEnv:  claudeKey,
		Services:         []config.Service{svc},
	}

	configPath := filepath.Join(startOutputDir, "datagen.toml")
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	absPath, _ := filepath.Abs(configPath)
	fmt.Printf("\n✅ Configuration saved to %s\n", absPath)
	fmt.Println("\n📝 Next steps:")
	if startOutputDir != "." {
		fmt.Printf("  1. cd %s\n", startOutputDir)
		fmt.Println("  2. Review and edit datagen.toml if needed")
		fmt.Println("  3. Run 'datagen build' to generate the boilerplate code")
		fmt.Println("  4. Test locally, then run 'datagen deploy railway' to deploy")
	} else {
		fmt.Println("  1. Review and edit datagen.toml if needed")
		fmt.Println("  2. Run 'datagen build' to generate the boilerplate code")
		fmt.Println("  3. Test locally, then run 'datagen deploy railway' to deploy")
	}

	return nil
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return aa == bb
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func chooseMode(flagValue string) (string, error) {
	if flagValue != "" {
		switch flagValue {
		case "api", "webhook":
			return flagValue, nil
		default:
			return "", fmt.Errorf("invalid --mode %q (expected 'api' or 'webhook')", flagValue)
		}
	}

	var mode string
	if err := survey.AskOne(&survey.Select{
		Message: "Deploy this agent as:",
		Options: []string{"api", "webhook"},
		Default: "api",
		Description: func(value string, index int) string {
			switch value {
			case "api":
				return "Synchronous endpoint (returns result)"
			case "webhook":
				return "Async background processing (fire-and-forget)"
			default:
				return ""
			}
		},
	}, &mode, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return mode, nil
}

func chooseAgent(selectable []agents.Agent, flagValue string) (agents.Agent, error) {
	if flagValue != "" {
		matches := make([]agents.Agent, 0, 2)
		for _, a := range selectable {
			base := filepath.Base(a.Path)
			stem := strings.TrimSuffix(base, filepath.Ext(base))
			if strings.EqualFold(a.Name, flagValue) || strings.EqualFold(base, flagValue) || strings.EqualFold(stem, flagValue) {
				matches = append(matches, a)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) == 0 {
			return agents.Agent{}, fmt.Errorf("no agent matches --agent %q", flagValue)
		}
		return agents.Agent{}, fmt.Errorf("multiple agents match --agent %q; use the full filename", flagValue)
	}

	options := make([]string, 0, len(selectable))
	byOption := map[string]agents.Agent{}
	for _, a := range selectable {
		opt := fmt.Sprintf("%s (%s)", a.Name, filepath.Base(a.Path))
		options = append(options, opt)
		byOption[opt] = a
	}

	var picked string
	if err := survey.AskOne(&survey.Select{
		Message: "Select an agent to deploy:",
		Options: options,
		Description: func(value string, index int) string {
			a := byOption[value]
			desc := strings.TrimSpace(a.Description)
			if desc == "" {
				desc = "No description"
			}
			switch a.Kind {
			case agents.KindDatagenOnly:
				return desc + " (datagen MCP only)"
			case agents.KindNoMCP:
				return desc + " (no MCP)"
			default:
				return desc
			}
		},
	}, &picked, survey.WithValidator(survey.Required)); err != nil {
		return agents.Agent{}, err
	}

	a, ok := byOption[picked]
	if !ok {
		return agents.Agent{}, fmt.Errorf("internal error: selected option not found")
	}
	return a, nil
}

func createAgentPromptFile(outputDir string, svc *config.Service) error {
	// Construct full file path
	promptPath := filepath.Join(outputDir, svc.Prompt)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(promptPath), 0755); err != nil {
		return fmt.Errorf("failed to create prompt directory: %w", err)
	}

	// Generate template content based on service type
	var template string
	switch svc.Type {
	case "webhook":
		template = generateWebhookTemplate(svc)
	case "streaming":
		template = generateStreamingTemplate(svc)
	default: // "api"
		template = generateAPITemplate(svc)
	}

	// Write the template to the file
	if err := os.WriteFile(promptPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	return nil
}

func generateAPITemplate(svc *config.Service) string {
	return fmt.Sprintf(`---
name: %s
description: %s
tools:
  - datagen
model: claude-sonnet-4
---

# %s

You are an AI assistant for the %s service. Your role is to help process requests and provide responses using the available tools.

## Available Tools

- **datagen**: Access to DataGen MCP tools for data processing and integration

## Instructions

1. Analyze the user's request carefully
2. Use the available tools to gather necessary information
3. Process the data and formulate a response
4. Return results in the expected output format

## Output Format

Ensure your responses match the expected output schema defined in the service configuration.

## Notes

- Customize this prompt to match your specific use case
- Add specific instructions for how the agent should behave
- Include examples if helpful for your workflow
`, svc.Name, svc.Description, svc.Name, svc.Name)
}

func generateWebhookTemplate(svc *config.Service) string {
	return fmt.Sprintf(`---
name: %s
description: %s
tools:
  - datagen
model: claude-sonnet-4
---

# %s (Webhook)

You are an AI assistant for the %s webhook service. This service receives webhook events and processes them asynchronously.

## Available Tools

- **datagen**: Access to DataGen MCP tools for data processing and integration

## Instructions

1. Parse the incoming webhook payload
2. Validate the webhook signature if configured
3. Process the event data using available tools
4. Perform any necessary actions or transformations
5. Return status information about the processing

## Webhook Handling

- This is an asynchronous service - responses are sent immediately
- Long-running processing happens in the background
- Use appropriate error handling for webhook failures

## Output Format

Return a status object indicating whether the webhook was accepted and queued for processing.

## Notes

- Customize this prompt for your specific webhook use case
- Add validation logic specific to your webhook source
- Include any security considerations
`, svc.Name, svc.Description, svc.Name, svc.Name)
}

func generateStreamingTemplate(svc *config.Service) string {
	return fmt.Sprintf(`---
name: %s
description: %s
tools:
  - datagen
model: claude-sonnet-4
---

# %s (Streaming)

You are an AI assistant for the %s streaming service. This service provides real-time streaming responses.

## Available Tools

- **datagen**: Access to DataGen MCP tools for data processing and integration

## Instructions

1. Process incoming requests and begin streaming responses
2. Use available tools to gather information as needed
3. Stream partial results as they become available
4. Provide continuous updates until processing is complete

## Streaming Behavior

- Start responding immediately with available information
- Stream updates as new data becomes available
- Indicate completion when all processing is done
- Handle interruptions gracefully

## Output Format

Stream responses in the expected output schema, sending partial updates incrementally.

## Notes

- Customize this prompt for your specific streaming use case
- Consider user experience for real-time updates
- Balance between response speed and accuracy
`, svc.Name, svc.Description, svc.Name, svc.Name)
}
