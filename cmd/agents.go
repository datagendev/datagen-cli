package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/datagendev/datagen-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	agentsListRepo     string
	agentsListType     string
	agentsDeployedOnly bool
	agentsRunPayload   string
	agentsExecLimit    int

	// Config flags
	configSetPrompt       string
	configClearPrompt     bool
	configSecrets         string
	configPrMode          string
	configAddRecipient    string
	configRemoveRecipient string
	configNotifySuccess   string
	configNotifyFailure   string
	configNotifyReply     string

	// Schedule flags
	scheduleCron     string
	scheduleTimezone string
	scheduleName     string
	schedulePause    string
	scheduleResume   string
	scheduleDelete   string
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage discovered agents, skills, and commands",
	Long: `Manage agents, skills, and commands discovered from your connected GitHub repositories.

These are markdown files in .claude/ directories that define AI behavior
for the DataGen platform:
  - Agents:   .claude/agents/*.md
  - Skills:   .claude/skills/*.md  (user-invocable slash commands)
  - Commands: .claude/commands/*.md (custom slash commands)

Use --type to filter by type, or use 'datagen skills' / 'datagen commands' shortcuts.`,
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered agents",
	Long:  `List all agents discovered from connected GitHub repositories.`,
	Run:   runAgentsList,
}

var agentsShowCmd = &cobra.Command{
	Use:   "show <agent-id>",
	Short: "Show agent details",
	Long:  `Display detailed information about a specific agent.`,
	Args:  cobra.ExactArgs(1),
	Run:   runAgentsShow,
}

var agentsDeployCmd = &cobra.Command{
	Use:   "deploy <agent-id>",
	Short: "Deploy an agent",
	Long: `Deploy an agent to the DataGen platform.

The agent-id is the UUID of the agent (not its name). You can find it
by running 'datagen agents list' or 'datagen agents show <agent-id>'.

This creates a webhook endpoint that can be triggered to run the agent.`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentsDeploy,
}

var agentsUndeployCmd = &cobra.Command{
	Use:   "undeploy <agent-id>",
	Short: "Undeploy an agent",
	Long:  `Remove an agent deployment and its webhook.`,
	Args:  cobra.ExactArgs(1),
	Run:   runAgentsUndeploy,
}

var agentsRunCmd = &cobra.Command{
	Use:   "run <agent-id>",
	Short: "Trigger agent execution",
	Long: `Trigger an agent to run with an optional payload.

The agent must be deployed before it can be run.`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentsRun,
}

var agentsLogsCmd = &cobra.Command{
	Use:   "logs <agent-id>",
	Short: "View agent execution logs",
	Long:  `View recent execution history for an agent.`,
	Args:  cobra.ExactArgs(1),
	Run:   runAgentsLogs,
}

var agentsScheduleCmd = &cobra.Command{
	Use:   "schedule <agent-id>",
	Short: "Manage agent schedules",
	Long: `Manage cron schedules for an agent.

With no flags, lists all schedules for the agent.
Use --cron to create a new schedule.
Use --pause, --resume, or --delete to manage existing schedules.

Examples:
  datagen agents schedule <agent-id>
  datagen agents schedule <agent-id> --cron "0 9 * * *" --timezone "America/New_York"
  datagen agents schedule <agent-id> --pause <schedule-id>
  datagen agents schedule <agent-id> --resume <schedule-id>
  datagen agents schedule <agent-id> --delete <schedule-id>`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentsSchedule,
}

var agentsConfigCmd = &cobra.Command{
	Use:   "config <agent-id>",
	Short: "View or update agent configuration",
	Long: `View or update the unified configuration for an agent.

With no flags, displays the current configuration (entry prompt,
webhook settings, notifications, and recipients).

With any update flag, applies the changes and displays the result.

Examples:
  datagen agents config <agent-id>
  datagen agents config <agent-id> --set-prompt "You are a helpful assistant"
  datagen agents config <agent-id> --secrets KEY1,KEY2 --pr-mode create_pr
  datagen agents config <agent-id> --add-recipient user@example.com:OWNER
  datagen agents config <agent-id> --notify-success true --notify-failure default`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentsConfig,
}

