package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/datagendev/datagen-cli/internal/api"
	"github.com/datagendev/datagen-cli/internal/customtools"
	"github.com/spf13/cobra"
)

var (
	toolCode          string
	toolFile          string
	toolDescription   string
	toolSchema        string
	toolSchemaFile    string
	toolDefaults      string
	toolDefaultsFile  string
	toolOutputs       string
	toolExpectedTools string
	toolImports       string
	toolNoAutoImports bool
	toolMCPServers    string
	toolSecrets       string
	toolPublic        bool
	toolInput         string
	toolInputFile     string
)

type deployToolOptions struct {
	Code          string
	FilePath      string
	Description   string
	SchemaJSON    string
	SchemaFile    string
	DefaultsJSON  string
	DefaultsFile  string
	Outputs       string
	ExpectedTools string
	Imports       string
	NoAutoImports bool
	MCPServers    string
	Secrets       string
	Public        bool
}

type updateToolOptions struct {
	Code           string
	FilePath       string
	Description    string
	HasDescription bool
	SchemaJSON     string
	SchemaFile     string
	DefaultsJSON   string
	DefaultsFile   string
	ExpectedTools  string
	HasTools       bool
	Imports        string
	HasImports     bool
	NoAutoImports  bool
	HasNoAuto      bool
	MCPServers     string
	HasMCPServers  bool
	Secrets        string
	HasSecrets     bool
	Public         bool
	HasPublic      bool
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage DataGen custom tools",
	Long: `Deploy, inspect, update, and run DataGen custom tools.

Custom tools are Python workflows deployed as reusable API endpoints on DataGen.`,
}

var toolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List custom tools",
	RunE:  runToolsList,
}

var toolsShowCmd = &cobra.Command{
	Use:   "show <tool-uuid>",
	Short: "Show custom tool details",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsShow,
}

var toolsDeployCmd = &cobra.Command{
	Use:   "deploy <name>",
	Short: "Deploy a custom tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsDeploy,
}

var toolsUpdateCmd = &cobra.Command{
	Use:   "update <tool-uuid>",
	Short: "Update a custom tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsUpdate,
}

var toolsRunCmd = &cobra.Command{
	Use:   "run <tool-uuid>",
	Short: "Run a custom tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsRun,
}

func init() {
	addToolSourceFlags(toolsDeployCmd)
	addToolSchemaFlags(toolsDeployCmd)
	addToolMetadataFlags(toolsDeployCmd, true)

	addToolSourceFlags(toolsUpdateCmd)
	addToolSchemaFlags(toolsUpdateCmd)
	addToolMetadataFlags(toolsUpdateCmd, false)

	toolsRunCmd.Flags().StringVar(&toolInput, "input", "", "Inline JSON object for input_vars")
	toolsRunCmd.Flags().StringVar(&toolInputFile, "input-file", "", "Path to a JSON file for input_vars")

	toolsCmd.AddCommand(toolsListCmd)
	toolsCmd.AddCommand(toolsShowCmd)
	toolsCmd.AddCommand(toolsDeployCmd)
	toolsCmd.AddCommand(toolsUpdateCmd)
	toolsCmd.AddCommand(toolsRunCmd)
}

func addToolSourceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&toolCode, "code", "", "Inline Python code")
	cmd.Flags().StringVar(&toolFile, "file", "", "Path to a Python source file")
}

func addToolSchemaFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&toolSchema, "schema", "", "Inline JSON schema for input_schema")
	cmd.Flags().StringVar(&toolSchemaFile, "schema-file", "", "Path to a JSON file for input_schema")
	cmd.Flags().StringVar(&toolDefaults, "defaults", "", "Inline JSON object for default_input_vars")
	cmd.Flags().StringVar(&toolDefaultsFile, "defaults-file", "", "Path to a JSON file for default_input_vars")
}

