package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "datagen",
	Short: "DataGen CLI - Deploy and manage AI agents",
	Long: `DataGen CLI connects your GitHub repos to the DataGen platform,
discovers Claude Code agents, and deploys them as live endpoints.

Workflow:
  datagen login              Save your DataGen API key
  datagen mcp                Configure DataGen MCP locally
  datagen github connect     Install the GitHub App and connect repos
  datagen agents list        List discovered agents
  datagen agents deploy      Deploy an agent as a webhook endpoint
  datagen agents run         Trigger an agent execution
  datagen agents schedule    Set up cron schedules
  datagen agents config      Configure prompts, secrets, and recipients
  datagen secrets set        Store API keys for agent use`,
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
