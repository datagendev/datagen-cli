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
discovers Claude Code agents, skills, and commands, and deploys them as live endpoints.

Workflow:
  datagen login              Save your DataGen API key
  datagen mcp                Configure DataGen MCP locally
  datagen tools list         List deployed custom tools
  datagen tools deploy       Deploy a Python custom tool
  datagen github connect     Install the GitHub App and connect repos
  datagen agents list        List discovered agents, skills, and commands
  datagen agents deploy      Deploy an agent/skill/command as a webhook endpoint
  datagen skills list        List discovered skills (shortcut)
  datagen commands list      List discovered commands (shortcut)
  datagen agents run         Trigger an execution
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
	rootCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(githubCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(skillsCmd)
	rootCmd.AddCommand(commandsCmd)
	rootCmd.AddCommand(secretsCmd)
}