func addToolMetadataFlags(cmd *cobra.Command, includeOutputs bool) {
	cmd.Flags().StringVar(&toolDescription, "description", "", "Tool description")
	if includeOutputs {
		cmd.Flags().StringVar(&toolOutputs, "outputs", "", "Comma-separated or newline-separated output variable names")
	}
	cmd.Flags().StringVar(&toolExpectedTools, "tools", "", "Comma-separated or newline-separated expected MCP tool names")
	cmd.Flags().StringVar(&toolImports, "imports", "", "Comma-separated or newline-separated extra Python package imports")
	cmd.Flags().BoolVar(&toolNoAutoImports, "no-auto-imports", false, "Disable third-party import inference from the Python file")
	cmd.Flags().StringVar(&toolMCPServers, "mcp-servers", "", "Comma-separated or newline-separated MCP server names")
	cmd.Flags().StringVar(&toolSecrets, "secrets", "", "Comma-separated or newline-separated secret names")
	cmd.Flags().BoolVar(&toolPublic, "public", false, "Deploy or update the tool as public")
}

func runToolsList(cmd *cobra.Command, args []string) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	fmt.Println("🧰 Fetching custom tools...")

	resp, err := client.ListCustomTools(100)
	if err != nil {
		return err
	}

	if len(resp.Data) == 0 {
		fmt.Println("\nNo custom tools found.")
		fmt.Println("Create one with: datagen tools deploy <name> --file script.py")
		return nil
	}

	fmt.Printf("\n📋 Custom tools (%d):\n\n", len(resp.Data))
	for _, tool := range resp.Data {
		fmt.Printf("%s %s\n", formatCustomToolVisibility(tool.DeploymentType), customToolName(tool))
		fmt.Printf("   UUID: %s\n", tool.DeploymentUUID)
		if strings.TrimSpace(tool.Description) != "" {
			desc := strings.TrimSpace(tool.Description)
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			fmt.Printf("   %s\n", desc)
		}
		fmt.Println()
	}

	fmt.Println("Use 'datagen tools show <tool-uuid>' for details.")
	return nil
}

func runToolsShow(cmd *cobra.Command, args []string) error {
	toolUUID := args[0]

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	fmt.Printf("🔍 Fetching custom tool: %s\n", toolUUID)

	resp, err := client.GetCustomTool(toolUUID)
	if err != nil {
		return err
	}

	tool := resp.Data
	fmt.Println()
	fmt.Printf("🧰 Tool: %s\n", customToolName(api.CustomToolSummary{
		DeploymentUUID: tool.DeploymentUUID,
		FlowName:       tool.FlowName,
		Name:           tool.Name,
	}))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("UUID:        %s\n", tool.DeploymentUUID)
	fmt.Printf("Visibility:  %s\n", describeCustomToolVisibility(tool.DeploymentType))
	if strings.TrimSpace(tool.Description) != "" {
		fmt.Printf("Description: %s\n", strings.TrimSpace(tool.Description))
	}
	if strings.TrimSpace(tool.CreatedAt) != "" {
		fmt.Printf("Created:     %s\n", tool.CreatedAt)
	}
	if strings.TrimSpace(tool.UpdatedAt) != "" {
		fmt.Printf("Updated:     %s\n", tool.UpdatedAt)
	}

	fmt.Println()
	fmt.Println("Dependencies:")
	fmt.Printf("  Imports: %s\n", formatStringList(tool.AdditionalImports))
	fmt.Printf("  Tools:   %s\n", formatStringList(tool.ExpectedTools))
	fmt.Printf("  Secrets: %s\n", formatStringList(tool.RequiredSecrets))
	fmt.Printf("  MCP:     %s\n", formatMCPServerNames(tool.MCPConfigs))

	fmt.Println()
	fmt.Println("Defaults:")
	fmt.Println(indentMultiline(prettyJSON(tool.DefaultInputVars), "  "))

	fmt.Println()
	fmt.Println("Input Schema:")
	fmt.Println(indentMultiline(prettyJSON(tool.InputSchema), "  "))

	fmt.Println()
	fmt.Println("Output Schema:")
	fmt.Println(indentMultiline(prettyJSON(tool.OutputSchema), "  "))

	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Printf("  Sync:  %s/apps/%s\n", client.BaseURL, tool.DeploymentUUID)
	fmt.Printf("  Async: %s/apps/%s/async\n", client.BaseURL, tool.DeploymentUUID)
	return nil
}

