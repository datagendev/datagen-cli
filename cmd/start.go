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
		fmt.Println("  3. Create your agent prompt files in .claude/agents/")
		fmt.Println("  4. Run 'datagen build' to generate the boilerplate code")
		fmt.Println("  5. Test locally, then run 'datagen deploy railway' to deploy")
	} else {
		fmt.Println("  1. Review and edit datagen.toml if needed")
		fmt.Println("  2. Create your agent prompt files in .claude/agents/")
		fmt.Println("  3. Run 'datagen build' to generate the boilerplate code")
		fmt.Println("  4. Test locally, then run 'datagen deploy railway' to deploy")
	}
}