func init() {
	agentsListCmd.Flags().StringVar(&agentsListRepo, "repo", "", "Filter by repository (owner/repo)")
	agentsListCmd.Flags().StringVar(&agentsListType, "type", "", "Filter by type: agent, skill, or command")
	agentsListCmd.Flags().BoolVar(&agentsDeployedOnly, "deployed", false, "Show only deployed agents")

	agentsRunCmd.Flags().StringVar(&agentsRunPayload, "payload", "{}", "JSON payload to send to the agent")

	agentsLogsCmd.Flags().IntVar(&agentsExecLimit, "limit", 10, "Maximum number of executions to show")

	agentsConfigCmd.Flags().StringVar(&configSetPrompt, "set-prompt", "", "Set the entry prompt text")
	agentsConfigCmd.Flags().BoolVar(&configClearPrompt, "clear-prompt", false, "Clear the entry prompt")
	agentsConfigCmd.Flags().StringVar(&configSecrets, "secrets", "", "Comma-separated secret names for webhook")
	agentsConfigCmd.Flags().StringVar(&configPrMode, "pr-mode", "", "PR mode: create_pr, auto_merge, or skip")
	agentsConfigCmd.Flags().StringVar(&configAddRecipient, "add-recipient", "", "Add recipient as email[:role] (role defaults to VIEWER)")
	agentsConfigCmd.Flags().StringVar(&configRemoveRecipient, "remove-recipient", "", "Remove recipient by ID")
	agentsConfigCmd.Flags().StringVar(&configNotifySuccess, "notify-success", "", "Email on success: true, false, or default")
	agentsConfigCmd.Flags().StringVar(&configNotifyFailure, "notify-failure", "", "Email on failure: true, false, or default")
	agentsConfigCmd.Flags().StringVar(&configNotifyReply, "notify-reply", "", "Email reply-to-resume: true, false, or default")

	agentsScheduleCmd.Flags().StringVar(&scheduleCron, "cron", "", "Cron expression to create a schedule (e.g. \"0 9 * * *\")")
	agentsScheduleCmd.Flags().StringVar(&scheduleTimezone, "timezone", "UTC", "Timezone for the schedule (e.g. \"America/New_York\")")
	agentsScheduleCmd.Flags().StringVar(&scheduleName, "name", "", "Optional name for the schedule")
	agentsScheduleCmd.Flags().StringVar(&schedulePause, "pause", "", "Pause a schedule by ID")
	agentsScheduleCmd.Flags().StringVar(&scheduleResume, "resume", "", "Resume a schedule by ID")
	agentsScheduleCmd.Flags().StringVar(&scheduleDelete, "delete", "", "Delete a schedule by ID")

	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsShowCmd)
	agentsCmd.AddCommand(agentsDeployCmd)
	agentsCmd.AddCommand(agentsUndeployCmd)
	agentsCmd.AddCommand(agentsRunCmd)
	agentsCmd.AddCommand(agentsLogsCmd)
	agentsCmd.AddCommand(agentsConfigCmd)
	agentsCmd.AddCommand(agentsScheduleCmd)
}

func runAgentsList(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	filterType := strings.ToUpper(agentsListType)
	if filterType != "" {
		fmt.Printf("%s Fetching %s...\n", typeIcon(filterType), typeLabelPlural(filterType))
	} else {
		fmt.Println("🤖 Fetching agents, skills, and commands...")
	}

	resp, err := client.ListAgents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Filter
	var filtered []api.Agent
	for _, a := range resp.Agents {
		if agentsListRepo != "" && a.Repo.FullName != agentsListRepo {
			continue
		}
		if agentsDeployedOnly && !a.IsDeployed {
			continue
		}
		if filterType != "" && strings.ToUpper(a.Type) != filterType {
			continue
		}
		filtered = append(filtered, a)
	}

	itemLabel := "items"
	if filterType != "" {
		itemLabel = typeLabelPlural(filterType)
	}

	if len(filtered) == 0 {
		fmt.Printf("\nNo %s found.\n", itemLabel)
		if agentsListRepo != "" || agentsDeployedOnly || filterType != "" {
			fmt.Println("Try removing filters to see all items.")
		} else {
			fmt.Println("Connect a repository with 'datagen github connect-repo <owner/repo>'")
		}
		return
	}

	if filterType != "" {
		// Flat by-repo grouping with type-specific header
		fmt.Printf("\n📋 %s (%d):\n\n", capitalize(itemLabel), len(filtered))
		printAgentsByRepo(filtered)
	} else {
		// Group by type, then by repo
		typeOrder := []string{"AGENT", "SKILL", "COMMAND"}
		for _, t := range typeOrder {
			var group []api.Agent
			for _, a := range filtered {
				if strings.ToUpper(a.Type) == t {
					group = append(group, a)
				}
			}
			if len(group) == 0 {
				continue
			}
			fmt.Printf("\n%s %s (%d):\n\n", typeIcon(t), capitalize(typeLabelPlural(t)), len(group))
			printAgentsByRepo(group)
		}

		// Items with unknown/empty type
		var other []api.Agent
		for _, a := range filtered {
			t := strings.ToUpper(a.Type)
			if t != "AGENT" && t != "SKILL" && t != "COMMAND" {
				other = append(other, a)
			}
		}
		if len(other) > 0 {
			fmt.Printf("\n🤖 Other (%d):\n\n", len(other))
			printAgentsByRepo(other)
		}
	}

	fmt.Println("Use 'datagen agents show <agent-id>' for details.")
	fmt.Println("Use 'datagen agents deploy <agent-id>' to deploy.")
}

