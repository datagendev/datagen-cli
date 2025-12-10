package cmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/datagendev/datagen-cli/internal/config"
	"github.com/datagendev/datagen-cli/internal/prompts"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Interactive project setup",
	Long:  `Start a new DataGen project with interactive prompts`,
	Run:   runStart,
}

func runStart(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Welcome to DataGen CLI!")
	fmt.Println("Let's set up your agent project.\n")

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

	// Save configuration
	configPath := "datagen.toml"
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Configuration saved to datagen.toml")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Review and edit datagen.toml if needed")
	fmt.Println("  2. Create your agent prompt files in .claude/agents/")
	fmt.Println("  3. Run 'datagen build' to generate the boilerplate code")
	fmt.Println("  4. Test locally, then run 'datagen deploy railway' to deploy")
}
