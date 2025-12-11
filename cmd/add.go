package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/datagendev/datagen-cli/internal/codegen"
	"github.com/datagendev/datagen-cli/internal/config"
	"github.com/datagendev/datagen-cli/internal/prompts"
	"github.com/spf13/cobra"
)

var (
	addOutputDir   string
	addConfigPath  string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new service to an existing project",
	Long: `Interactively add a new service (endpoint) to an existing DataGen project.
This command will update the configuration and inject new code into existing files
without overwriting user customizations.`,
	Run: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addOutputDir, "output", "o", ".", "Project directory")
	addCmd.Flags().StringVarP(&addConfigPath, "config", "c", "datagen.toml", "Path to datagen.toml configuration file")
	addCmd.MarkFlagDirname("output")
	addCmd.MarkFlagFilename("config", "toml")
}

func runAdd(cmd *cobra.Command, args []string) {
	fmt.Println("➕ Adding a new service to your project...")

	// Load existing configuration
	cfg, err := config.LoadConfig(addConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Println("\nMake sure you run this command from your project directory,")
		fmt.Println("or use --config to specify the path to datagen.toml")
		os.Exit(1)
	}

	fmt.Printf("✓ Loaded configuration with %d existing service(s)\n", len(cfg.Services))

	// Collect new service configuration
	fmt.Println("\n📦 Configure new service:")
	newService, err := prompts.CollectServiceConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check for duplicate service names
	for _, svc := range cfg.Services {
		if svc.Name == newService.Name {
			fmt.Fprintf(os.Stderr, "Error: Service '%s' already exists\n", newService.Name)
			os.Exit(1)
		}
	}

	// Add service to configuration
	cfg.Services = append(cfg.Services, *newService)

	// Save updated configuration
	if err := config.SaveConfig(cfg, addConfigPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Configuration updated")

	// Create agent prompt file
	fmt.Println("\n📝 Creating agent prompt file...")
	if err := createAgentPromptFile(addOutputDir, newService); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create prompt file: %v\n", err)
		fmt.Println("You may need to create it manually.")
	} else {
		fmt.Printf("  ✓ Created %s\n", newService.Prompt)
	}

	// Update existing code files incrementally
	fmt.Println("\n🔄 Updating project files...")
	if err := codegen.IncrementalAddService(cfg, newService, addOutputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating project files: %v\n", err)
		fmt.Println("\nNote: If marker comments are missing, you may need to run 'datagen build'")
		fmt.Println("to fully regenerate the project (this will overwrite customizations).")
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(addOutputDir)
	fmt.Printf("\n✅ Service '%s' added successfully to %s\n", newService.Name, absPath)
	fmt.Println("\n📝 Next steps:")
	fmt.Printf("  1. Customize the agent prompt file: %s\n", newService.Prompt)
	fmt.Println("  2. Test the new endpoint locally")
	fmt.Println("  3. Deploy your updated project: datagen deploy railway")
	fmt.Println("\n💡 Tip: Your custom code in other parts of the files has been preserved!")
}
