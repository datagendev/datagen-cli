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
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage discovered agents",
	Long: `Manage agents discovered from your connected GitHub repositories.

Agents are markdown files in .claude/agents/ directories that define
AI agent behavior for the DataGen platform.`,
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

	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsShowCmd)
	agentsCmd.AddCommand(agentsDeployCmd)
	agentsCmd.AddCommand(agentsUndeployCmd)
	agentsCmd.AddCommand(agentsRunCmd)
	agentsCmd.AddCommand(agentsLogsCmd)
	agentsCmd.AddCommand(agentsConfigCmd)
}

func runAgentsList(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🤖 Fetching agents...")

	resp, err := client.ListAgents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Filter agents
	var filtered []api.Agent
	for _, a := range resp.Agents {
		if agentsListRepo != "" && a.Repo.FullName != agentsListRepo {
			continue
		}
		if agentsDeployedOnly && !a.IsDeployed {
			continue
		}
		filtered = append(filtered, a)
	}

	if len(filtered) == 0 {
		fmt.Println("\nNo agents found.")
		if agentsListRepo != "" || agentsDeployedOnly {
			fmt.Println("Try removing filters to see all agents.")
		} else {
			fmt.Println("Connect a repository with 'datagen github connect-repo <owner/repo>'")
		}
		return
	}

	fmt.Printf("\n📋 Agents (%d):\n\n", len(filtered))

	// Group by repo
	byRepo := make(map[string][]api.Agent)
	for _, a := range filtered {
		byRepo[a.Repo.FullName] = append(byRepo[a.Repo.FullName], a)
	}

	for repo, agents := range byRepo {
		fmt.Printf("📁 %s\n", repo)
		for _, a := range agents {
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

			fmt.Printf("  %s %s (%s)\n", statusIcon, a.AgentName, status)
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

	fmt.Println("Use 'datagen agents show <agent-id>' for details.")
	fmt.Println("Use 'datagen agents deploy <agent-id>' to deploy an agent.")
}

func runAgentsShow(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Fetching agent: %s\n", agentID)

	agent, err := client.GetAgent(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("🤖 Agent: %s\n", agent.Agent.AgentName)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("ID:          %s\n", agent.Agent.ID)
	fmt.Printf("Repository:  %s\n", agent.Agent.Repo.FullName)
	fmt.Printf("File:        %s\n", agent.Agent.FilePath)

	if agent.Agent.Description != "" {
		fmt.Printf("Description: %s\n", agent.Agent.Description)
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

	fmt.Printf("🚀 Deploying agent: %s\n", agentID)

	resp, err := client.DeployAgent(agentID, "", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Agent deployed successfully!")
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

	fmt.Printf("🛑 Undeploying agent: %s\n", agentID)

	_, err = client.UndeployAgent(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Agent undeployed successfully!")
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

	fmt.Printf("▶️  Running agent: %s\n", agentID)

	resp, err := client.RunAgent(agentID, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Execution started!")
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
