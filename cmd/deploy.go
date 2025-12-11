package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	deployOutputDir string
)

var deployCmd = &cobra.Command{
	Use:   "deploy [platform]",
	Short: "Deploy to cloud platform",
	Long:  `Deploy your project to a cloud platform (currently supports: railway)`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runDeploy,
}

func init() {
	deployCmd.Flags().StringVarP(&deployOutputDir, "output", "o", ".", "Directory containing the project to deploy")
	deployCmd.MarkFlagDirname("output")
}

func runDeploy(cmd *cobra.Command, args []string) {
	platform := "railway"
	if len(args) > 0 {
		platform = args[0]
	}

	if platform != "railway" {
		fmt.Fprintf(os.Stderr, "Error: Only 'railway' platform is currently supported\n")
		os.Exit(1)
	}

	// Change to output directory if specified
	if deployOutputDir != "." {
		if err := os.Chdir(deployOutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Cannot access directory %s: %v\n", deployOutputDir, err)
			os.Exit(1)
		}
		fmt.Printf("📁 Working in directory: %s\n", deployOutputDir)
	}

	fmt.Println("🚀 Deploying to Railway...")

	// Check if railway CLI is installed
	if _, err := exec.LookPath("railway"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Railway CLI is not installed\n")
		fmt.Println("Install it from: https://docs.railway.com/guides/cli")
		os.Exit(1)
	}

	// Check if logged in
	checkCmd := exec.Command("railway", "whoami")
	if err := checkCmd.Run(); err != nil {
		fmt.Println("⚠️  Not logged in to Railway")
		fmt.Println("Run 'railway login' to authenticate")
		os.Exit(1)
	}

	// TODO: Implement full deployment logic
	// - Check for railway.json
	// - Upload environment variables
	// - Run railway up
	// - Get domain

	fmt.Println("\n✅ Deployment placeholder - full implementation coming soon!")
	fmt.Println("\nManual steps:")
	fmt.Println("  1. railway login")
	fmt.Println("  2. railway init")
	fmt.Println("  3. railway up --detach")
	fmt.Println("  4. railway domain")
}
