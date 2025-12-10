package cmd

import (
	"fmt"
	"os"

	"github.com/datagendev/datagen-cli/internal/codegen"
	"github.com/datagendev/datagen-cli/internal/config"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate code from datagen.toml",
	Long:  `Read datagen.toml and generate FastAPI boilerplate code`,
	Run:   runBuild,
}

func runBuild(cmd *cobra.Command, args []string) {
	fmt.Println("🔨 Building project from datagen.toml...")

	// Load configuration
	cfg, err := config.LoadConfig("datagen.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Loaded configuration with %d service(s)\n", len(cfg.Services))

	// Generate code
	if err := codegen.GenerateProject(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Project generated successfully!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Review the generated code in app/")
	fmt.Println("  2. Set up your .env file (see .env.example)")
	fmt.Println("  3. Test locally: uvicorn app.main:app --reload")
	fmt.Println("  4. Deploy: datagen deploy railway")
}