func runToolsDeploy(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	req, err := buildDeployCustomToolRequest(name, deployToolOptions{
		Code:          toolCode,
		FilePath:      toolFile,
		Description:   toolDescription,
		SchemaJSON:    toolSchema,
		SchemaFile:    toolSchemaFile,
		DefaultsJSON:  toolDefaults,
		DefaultsFile:  toolDefaultsFile,
		Outputs:       toolOutputs,
		ExpectedTools: toolExpectedTools,
		Imports:       toolImports,
		NoAutoImports: toolNoAutoImports,
		MCPServers:    toolMCPServers,
		Secrets:       toolSecrets,
		Public:        toolPublic,
	})
	if err != nil {
		return err
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Deploying custom tool: %s\n", name)

	resp, err := client.DeployCustomTool(req)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✅ Custom tool deployed successfully!")
	fmt.Printf("   UUID: %s\n", resp.Data.DeploymentUUID)
	if resp.Data.Status != "" {
		fmt.Printf("   Status: %s\n", resp.Data.Status)
	}
	fmt.Println()
	fmt.Printf("Show details: datagen tools show %s\n", resp.Data.DeploymentUUID)
	fmt.Printf("Run tool:     datagen tools run %s\n", resp.Data.DeploymentUUID)
	return nil
}

func runToolsUpdate(cmd *cobra.Command, args []string) error {
	toolUUID := args[0]

	req, hasChanges, err := buildUpdateCustomToolRequest(cmd, updateToolOptions{
		Code:           toolCode,
		FilePath:       toolFile,
		Description:    toolDescription,
		HasDescription: cmd.Flags().Changed("description"),
		SchemaJSON:     toolSchema,
		SchemaFile:     toolSchemaFile,
		DefaultsJSON:   toolDefaults,
		DefaultsFile:   toolDefaultsFile,
		ExpectedTools:  toolExpectedTools,
		HasTools:       cmd.Flags().Changed("tools"),
		Imports:        toolImports,
		HasImports:     cmd.Flags().Changed("imports"),
		NoAutoImports:  toolNoAutoImports,
		HasNoAuto:      cmd.Flags().Changed("no-auto-imports"),
		MCPServers:     toolMCPServers,
		HasMCPServers:  cmd.Flags().Changed("mcp-servers"),
		Secrets:        toolSecrets,
		HasSecrets:     cmd.Flags().Changed("secrets"),
		Public:         toolPublic,
		HasPublic:      cmd.Flags().Changed("public"),
	})
	if err != nil {
		return err
	}
	if !hasChanges {
		return fmt.Errorf("no update flags provided")
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	fmt.Printf("✏️  Updating custom tool: %s\n", toolUUID)

	resp, err := client.UpdateCustomTool(toolUUID, req)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✅ Custom tool updated successfully!")
	fmt.Printf("   UUID: %s\n", resp.Data.DeploymentUUID)
	fmt.Println()
	fmt.Printf("Show details: datagen tools show %s\n", toolUUID)
	fmt.Printf("Run tool:     datagen tools run %s\n", toolUUID)
	return nil
}

func runToolsRun(cmd *cobra.Command, args []string) error {
	toolUUID := args[0]

	inputVars, err := parseRunInput(toolInput, toolInputFile)
	if err != nil {
		return err
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	fmt.Printf("🔐 Validating custom tool requirements: %s\n", toolUUID)

	validateResp, err := client.ValidateCustomTool(toolUUID)
	if err != nil {
		return err
	}

	// Backend returns is_ready; accept either is_ready or is_valid for compatibility
	isReady := validateResp.Data.IsReady || validateResp.Data.IsValid
	if !isReady {
		fmt.Println()
		fmt.Println("❌ Custom tool is not ready to run.")
		if len(validateResp.Data.MissingRequirements.EnvironmentVariables) > 0 {
			fmt.Printf("   Missing environment variables: %s\n", strings.Join(validateResp.Data.MissingRequirements.EnvironmentVariables, ", "))
		}
		// Check both missing_requirements.secrets and top-level missing_secrets
		missingSecrets := validateResp.Data.MissingRequirements.Secrets
		if len(missingSecrets) == 0 {
			missingSecrets = validateResp.Data.MissingSecrets
		}
		if len(missingSecrets) > 0 {
			fmt.Printf("   Missing secrets: %s\n", strings.Join(missingSecrets, ", "))
			fmt.Println("   Set them with: datagen secrets set KEY=VALUE")
		}
		if len(validateResp.Data.MissingRequirements.OAuthProviders) > 0 {
			fmt.Printf("   Missing OAuth/MCP providers: %s\n", strings.Join(validateResp.Data.MissingRequirements.OAuthProviders, ", "))
		}
		for _, step := range validateResp.Data.NextSteps {
			fmt.Printf("   Next: %s\n", step)
		}
		return fmt.Errorf("custom tool is not ready to run")
	}

	fmt.Printf("▶️  Running custom tool: %s\n", toolUUID)

	runResp, err := client.RunCustomTool(toolUUID, inputVars)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✅ Custom tool run started!")
	if runResp.Data.RunUUID != "" {
		fmt.Printf("   Run UUID: %s\n", runResp.Data.RunUUID)
	}
	if runResp.Data.ExecutionUUID != "" {
		fmt.Printf("   Execution UUID: %s\n", runResp.Data.ExecutionUUID)
	}
	if runResp.Data.Status != "" {
		fmt.Printf("   Status: %s\n", runResp.Data.Status)
	}
	if runResp.Data.Message != "" {
		fmt.Printf("   Message: %s\n", runResp.Data.Message)
	}
	fmt.Println("   Inspect the run in the DataGen UI or with the returned run UUID.")
	return nil
}

func buildDeployCustomToolRequest(name string, opts deployToolOptions) (api.DeployCustomToolRequest, error) {
	code, err := customtools.ResolveCode(opts.Code, opts.FilePath, true)
	if err != nil {
		return api.DeployCustomToolRequest{}, err
	}

	inputSchema, _, err := customtools.ParseJSONObject(opts.SchemaJSON, opts.SchemaFile)
	if err != nil {
		return api.DeployCustomToolRequest{}, fmt.Errorf("schema: %w", err)
	}

	defaults, _, err := customtools.ParseJSONObject(opts.DefaultsJSON, opts.DefaultsFile)
	if err != nil {
		return api.DeployCustomToolRequest{}, fmt.Errorf("defaults: %w", err)
	}

	expectedTools := customtools.ParseList(opts.ExpectedTools)
	imports := resolveToolImports(code, opts.FilePath, opts.Imports, opts.NoAutoImports)

	req := api.DeployCustomToolRequest{
		Name:              name,
		Description:       strings.TrimSpace(opts.Description),
		FinalCode:         code,
		InputSchema:       inputSchema,
		OutputVarsList:    customtools.ParseList(opts.Outputs),
		ExpectedTools:     expectedTools,
		AdditionalImports: imports,
		DeploymentType:    deploymentTypeFromPublic(opts.Public),
		DefaultInputVars:  defaults,
		MCPServerNames:    customtools.ParseList(opts.MCPServers),
		MCPToolNames:      filterMCPToolNames(expectedTools),
		RequiredSecrets:   customtools.ParseList(opts.Secrets),
	}

	return req, nil
}

func buildUpdateCustomToolRequest(cmd *cobra.Command, opts updateToolOptions) (api.UpdateCustomToolRequest, bool, error) {
	var req api.UpdateCustomToolRequest
	var hasChanges bool

	codeProvided := strings.TrimSpace(opts.Code) != "" || strings.TrimSpace(opts.FilePath) != ""
	if codeProvided {
		code, err := customtools.ResolveCode(opts.Code, opts.FilePath, false)
		if err != nil {
			return req, false, err
		}
		req.FinalCode = &code
		hasChanges = true
	}

	if opts.HasDescription {
		description := strings.TrimSpace(opts.Description)
		req.Description = &description
		hasChanges = true
	}

	if strings.TrimSpace(opts.SchemaJSON) != "" || strings.TrimSpace(opts.SchemaFile) != "" {
		inputSchema, _, err := customtools.ParseJSONObject(opts.SchemaJSON, opts.SchemaFile)
		if err != nil {
			return req, false, fmt.Errorf("schema: %w", err)
		}
		req.InputSchema = inputSchema
		hasChanges = true
	}

	if strings.TrimSpace(opts.DefaultsJSON) != "" || strings.TrimSpace(opts.DefaultsFile) != "" {
		defaults, _, err := customtools.ParseJSONObject(opts.DefaultsJSON, opts.DefaultsFile)
		if err != nil {
			return req, false, fmt.Errorf("defaults: %w", err)
		}
		req.DefaultInputVars = defaults
		hasChanges = true
	}

	if opts.HasTools {
		req.ExpectedTools = customtools.ParseList(opts.ExpectedTools)
		hasChanges = true
	}

	if opts.HasMCPServers {
		req.MCPServerNames = customtools.ParseList(opts.MCPServers)
		hasChanges = true
	}

	if opts.HasSecrets {
		req.RequiredSecrets = customtools.ParseList(opts.Secrets)
		hasChanges = true
	}

	if opts.HasPublic {
		deploymentType := deploymentTypeFromPublic(opts.Public)
		req.DeploymentType = &deploymentType
		hasChanges = true
	}

	if codeProvided || opts.HasImports || opts.HasNoAuto {
		codeForImports := opts.Code
		if strings.TrimSpace(codeForImports) == "" && strings.TrimSpace(opts.FilePath) != "" {
			resolvedCode, err := customtools.ResolveCode(opts.Code, opts.FilePath, false)
			if err != nil {
				return req, false, err
			}
			codeForImports = resolvedCode
		}
		req.AdditionalImports = resolveToolImports(codeForImports, opts.FilePath, opts.Imports, opts.NoAutoImports)
		hasChanges = true
	}

	return req, hasChanges, nil
}

func parseRunInput(inputJSON, inputFile string) (map[string]interface{}, error) {
	inputVars, provided, err := customtools.ParseJSONObject(inputJSON, inputFile)
	if err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}
	if !provided {
		return map[string]interface{}{}, nil
	}
	return inputVars, nil
}

func resolveToolImports(code, filePath, explicitImports string, noAutoImports bool) []string {
	imports := customtools.ParseList(explicitImports)
	if noAutoImports {
		return imports
	}
	inferred := customtools.InferThirdPartyImports(code, filePath)
	return customtools.DedupSorted(append(imports, inferred...))
}

func filterMCPToolNames(expectedTools []string) []string {
	if len(expectedTools) == 0 {
		return nil
	}
	mcpToolNames := make([]string, 0, len(expectedTools))
	for _, tool := range expectedTools {
		if strings.HasPrefix(tool, "mcp_") {
			mcpToolNames = append(mcpToolNames, tool)
		}
	}
	return customtools.DedupSorted(mcpToolNames)
}

func deploymentTypeFromPublic(public bool) int {
	if public {
		return 1
	}
	return 0
}

func customToolName(tool api.CustomToolSummary) string {
	if strings.TrimSpace(tool.FlowName) != "" {
		return strings.TrimSpace(tool.FlowName)
	}
	return strings.TrimSpace(tool.Name)
}

func formatCustomToolVisibility(deploymentType *int) string {
	if deploymentType != nil && *deploymentType == 1 {
		return "🌍"
	}
	return "🔒"
}

func describeCustomToolVisibility(deploymentType *int) string {
	if deploymentType != nil && *deploymentType == 1 {
		return "public"
	}
	return "private"
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func formatMCPServerNames(configs []api.MCPConfigSummary) string {
	if len(configs) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(configs))
	for _, cfg := range configs {
		if strings.TrimSpace(cfg.Name) == "" {
			continue
		}
		names = append(names, strings.TrimSpace(cfg.Name))
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(customtools.DedupSorted(names), ", ")
}

func prettyJSON(value interface{}) string {
	if value == nil {
		return "{}"
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func indentMultiline(value, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
