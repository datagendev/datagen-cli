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

	// Logs flags
	logsExecID    string
	logsSessionID string
	logsLevel     string
	logsTranscript bool

	// Output flags
	outputExecID    string
	outputSessionID string
	outputJSON      bool

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
	Long: `View recent execution history for an agent.

By default, lists recent executions as a summary.
Use --execution or --session to view detailed logs for a specific execution.
Use --transcript to view the full Claude conversation transcript.

Examples:
  datagen agents logs <agent-id>
  datagen agents logs <agent-id> --execution <execution-id>
  datagen agents logs <agent-id> --session <session-id>
  datagen agents logs <agent-id> --execution <execution-id> --transcript
  datagen agents logs <agent-id> --execution <execution-id> --level ERROR`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentsLogs,
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

var agentsOutputCmd = &cobra.Command{
	Use:   "output <agent-id>",
	Short: "Show agent execution output",
	Long: `Show the final output/result of an agent execution.

By default, shows the output of the most recent execution.
Use --execution to specify an execution ID, or --session to look up by session ID.

Examples:
  datagen agents output <agent-id>
  datagen agents output <agent-id> --execution <execution-id>
  datagen agents output <agent-id> --session <session-id>
  datagen agents output <agent-id> --json`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentsOutput,
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

	agentsLogsCmd.Flags().IntVar(&agentsExecLimit, "limit", 10, "Maximum number of executions (or log entries) to show")
	agentsLogsCmd.Flags().StringVar(&logsExecID, "execution", "", "Show detailed logs for a specific execution")
	agentsLogsCmd.Flags().StringVar(&logsSessionID, "session", "", "Show detailed logs by session ID")
	agentsLogsCmd.Flags().StringVar(&logsLevel, "level", "", "Filter logs by level (INFO, WARNING, ERROR)")
	agentsLogsCmd.Flags().BoolVar(&logsTranscript, "transcript", false, "Show Claude conversation transcript instead of execution logs")

	agentsOutputCmd.Flags().StringVar(&outputExecID, "execution", "", "Execution ID to show output for")
	agentsOutputCmd.Flags().StringVar(&outputSessionID, "session", "", "Session ID to look up")
	agentsOutputCmd.Flags().BoolVar(&outputJSON, "json", false, "Output raw result as JSON")

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
	agentsCmd.AddCommand(agentsOutputCmd)
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

	// Resolve execution ID from session if needed
	executionID := logsExecID
	if logsSessionID != "" && executionID == "" {
		executionID, err = resolveExecutionFromSession(client, agentID, logsSessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// If we have an execution ID, show detailed logs or transcript
	if executionID != "" {
		if logsTranscript {
			runTranscript(client, agentID, executionID)
		} else {
			runDetailedLogs(client, executionID)
		}
		return
	}

	// Default: list executions summary
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
		if exec.SdkSessionID != nil && *exec.SdkSessionID != "" {
			fmt.Printf("   Session: %s\n", *exec.SdkSessionID)
		}
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

// resolveExecutionFromSession looks up the execution ID for a given session ID
func resolveExecutionFromSession(client *api.Client, agentID, sessionID string) (string, error) {
	fmt.Printf("🔍 Looking up execution by session: %s\n", sessionID)

	output, err := client.GetAgentExecutionOutputBySession(agentID, sessionID)
	if err != nil {
		return "", fmt.Errorf("could not find execution for session %s: %w", sessionID, err)
	}

	fmt.Printf("   Found execution: %s\n\n", output.ExecutionID)
	return output.ExecutionID, nil
}

// runDetailedLogs fetches and displays detailed execution logs
func runDetailedLogs(client *api.Client, executionID string) {
	limit := agentsExecLimit
	if limit <= 10 {
		limit = 1000 // default to more logs when viewing details
	}

	fmt.Printf("📜 Fetching detailed logs for execution: %s\n", executionID)

	resp, err := client.GetExecutionLogs(executionID, logsLevel, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Execution != nil {
		fmt.Printf("   Status: %s %s\n", getExecutionStatusIcon(resp.Execution.Status), resp.Execution.Status)
	}
	if resp.Pagination != nil {
		fmt.Printf("   Showing %d of %d log entries (deduplicated)\n", len(resp.Logs), resp.Pagination.Total)
	}
	fmt.Println()

	if len(resp.Logs) == 0 {
		fmt.Println("No log entries found.")
		return
	}

	// Deduplicate logs by message content
	seen := make(map[string]bool)
	for _, log := range resp.Logs {
		key := log.Timestamp.Format("15:04:05") + "|" + log.Message
		if seen[key] {
			continue
		}
		seen[key] = true

		levelTag := formatLogLevel(log.Level)
		ts := log.Timestamp.Format("15:04:05")
		msg := formatLogMessage(log.Message)

		if msg != "" {
			fmt.Printf("[%s] %s %s\n", ts, levelTag, msg)
		}
	}
}

// formatLogMessage cleans up a raw log message:
// - Parses JSON assistant/tool messages into readable summaries
// - Strips signatures and binary data
// - Keeps plain text as-is
func formatLogMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	// Not JSON -- return as plain text
	if !strings.HasPrefix(msg, "{") {
		return msg
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(msg), &parsed); err != nil {
		return msg // not valid JSON, return as-is
	}

	msgType, _ := parsed["type"].(string)
	subtype, _ := parsed["subtype"].(string)

	switch msgType {
	case "system":
		return formatSystemLog(parsed, subtype)
	case "assistant":
		return formatAssistantLog(parsed)
	case "user":
		return formatUserLog(parsed)
	case "result":
		return formatResultLog(parsed)
	default:
		// Unknown type -- show type + compact summary
		if msgType != "" {
			return fmt.Sprintf("[%s] %s", msgType, compactJSON(parsed, 200))
		}
		return compactJSON(parsed, 200)
	}
}

func formatSystemLog(parsed map[string]interface{}, subtype string) string {
	switch subtype {
	case "init":
		sessionID, _ := parsed["session_id"].(string)
		model, _ := parsed["model"].(string)
		cwd, _ := parsed["cwd"].(string)

		parts := []string{"[system:init]"}
		if model != "" {
			parts = append(parts, "model="+model)
		}
		if sessionID != "" {
			parts = append(parts, "session="+sessionID)
		}
		if cwd != "" {
			// Shorten long paths
			if idx := strings.LastIndex(cwd, "/"); idx > 30 {
				cwd = "..." + cwd[idx:]
			}
			parts = append(parts, "cwd="+cwd)
		}

		// Count tools
		if tools, ok := parsed["tools"].([]interface{}); ok {
			parts = append(parts, fmt.Sprintf("tools=%d", len(tools)))
		}

		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("[system:%s] %s", subtype, compactJSON(parsed, 150))
	}
}

func formatAssistantLog(parsed map[string]interface{}) string {
	message, ok := parsed["message"].(map[string]interface{})
	if !ok {
		return "[assistant] (no message)"
	}

	content, ok := message["content"].([]interface{})
	if !ok {
		return "[assistant] (no content)"
	}

	var parts []string
	for _, block := range content {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, _ := blockMap["type"].(string)
		switch blockType {
		case "thinking":
			thinking, _ := blockMap["thinking"].(string)
			if len(thinking) > 150 {
				thinking = thinking[:147] + "..."
			}
			parts = append(parts, fmt.Sprintf("[thinking] %s", thinking))
		case "text":
			text, _ := blockMap["text"].(string)
			if len(text) > 300 {
				text = text[:297] + "..."
			}
			parts = append(parts, text)
		case "tool_use":
			name, _ := blockMap["name"].(string)
			input, _ := blockMap["input"].(map[string]interface{})
			inputStr := compactJSON(input, 150)
			parts = append(parts, fmt.Sprintf("[tool_use: %s] %s", name, inputStr))
		case "tool_result":
			parts = append(parts, "[tool_result]")
		case "signature":
			// Skip signatures entirely
			continue
		default:
			parts = append(parts, fmt.Sprintf("[%s]", blockType))
		}
	}

	if len(parts) == 0 {
		return "" // skip empty (signature-only) messages
	}

	return "[assistant] " + strings.Join(parts, " | ")
}

func formatUserLog(parsed map[string]interface{}) string {
	message, ok := parsed["message"].(map[string]interface{})
	if !ok {
		return "[user] (no message)"
	}

	content := message["content"]
	text := formatContent(content)
	if len(text) > 300 {
		text = text[:297] + "..."
	}
	return "[user] " + text
}

func formatResultLog(parsed map[string]interface{}) string {
	result, _ := parsed["result"].(string)
	if len(result) > 300 {
		result = result[:297] + "..."
	}
	if result != "" {
		return "[result] " + result
	}
	return "[result] " + compactJSON(parsed, 200)
}

// compactJSON returns a compact one-line JSON representation, truncated to maxLen
func compactJSON(v interface{}, maxLen int) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	s := string(data)
	if len(s) > maxLen {
		s = s[:maxLen-3] + "..."
	}
	return s
}

// runTranscript fetches and displays the Claude conversation transcript
func runTranscript(client *api.Client, agentID, executionID string) {
	limit := agentsExecLimit
	if limit <= 10 {
		limit = 200
	}

	fmt.Printf("📜 Fetching transcript for execution: %s\n", executionID)

	resp, err := client.GetExecutionTranscript(agentID, executionID, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("   Status: %s %s\n", getExecutionStatusIcon(resp.Execution.Status), resp.Execution.Status)
	if resp.Execution.StartedAt != nil {
		fmt.Printf("   Started: %s\n", resp.Execution.StartedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("   Session: %s\n", resp.Transcript.SessionID)
	fmt.Printf("   Entries: %d\n", resp.Transcript.EntryCount)
	fmt.Println()

	if len(resp.Transcript.Entries) == 0 {
		fmt.Println("No transcript entries found.")
		return
	}

	for _, entry := range resp.Transcript.Entries {
		entryType := "<unknown>"
		if entry.Type != nil {
			entryType = *entry.Type
		}

		switch entryType {
		case "user_message", "human":
			content := extractMessageContent(entry.Raw)
			fmt.Printf(">>> USER:\n%s\n\n", content)

		case "assistant_message", "assistant":
			content := extractMessageContent(entry.Raw)
			if entry.Subtype != nil && *entry.Subtype == "tool_use" {
				fmt.Printf("<<< ASSISTANT (tool_use):\n%s\n\n", content)
			} else {
				fmt.Printf("<<< ASSISTANT:\n%s\n\n", content)
			}

		case "tool_result":
			content := extractMessageContent(entry.Raw)
			// Truncate long tool results
			if len(content) > 500 {
				content = content[:497] + "..."
			}
			fmt.Printf("    TOOL RESULT:\n%s\n\n", content)

		default:
			// Show type and a compact representation
			data, _ := json.MarshalIndent(entry.Raw, "    ", "  ")
			preview := string(data)
			if len(preview) > 300 {
				preview = preview[:297] + "..."
			}
			fmt.Printf("--- %s:\n    %s\n\n", entryType, preview)
		}
	}
}

// extractMessageContent pulls readable content from a transcript entry's raw JSON
func extractMessageContent(raw map[string]interface{}) string {
	// Try "message" field first (Claude SDK format)
	if msg, ok := raw["message"]; ok {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if content, ok := msgMap["content"]; ok {
				return formatContent(content)
			}
		}
	}

	// Try top-level "content"
	if content, ok := raw["content"]; ok {
		return formatContent(content)
	}

	// Try "text"
	if text, ok := raw["text"]; ok {
		if s, ok := text.(string); ok {
			return s
		}
	}

	// Fallback: compact JSON
	data, _ := json.Marshal(raw)
	s := string(data)
	if len(s) > 500 {
		s = s[:497] + "..."
	}
	return s
}

// formatContent handles content that may be a string or array of content blocks
func formatContent(content interface{}) string {
	if s, ok := content.(string); ok {
		return s
	}

	if arr, ok := content.([]interface{}); ok {
		var parts []string
		for _, item := range arr {
			if block, ok := item.(map[string]interface{}); ok {
				blockType, _ := block["type"].(string)
				switch blockType {
				case "text":
					if text, ok := block["text"].(string); ok {
						parts = append(parts, text)
					}
				case "tool_use":
					name, _ := block["name"].(string)
					input, _ := json.Marshal(block["input"])
					inputStr := string(input)
					if len(inputStr) > 200 {
						inputStr = inputStr[:197] + "..."
					}
					parts = append(parts, fmt.Sprintf("[tool_use: %s] %s", name, inputStr))
				case "tool_result":
					content, _ := json.Marshal(block["content"])
					contentStr := string(content)
					if len(contentStr) > 200 {
						contentStr = contentStr[:197] + "..."
					}
					parts = append(parts, fmt.Sprintf("[tool_result] %s", contentStr))
				default:
					data, _ := json.Marshal(block)
					s := string(data)
					if len(s) > 200 {
						s = s[:197] + "..."
					}
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, "\n")
	}

	data, _ := json.Marshal(content)
	return string(data)
}

// formatLogLevel returns a formatted log level tag
func formatLogLevel(level string) string {
	switch strings.ToUpper(level) {
	case "ERROR":
		return "ERROR"
	case "WARNING", "WARN":
		return "WARN "
	case "DEBUG":
		return "DEBUG"
	default:
		return "INFO "
	}
}

func runAgentsOutput(cmd *cobra.Command, args []string) {
	agentID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var output *api.ExecutionOutputResponse

	switch {
	case outputSessionID != "":
		// Look up by session ID
		fmt.Printf("🔍 Looking up output by session: %s\n", outputSessionID)
		output, err = client.GetAgentExecutionOutputBySession(agentID, outputSessionID)

	case outputExecID != "":
		// Look up by execution ID
		fmt.Printf("🔍 Fetching output for execution: %s\n", outputExecID)
		output, err = client.GetAgentExecutionOutput(agentID, outputExecID)

	default:
		// Get latest execution, then fetch its output
		fmt.Println("🔍 Fetching latest execution output...")
		execResp, execErr := client.ListAgentExecutions(agentID, 1)
		if execErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", execErr)
			os.Exit(1)
		}
		if len(execResp.Executions) == 0 {
			fmt.Println("\nNo executions found for this agent.")
			fmt.Println("Run the agent first with: datagen agents run " + agentID)
			return
		}
		latestExec := execResp.Executions[0]
		output, err = client.GetAgentExecutionOutput(agentID, latestExec.ID)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// JSON mode: dump the raw result
	if outputJSON {
		data, _ := json.MarshalIndent(output.Result, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Display formatted output
	label := typeLabel(strings.ToUpper(output.Type))
	fmt.Println()
	fmt.Printf("%s %s: %s\n", typeIcon(strings.ToUpper(output.Type)), capitalize(label), output.AgentName)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Execution: %s\n", output.ExecutionID)
	fmt.Printf("Status:    %s %s\n", getExecutionStatusIcon(output.Status), output.Status)

	if output.SdkSessionID != nil && *output.SdkSessionID != "" {
		fmt.Printf("Session:   %s\n", *output.SdkSessionID)
	}

	if output.StartedAt != nil {
		fmt.Printf("Started:   %s\n", output.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if output.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", output.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if output.DurationMs != nil {
		fmt.Printf("Duration:  %dms\n", *output.DurationMs)
	}

	if output.AgentBranch != "" {
		fmt.Printf("Branch:    %s\n", output.AgentBranch)
	}
	if output.PrUrl != "" {
		fmt.Printf("PR:        %s\n", output.PrUrl)
	}

	if output.ErrorMessage != "" {
		fmt.Println()
		fmt.Println("Error:")
		fmt.Printf("  %s\n", output.ErrorMessage)
	}

	if output.Result != nil && len(output.Result) > 0 {
		fmt.Println()
		fmt.Println("Result:")
		data, _ := json.MarshalIndent(output.Result, "  ", "  ")
		fmt.Printf("  %s\n", string(data))
	}

	fmt.Println()
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
