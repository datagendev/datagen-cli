package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/datagendev/datagen-cli/internal/config"
	"github.com/datagendev/datagen-cli/internal/prompts"
	"github.com/spf13/cobra"
)

var startOutputDir string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Interactive project setup",
	Long:  `Start a new DataGen project with interactive prompts`,
	Run:   runStart,
}

func init() {
	startCmd.Flags().StringVarP(&startOutputDir, "output", "o", ".", "Output directory for project configuration")
	startCmd.MarkFlagDirname("output")
}

func runStart(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Welcome to DataGen CLI!")
	fmt.Println("Let's set up your agent project.\n")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(startOutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

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
