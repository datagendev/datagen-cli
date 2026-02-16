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
  datagen login      - Save your DataGen API key
  datagen mcp        - Configure DataGen MCP locally
  datagen github     - Manage GitHub connection
  datagen agents     - Manage discovered agents
  datagen secrets    - Manage secrets stored in DataGen`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(githubCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(secretsCmd)
}