func runAgentsShow(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Fetching details: %s\n", agentID)

	agent, err := client.GetAgent(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	t := strings.ToUpper(agent.Agent.Type)
	label := typeLabel(t)

	fmt.Println()
	fmt.Printf("%s %s: %s\n", typeIcon(t), capitalize(label), agent.Agent.AgentName)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("ID:          %s\n", agent.Agent.ID)
	fmt.Printf("Type:        %s\n", formatAgentType(agent.Agent.Type))
	fmt.Printf("Repository:  %s\n", agent.Agent.Repo.FullName)
	fmt.Printf("File:        %s\n", agent.Agent.FilePath)

	if agent.Agent.Description != "" {
		fmt.Printf("Description: %s\n", agent.Agent.Description)
	}
	if agent.Agent.EntryPrompt != "" {
		fmt.Printf("Prompt:      %s\n", agent.Agent.EntryPrompt)
	}

	// Status
	statusIcon := "⚪"
	status := "Not Deployed"
	if agent.Agent.IsDeployed {
		statusIcon = "🟢"
		status = "Deployed"
	}
	if agent.Agent.IsMissing {
		statusIcon = "🔴"
		status = "Missing (file deleted)"
	}
	fmt.Printf("Status:      %s %s\n", statusIcon, status)

	// Frontmatter
	if len(agent.Agent.Frontmatter) > 0 {
		fmt.Println()
		fmt.Println("📝 Configuration:")
		for k, v := range agent.Agent.Frontmatter {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	// Webhook info
	if agent.Agent.Webhook != nil {
		fmt.Println()
		fmt.Println("🔗 Webhook:")
		fmt.Printf("  Token: %s\n", agent.Agent.Webhook.WebhookToken)
		if agent.Agent.Webhook.LastTriggeredAt != nil {
			fmt.Printf("  Last triggered: %s\n", agent.Agent.Webhook.LastTriggeredAt.Format("2006-01-02 15:04:05"))
		}
	}

	// Recent executions summary
	if len(agent.RecentExecutions) > 0 {
		fmt.Println()
		fmt.Printf("📊 Recent executions (%d):\n", len(agent.RecentExecutions))
		for _, exec := range agent.RecentExecutions {
			statusIcon := getExecutionStatusIcon(exec.Status)
			execID := exec.ID
			if len(execID) > 8 {
				execID = execID[:8]
			}
			fmt.Printf("  %s %s - %s\n", statusIcon, execID, exec.Status)
		}
	}

	fmt.Println()
	if !agent.Agent.IsDeployed {
		fmt.Println("Use 'datagen agents deploy " + agentID + "' to deploy this agent.")
	} else {
		fmt.Println("Use 'datagen agents run " + agentID + "' to trigger execution.")
	}
}

func runAgentsDeploy(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	label := resolveAgentTypeLabel(client, agentID)

	fmt.Printf("🚀 Deploying %s: %s\n", label, agentID)

	resp, err := client.DeployAgent(agentID, "", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✅ %s deployed successfully!\n", capitalize(label))
	fmt.Println()
	fmt.Println("🔗 Webhook URL:")
	fmt.Printf("   %s\n", resp.WebhookUrl)
	fmt.Println()
	fmt.Println("📝 Trigger with:")
	fmt.Printf("   curl -X POST %s \\\n", resp.WebhookUrl)
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{\"message\": \"Hello\"}'")
	fmt.Println()
	fmt.Println("Or use: datagen agents run " + agentID + " --payload '{\"message\": \"Hello\"}'")
}

func runAgentsUndeploy(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	label := resolveAgentTypeLabel(client, agentID)

	fmt.Printf("🛑 Undeploying %s: %s\n", label, agentID)

	_, err = client.UndeployAgent(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ %s undeployed successfully!\n", capitalize(label))
	fmt.Println("   The webhook URL is no longer active.")
}

func runAgentsRun(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse payload
	var payload interface{}
	if err := json.Unmarshal([]byte(agentsRunPayload), &payload); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid JSON payload: %v\n", err)
		os.Exit(1)
	}

	label := resolveAgentTypeLabel(client, agentID)

	fmt.Printf("▶️  Running %s: %s\n", label, agentID)

	resp, err := client.RunAgent(agentID, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✅ %s execution started!\n", capitalize(label))
	fmt.Printf("   Execution ID: %s\n", resp.ExecutionID)
	fmt.Printf("   Status: %s\n", resp.Status)
	fmt.Println()
	fmt.Println("Check status with: datagen agents logs " + agentID)
}

func runAgentsLogs(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📜 Fetching execution logs for: %s\n", agentID)

	resp, err := client.ListAgentExecutions(agentID, agentsExecLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Executions) == 0 {
		fmt.Println("\nNo executions found for this agent.")
		return
	}

	fmt.Printf("\n📊 Executions (%d):\n\n", len(resp.Executions))

	for _, exec := range resp.Executions {
		statusIcon := getExecutionStatusIcon(exec.Status)
		duration := ""
		if exec.StartedAt != nil && exec.CompletedAt != nil {
			durationMs := exec.CompletedAt.Sub(*exec.StartedAt).Milliseconds()
			duration = fmt.Sprintf(" (%dms)", durationMs)
		}

		fmt.Printf("%s %s\n", statusIcon, exec.ID)
		fmt.Printf("   Status: %s%s\n", exec.Status, duration)
		if exec.StartedAt != nil {
			fmt.Printf("   Started: %s\n", exec.StartedAt.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Created: %s\n", exec.CreatedAt.Format("2006-01-02 15:04:05"))
		}

		if exec.ErrorMessage != "" {
			// Truncate error message
			errMsg := exec.ErrorMessage
			if len(errMsg) > 80 {
				errMsg = errMsg[:77] + "..."
			}
			fmt.Printf("   Error: %s\n", errMsg)
		}

		// Show truncated result if available
		if exec.Result != nil {
			resultStr := fmt.Sprintf("%v", exec.Result)
			if len(resultStr) > 100 {
				resultStr = resultStr[:97] + "..."
			}
			// Only show if not too verbose
			if !strings.Contains(resultStr, "\n") {
				fmt.Printf("   Result: %s\n", resultStr)
			}
		}

		fmt.Println()
	}
}

func runAgentsConfig(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if any update flags were provided
	hasUpdate := configSetPrompt != "" || configClearPrompt ||
		configSecrets != "" || configPrMode != "" ||
		configAddRecipient != "" || configRemoveRecipient != "" ||
		configNotifySuccess != "" || configNotifyFailure != "" ||
		configNotifyReply != ""

	if hasUpdate {
		req := buildConfigUpdateRequest()
		_, err := client.UpdateAgentConfig(agentID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration updated.")
		fmt.Println()
	}

	// Always fetch and display current config
	config, err := client.GetAgentConfig(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	displayAgentConfig(config)
}

func buildConfigUpdateRequest() api.UpdateAgentConfigRequest {
	req := api.UpdateAgentConfigRequest{}

	// Entry prompt
	if configClearPrompt {
		empty := ""
		req.EntryPrompt = &empty
	} else if configSetPrompt != "" {
		req.EntryPrompt = &configSetPrompt
	}

	// Webhook settings
	webhook := map[string]interface{}{}
	if configSecrets != "" {
		names := strings.Split(configSecrets, ",")
		trimmed := make([]string, len(names))
		for i, n := range names {
			trimmed[i] = strings.TrimSpace(n)
		}
		webhook["secretNames"] = trimmed
	}
	if configPrMode != "" {
		webhook["prMode"] = configPrMode
	}
	if len(webhook) > 0 {
		req.Webhook = webhook
	}

	// Notifications
	notifications := map[string]interface{}{}
	if configNotifySuccess != "" {
		notifications["emailOnSuccess"] = parseBoolOrNull(configNotifySuccess)
	}
	if configNotifyFailure != "" {
		notifications["emailOnFailure"] = parseBoolOrNull(configNotifyFailure)
	}
	if configNotifyReply != "" {
		notifications["emailReplyEnabled"] = parseBoolOrNull(configNotifyReply)
	}
	if len(notifications) > 0 {
		req.Notifications = notifications
	}

	// Recipients
	var recipientsUpdate *api.RecipientsUpdate
	if configAddRecipient != "" {
		email, role := parseRecipientFlag(configAddRecipient)
		if recipientsUpdate == nil {
			recipientsUpdate = &api.RecipientsUpdate{}
		}
		recipientsUpdate.Add = []api.RecipientAdd{{Email: email, Role: role}}
	}
	if configRemoveRecipient != "" {
		if recipientsUpdate == nil {
			recipientsUpdate = &api.RecipientsUpdate{}
		}
		recipientsUpdate.Remove = []string{configRemoveRecipient}
	}
	req.Recipients = recipientsUpdate

	return req
}

// parseBoolOrNull converts "true"/"false"/"default" to the appropriate value.
// "default" returns nil (clears the override), true/false return the boolean.
func parseBoolOrNull(val string) interface{} {
	switch strings.ToLower(val) {
	case "true":
		return true
	case "false":
		return false
	default:
		return nil
	}
}

// parseRecipientFlag parses "email[:role]" format. Defaults role to "VIEWER".
func parseRecipientFlag(val string) (string, string) {
	parts := strings.SplitN(val, ":", 2)
	email := parts[0]
	role := "VIEWER"
	if len(parts) == 2 && parts[1] != "" {
		role = strings.ToUpper(parts[1])
	}
	return email, role
}

func displayAgentConfig(config *api.AgentConfigResponse) {
	fmt.Printf("Agent: %s (%s)\n", config.AgentName, config.AgentID)
	fmt.Printf("Repo:  %s\n", config.Repo)

	// Entry Prompt
	fmt.Println()
	fmt.Println("Entry Prompt:")
	if config.EntryPrompt != nil && *config.EntryPrompt != "" {
		fmt.Printf("  \"%s\"\n", *config.EntryPrompt)
	} else {
		fmt.Println("  (not set)")
	}

	// Webhook
	fmt.Println()
	fmt.Println("Webhook:")
	if config.Webhook != nil {
		if config.Webhook.IsActive {
			fmt.Println("  Status:  Active")
		} else {
			fmt.Println("  Status:  Inactive")
		}
		fmt.Printf("  PR Mode: %s\n", config.Webhook.PrMode)
		if len(config.Webhook.SecretNames) > 0 {
			fmt.Printf("  Secrets: %s\n", strings.Join(config.Webhook.SecretNames, ", "))
		} else {
			fmt.Println("  Secrets: (none)")
		}
	} else {
		fmt.Println("  (not deployed)")
	}

	// Notifications
	fmt.Println()
	fmt.Println("Notifications:")
	if config.Notifications != nil {
		fmt.Printf("  On Success: %s\n", formatBoolOverride(config.Notifications.EmailOnSuccess))
		fmt.Printf("  On Failure: %s\n", formatBoolOverride(config.Notifications.EmailOnFailure))
		fmt.Printf("  Reply:      %s\n", formatBoolOverride(config.Notifications.EmailReplyEnabled))
	} else {
		fmt.Println("  (using global defaults)")
	}

	// Recipients
	fmt.Println()
	fmt.Println("Recipients:")
	if len(config.Recipients) > 0 {
		for _, r := range config.Recipients {
			name := ""
			if r.Name != nil {
				name = " (" + *r.Name + ")"
			}
			fmt.Printf("  %s%s [%s] id=%s\n", r.Email, name, r.Role, r.ID)
		}
	} else {
		fmt.Println("  (none)")
	}
}

func formatBoolOverride(val *bool) string {
	if val == nil {
		return "default (global)"
	}
	if *val {
		return "true (agent override)"
	}
	return "false (agent override)"
}

func formatAgentType(t string) string {
	return typeLabel(strings.ToUpper(t))
}

// capitalize uppercases the first letter of a string.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// typeLabel returns the singular lowercase label for a type.
func typeLabel(t string) string {
	switch strings.ToUpper(t) {
	case "SKILL":
		return "skill"
	case "COMMAND":
		return "command"
	default:
		return "agent"
	}
}

// typeLabelPlural returns the plural lowercase label for a type.
func typeLabelPlural(t string) string {
	switch strings.ToUpper(t) {
	case "SKILL":
		return "skills"
	case "COMMAND":
		return "commands"
	default:
		return "agents"
	}
}

// typeIcon returns the emoji icon for a type.
func typeIcon(t string) string {
	switch strings.ToUpper(t) {
	case "SKILL":
		return "⚡"
	case "COMMAND":
		return "📎"
	default:
		return "🤖"
	}
}

// resolveAgentTypeLabel fetches the agent to determine its type label.
// Falls back to "agent" on error.
func resolveAgentTypeLabel(client *api.Client, agentID string) string {
	resp, err := client.GetAgent(agentID)
	if err != nil {
		return "agent"
	}
	return typeLabel(strings.ToUpper(resp.Agent.Type))
}

// printAgentsByRepo prints agents grouped by repository.
func printAgentsByRepo(agents []api.Agent) {
	byRepo := make(map[string][]api.Agent)
	var repoOrder []string
	for _, a := range agents {
		if _, seen := byRepo[a.Repo.FullName]; !seen {
			repoOrder = append(repoOrder, a.Repo.FullName)
		}
		byRepo[a.Repo.FullName] = append(byRepo[a.Repo.FullName], a)
	}

	for _, repo := range repoOrder {
		repoAgents := byRepo[repo]
		fmt.Printf("📁 %s\n", repo)
		for _, a := range repoAgents {
			statusIcon := "⚪"
			status := "not deployed"
			if a.IsDeployed {
				statusIcon = "🟢"
				status = "deployed"
			}
			if a.IsMissing {
				statusIcon = "🔴"
				status = "missing"
			}

			tl := formatAgentType(a.Type)
			fmt.Printf("  %s %s [%s] (%s)\n", statusIcon, a.AgentName, tl, status)
			fmt.Printf("    ID: %s\n", a.ID)
			if a.Description != "" {
				desc := a.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				fmt.Printf("    %s\n", desc)
			}
		}
		fmt.Println()
	}
}

func runAgentsSchedule(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine action based on flags
	switch {
	case scheduleCron != "":
		// Create schedule
		req := api.CreateScheduleRequest{
			Cron:     scheduleCron,
			Timezone: scheduleTimezone,
			Name:     scheduleName,
		}

		resp, err := client.CreateAgentSchedule(agentID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating schedule: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println("Schedule created successfully!")
		fmt.Println()
		displayScheduleInfo(&resp.Schedule)

	case schedulePause != "":
		updates := map[string]interface{}{"isActive": false}
		_, err := client.UpdateAgentSchedule(agentID, schedulePause, updates)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error pausing schedule: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Schedule %s paused.\n", schedulePause)

	case scheduleResume != "":
		updates := map[string]interface{}{"isActive": true}
		_, err := client.UpdateAgentSchedule(agentID, scheduleResume, updates)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming schedule: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Schedule %s resumed.\n", scheduleResume)

	case scheduleDelete != "":
		err := client.DeleteAgentSchedule(agentID, scheduleDelete)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting schedule: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Schedule %s deleted.\n", scheduleDelete)

	default:
		// List schedules
		resp, err := client.ListAgentSchedules(agentID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(resp.Schedules) == 0 {
			fmt.Println("\nNo schedules found for this agent.")
			fmt.Println("Create one with: datagen agents schedule " + agentID + " --cron \"0 9 * * *\"")
			return
		}

		fmt.Printf("\nSchedules (%d):\n\n", len(resp.Schedules))

		for _, s := range resp.Schedules {
			displayScheduleInfo(&s)
			fmt.Println()
		}
	}
}

func displayScheduleInfo(s *api.ScheduleInfo) {
	status := "Active"
	if !s.IsActive {
		status = "Paused"
	}

	name := s.Name
	if name == "" {
		name = "(unnamed)"
	}

	fmt.Printf("  %-8s %s\n", status, name)
	fmt.Printf("           ID: %s\n", s.ID)
	fmt.Printf("           Cron: %s (%s)\n", s.CronExpression, s.Timezone)
	if s.NextRunAt != nil {
		fmt.Printf("           Next: %s\n", s.NextRunAt.Format("2006-01-02 15:04:05"))
	}
	if s.LastRunAt != nil {
		fmt.Printf("           Last: %s\n", s.LastRunAt.Format("2006-01-02 15:04:05"))
	}
}

func getExecutionStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "completed", "success":
		return "✅"
	case "failed", "error":
		return "❌"
	case "running", "in_progress":
		return "🔄"
	case "pending", "queued":
		return "⏳"
	default:
		return "⚪"
	}
}
