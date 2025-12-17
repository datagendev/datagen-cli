package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/datagendev/datagen-cli/internal/auth"
	"github.com/datagendev/datagen-cli/internal/mcpconfig"
	"github.com/spf13/cobra"
)

var (
	mcpClients     string
	mcpAPIKey      string
	mcpEnvVar      string
	mcpYes         bool
	mcpDryRun      bool
	mcpCodexStatic bool
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Configure DataGen MCP in local tools",
	Long: `Configure the DataGen MCP server in supported local tools if their config files exist:
- Codex (~/.codex/config.toml)
- Claude (~/.claude.json)
- Gemini (~/.gemini/settings.json)
- MCP JSON (~/.mcp.json)`,
	Run: runMCP,
}

func init() {
	mcpCmd.Flags().StringVar(&mcpClients, "clients", "codex,claude,gemini,mcp", "Comma-separated clients to configure (codex, claude, gemini, mcp)")
	mcpCmd.Flags().StringVar(&mcpAPIKey, "api-key", "", "DataGen API key (if empty, uses env/profile lookup or prompts when needed)")
	mcpCmd.Flags().StringVar(&mcpEnvVar, "env", "DATAGEN_API_KEY", "Environment variable name to look up for the API key")
	mcpCmd.Flags().BoolVarP(&mcpYes, "yes", "y", false, "Skip confirmation prompts")
	mcpCmd.Flags().BoolVar(&mcpDryRun, "dry-run", false, "Show what would change without writing files")
	mcpCmd.Flags().BoolVar(&mcpCodexStatic, "codex-static", false, "Write a static x-api-key header in Codex config (default uses env_http_headers)")
}

func runMCP(cmd *cobra.Command, args []string) {
	selected := parseCSVSet(mcpClients)
	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, "Error: --clients cannot be empty")
		os.Exit(1)
	}

	var didAnything bool

	if selected["codex"] {
		// Defer until after we resolve API key (if codex-static is enabled).
	}

	apiKeyNeeded := selected["claude"] || selected["gemini"] || selected["mcp"] || (selected["codex"] && mcpCodexStatic)
	apiKey := ""
	if apiKeyNeeded {
		apiKey = mustResolveAPIKey()
	}

	if selected["codex"] {
		changed, ok, err := configureCodex(apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Codex: %v\n", err)
			os.Exit(1)
		}
		if ok {
			didAnything = didAnything || changed
		}
	}

	if selected["claude"] {
		changed, ok, err := configureClaude(apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Claude: %v\n", err)
			os.Exit(1)
		}
		if ok {
			didAnything = didAnything || changed
		}
	}

	if selected["gemini"] {
		changed, ok, err := configureGemini(apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Gemini: %v\n", err)
			os.Exit(1)
		}
		if ok {
			didAnything = didAnything || changed
		}
	}

	if selected["mcp"] {
		changed, ok, err := configureMCPJSON(apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "MCP JSON: %v\n", err)
			os.Exit(1)
		}
		if ok {
			didAnything = didAnything || changed
		}
	}

	if !didAnything {
		fmt.Println("No changes needed.")
	}
}

func configureCodex(apiKey string) (changed bool, fileExists bool, err error) {
	path, err := mcpconfig.CodexConfigPath()
	if err != nil {
		return false, false, err
	}
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			fmt.Printf("Codex: skipped (missing %s)\n", path)
			return false, false, nil
		}
		return false, false, statErr
	}

	useEnv := !mcpCodexStatic

	if mcpDryRun {
		data, err := os.ReadFile(path)
		if err != nil {
			return false, true, err
		}
		_, changed, err := mcpconfig.UpdateCodexConfig(string(data), apiKey, useEnv, strings.TrimSpace(mcpEnvVar))
		if err != nil {
			return false, true, err
		}
		if changed {
			fmt.Printf("Codex: would update %s\n", path)
		} else {
			fmt.Printf("Codex: already configured (%s)\n", path)
		}
		return changed, true, nil
	}

	if !mcpYes {
		confirm := true
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Update Codex config at %s?", path),
			Default: true,
		}, &confirm); err != nil {
			return false, true, err
		}
		if !confirm {
			fmt.Printf("Codex: skipped (%s)\n", path)
			return false, true, nil
		}
	}

	changed, err = mcpconfig.UpdateCodexConfigFile(path, apiKey, useEnv, strings.TrimSpace(mcpEnvVar))
	if err != nil {
		return false, true, err
	}
	if changed {
		fmt.Printf("Codex: updated %s\n", path)
	} else {
		fmt.Printf("Codex: already configured (%s)\n", path)
	}
	return changed, true, nil
}

