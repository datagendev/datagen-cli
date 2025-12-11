package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "datagen",
	Short: "DataGen CLI - Generate agent boilerplate projects",
	Long: `DataGen CLI helps you create production-ready FastAPI boilerplate
for deploying Claude Code agents with DataGen MCP integration.

Usage:
  datagen start      - Interactive project setup
  datagen build      - Generate code from datagen.toml
  datagen add        - Add a new service to existing project
  datagen deploy     - Deploy to Railway`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(deployCmd)
}
