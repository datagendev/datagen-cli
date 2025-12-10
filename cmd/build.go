package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/datagendev/datagen-cli/internal/codegen"
	"github.com/datagendev/datagen-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	buildOutputDir string
	buildConfigPath string
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate code from datagen.toml",
	Long:  `Read datagen.toml and generate FastAPI boilerplate code`,
	Run:   runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&buildOutputDir, "output", "o", ".", "Output directory for generated project")
	buildCmd.Flags().StringVarP(&buildConfigPath, "config", "c", "datagen.toml", "Path to datagen.toml configuration file")
	buildCmd.MarkFlagDirname("output")
	buildCmd.MarkFlagFilename("config", "toml")
}

func runBuild(cmd *cobra.Command, args []string) {
	fmt.Println("🔨 Building project from datagen.toml...")

	// Load configuration
	cfg, err := config.LoadConfig(buildConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Loaded configuration with %d service(s)\n", len(cfg.Services))

	// Generate code with output directory
	if err := codegen.GenerateProject(cfg, buildOutputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(buildOutputDir)
	fmt.Printf("\n✅ Project generated successfully in %s\n", absPath)
	fmt.Println("\n📝 Next steps:")
	fmt.Printf("  1. cd %s\n", buildOutputDir)
	fmt.Println("  2. Review the generated code in app/")
	fmt.Println("  3. Set up your .env file (see .env.example)")
	fmt.Println("  4. Test locally: uvicorn app.main:app --reload")
	fmt.Println("  5. Deploy: datagen deploy railway")
}