func configureClaude(apiKey string) (changed bool, fileExists bool, err error) {
	path, err := mcpconfig.ClaudeConfigPath()
	if err != nil {
		return false, false, err
	}
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			legacy, err := mcpconfig.ClaudeConfigPathLegacy()
			if err == nil {
				if _, legacyStat := os.Stat(legacy); legacyStat == nil {
					path = legacy
				} else if os.IsNotExist(legacyStat) {
					fmt.Printf("Claude: skipped (missing %s)\n", mcpconfigPathHint(path, legacy))
					return false, false, nil
				} else {
					return false, false, legacyStat
				}
			} else {
				fmt.Printf("Claude: skipped (missing %s)\n", path)
				return false, false, nil
			}
		} else {
			return false, false, statErr
		}
	}

	if mcpDryRun {
		data, err := os.ReadFile(path)
		if err != nil {
			return false, true, err
		}
		_, changed, err := mcpconfig.UpdateClaudeConfig(string(data), apiKey)
		if err != nil {
			return false, true, err
		}
		if changed {
			fmt.Printf("Claude: would update %s\n", path)
		} else {
			fmt.Printf("Claude: already configured (%s)\n", path)
		}
		return changed, true, nil
	}

	if !mcpYes {
		confirm := true
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Update Claude config at %s? (stores API key in the file)", path),
			Default: true,
		}, &confirm); err != nil {
			return false, true, err
		}
		if !confirm {
			fmt.Printf("Claude: skipped (%s)\n", path)
			return false, true, nil
		}
	}

	changed, err = mcpconfig.UpdateClaudeConfigFile(path, apiKey)
	if err != nil {
		return false, true, err
	}
	if changed {
		fmt.Printf("Claude: updated %s\n", path)
	} else {
		fmt.Printf("Claude: already configured (%s)\n", path)
	}
	return changed, true, nil
}

func mcpconfigPathHint(primary string, legacy string) string {
	// keep message short but informative
	return primary + " or " + legacy
}

func configureGemini(apiKey string) (changed bool, fileExists bool, err error) {
	path, err := mcpconfig.GeminiConfigPath()
	if err != nil {
		return false, false, err
	}
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			fmt.Printf("Gemini: skipped (missing %s)\n", path)
			return false, false, nil
		}
		return false, false, statErr
	}

	if mcpDryRun {
		data, err := os.ReadFile(path)
		if err != nil {
			return false, true, err
		}
		_, changed, err := mcpconfig.UpdateGeminiConfig(string(data), apiKey)
		if err != nil {
			return false, true, err
		}
		if changed {
			fmt.Printf("Gemini: would update %s\n", path)
		} else {
			fmt.Printf("Gemini: already configured (%s)\n", path)
		}
		return changed, true, nil
	}

	if !mcpYes {
		confirm := true
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Update Gemini config at %s? (stores API key in the file)", path),
			Default: true,
		}, &confirm); err != nil {
			return false, true, err
		}
		if !confirm {
			fmt.Printf("Gemini: skipped (%s)\n", path)
			return false, true, nil
		}
	}

	changed, err = mcpconfig.UpdateGeminiConfigFile(path, apiKey)
	if err != nil {
		return false, true, err
	}
	if changed {
		fmt.Printf("Gemini: updated %s\n", path)
	} else {
		fmt.Printf("Gemini: already configured (%s)\n", path)
	}
	return changed, true, nil
}

func configureMCPJSON(apiKey string) (changed bool, fileExists bool, err error) {
	path, err := mcpconfig.MCPJSONConfigPath()
	if err != nil {
		return false, false, err
	}
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			fmt.Printf("MCP JSON: skipped (missing %s)\n", path)
			return false, false, nil
		}
		return false, false, statErr
	}

	if mcpDryRun {
		data, err := os.ReadFile(path)
		if err != nil {
			return false, true, err
		}
		_, changed, err := mcpconfig.UpdateMCPJSONConfig(string(data), apiKey)
		if err != nil {
			return false, true, err
		}
		if changed {
			fmt.Printf("MCP JSON: would update %s\n", path)
		} else {
			fmt.Printf("MCP JSON: already configured (%s)\n", path)
		}
		return changed, true, nil
	}

	if !mcpYes {
		confirm := true
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Update MCP config at %s? (stores API key in the file)", path),
			Default: true,
		}, &confirm); err != nil {
			return false, true, err
		}
		if !confirm {
			fmt.Printf("MCP JSON: skipped (%s)\n", path)
			return false, true, nil
		}
	}

	changed, err = mcpconfig.UpdateMCPJSONConfigFile(path, apiKey)
	if err != nil {
		return false, true, err
	}
	if changed {
		fmt.Printf("MCP JSON: updated %s\n", path)
	} else {
		fmt.Printf("MCP JSON: already configured (%s)\n", path)
	}
	return changed, true, nil
}

func mustResolveAPIKey() string {
	if strings.TrimSpace(mcpAPIKey) != "" {
		return strings.TrimSpace(mcpAPIKey)
	}

	if v, _, ok := auth.FindEnvVarOrProfile(mcpEnvVar); ok {
		return strings.TrimSpace(v)
	}

	if mcpYes {
		fmt.Fprintf(os.Stderr, "Error: could not find %s in environment/profile; pass --api-key or run 'datagen login' then restart your terminal\n", mcpEnvVar)
		os.Exit(1)
	}

	var apiKey string
	if err := survey.AskOne(&survey.Password{
		Message: "Enter your DataGen API key:",
	}, &apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key cannot be empty")
		os.Exit(1)
	}
	return apiKey
}

func parseCSVSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		if p == "" {
			continue
		}
		out[p] = true
	}
	return out
}
