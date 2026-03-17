package cmd

import (
	"github.com/spf13/cobra"
)

// skillsCmd is a top-level alias for "agents --type skill"
var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage discovered skills (alias for 'agents --type skill')",
	Long: `Manage skills discovered from your connected GitHub repositories.

Skills are markdown files in .claude/skills/ directories that define
user-invocable slash commands for the DataGen platform.

This is a convenience alias for 'datagen agents --type skill'.`,
}

// commandsCmd is a top-level alias for "agents --type command"
var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "Manage discovered commands (alias for 'agents --type command')",
	Long: `Manage commands discovered from your connected GitHub repositories.

Commands are markdown files in .claude/commands/ directories that define
custom slash commands for the DataGen platform.

This is a convenience alias for 'datagen agents --type command'.`,
}

// makeTypedListFunc returns a run function that sets the type filter then delegates.
func makeTypedListFunc(typeName string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		agentsListType = typeName
		runAgentsList(cmd, args)
	}
}

// makeTypedRunFunc returns a run function that delegates directly (type resolved from API).
func makeTypedRunFunc(original func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return original
}

func init() {
	// Skills subcommands
	skillsList := &cobra.Command{
		Use:   "list",
		Short: "List discovered skills",
		Run:   makeTypedListFunc("skill"),
	}
	skillsList.Flags().StringVar(&agentsListRepo, "repo", "", "Filter by repository (owner/repo)")
	skillsList.Flags().BoolVar(&agentsDeployedOnly, "deployed", false, "Show only deployed skills")

	skillsCmd.AddCommand(skillsList)
	skillsCmd.AddCommand(&cobra.Command{Use: "show <id>", Short: "Show skill details", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsShow)})
	skillsCmd.AddCommand(&cobra.Command{Use: "deploy <id>", Short: "Deploy a skill", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsDeploy)})
	skillsCmd.AddCommand(&cobra.Command{Use: "undeploy <id>", Short: "Undeploy a skill", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsUndeploy)})
	skillsCmd.AddCommand(&cobra.Command{Use: "run <id>", Short: "Trigger skill execution", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsRun)})
	skillsCmd.AddCommand(&cobra.Command{Use: "logs <id>", Short: "View skill execution logs", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsLogs)})
	skillsCmd.AddCommand(&cobra.Command{Use: "config <id>", Short: "View or update skill configuration", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsConfig)})
	skillsCmd.AddCommand(&cobra.Command{Use: "schedule <id>", Short: "Manage skill schedules", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsSchedule)})

	// Commands subcommands
	commandsList := &cobra.Command{
		Use:   "list",
		Short: "List discovered commands",
		Run:   makeTypedListFunc("command"),
	}
	commandsList.Flags().StringVar(&agentsListRepo, "repo", "", "Filter by repository (owner/repo)")
	commandsList.Flags().BoolVar(&agentsDeployedOnly, "deployed", false, "Show only deployed commands")

	commandsCmd.AddCommand(commandsList)
	commandsCmd.AddCommand(&cobra.Command{Use: "show <id>", Short: "Show command details", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsShow)})
	commandsCmd.AddCommand(&cobra.Command{Use: "deploy <id>", Short: "Deploy a command", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsDeploy)})
	commandsCmd.AddCommand(&cobra.Command{Use: "undeploy <id>", Short: "Undeploy a command", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsUndeploy)})
	commandsCmd.AddCommand(&cobra.Command{Use: "run <id>", Short: "Trigger command execution", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsRun)})
	commandsCmd.AddCommand(&cobra.Command{Use: "logs <id>", Short: "View command execution logs", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsLogs)})
	commandsCmd.AddCommand(&cobra.Command{Use: "config <id>", Short: "View or update command configuration", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsConfig)})
	commandsCmd.AddCommand(&cobra.Command{Use: "schedule <id>", Short: "Manage command schedules", Args: cobra.ExactArgs(1), Run: makeTypedRunFunc(runAgentsSchedule)})
}
