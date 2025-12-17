package mcpconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DatagenMCPURL = "https://mcp.datagen.dev/mcp"
)

func CodexConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func ClaudeConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

func GeminiConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "settings.json"), nil
}

func ClaudeConfigPathLegacy() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json.local"), nil
}

func UpdateCodexConfigFile(path string, apiKey string, useEnvHeaders bool, envVarName string) (bool, error) {
	contents, mode, err := readFileWithMode(path)
	if err != nil {
		return false, err
	}

	updated, changed, err := UpdateCodexConfig(contents, apiKey, useEnvHeaders, envVarName)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	return true, writeFileAtomic(path, []byte(updated), mode)
}

func UpdateCodexConfig(contents string, apiKey string, useEnvHeaders bool, envVarName string) (string, bool, error) {
	if useEnvHeaders && strings.TrimSpace(envVarName) == "" {
		return "", false, errors.New("env var name is required for env_http_headers")
	}
	if !useEnvHeaders && strings.TrimSpace(apiKey) == "" {
		return "", false, errors.New("api key is required for http_headers")
	}

	original := contents
	contents = ensureFeaturesRmcpClientTrue(contents)
	contents = upsertTomlTable(contents, "mcp_servers.datagen", renderCodexDatagenTable(apiKey, useEnvHeaders, envVarName))

	if !strings.HasSuffix(contents, "\n") {
		contents += "\n"
	}

	return contents, contents != original, nil
}

func renderCodexDatagenTable(apiKey string, useEnvHeaders bool, envVarName string) string {
	var headerLine string
	if useEnvHeaders {
		headerLine = fmt.Sprintf(`env_http_headers = { "x-api-key" = %q }`, envVarName)
	} else {
		headerLine = fmt.Sprintf(`http_headers = { "x-api-key" = %q }`, apiKey)
	}

	return strings.Join([]string{
		"[mcp_servers.datagen]",
		fmt.Sprintf("url = %q", DatagenMCPURL),
		headerLine,
		"",
	}, "\n")
}

func ensureFeaturesRmcpClientTrue(contents string) string {
	// If [features] exists, rewrite rmcp_client to true or insert it.
	lines := strings.Split(contents, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "[features]" {
			continue
		}
		indent := ""
		for j := i + 1; j < len(lines); j++ {
			trim := strings.TrimSpace(lines[j])
			if strings.HasPrefix(trim, "[") {
				// Insert rmcp_client before the next table.
				lines = append(lines[:j], append([]string{indent + "rmcp_client = true"}, lines[j:]...)...)
				return strings.Join(lines, "\n")
			}
			if indent == "" && lines[j] != "" {
				indent = lines[j][:len(lines[j])-len(strings.TrimLeft(lines[j], " \t"))]
			}
			if trim == "" {
				continue
			}
			if strings.HasPrefix(trim, "#") {
				continue
			}
			if strings.HasPrefix(trim, "rmcp_client") {
				prefix := lines[j][:len(lines[j])-len(strings.TrimLeft(lines[j], " \t"))]
				lines[j] = prefix + "rmcp_client = true"
				return strings.Join(lines, "\n")
			}
		}
		// Reached EOF inside features table; append.
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "rmcp_client = true")
		return strings.Join(lines, "\n")
	}

	// No [features] table; append a minimal one.
	contents = strings.TrimRight(contents, "\n")
	if contents != "" {
		contents += "\n\n"
	}
	contents += strings.Join([]string{
		"[features]",
		"rmcp_client = true",
		"",
	}, "\n")
	return contents
}

func upsertTomlTable(contents string, tableName string, desiredTable string) string {
	header := "[" + tableName + "]"
	lines := strings.Split(contents, "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			start = i
			break
		}
	}

	if start == -1 {
		contents = strings.TrimRight(contents, "\n")
		if contents != "" {
			contents += "\n\n"
		}
		return contents + strings.TrimLeft(desiredTable, "\n")
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "[") {
			end = i
			break
		}
	}

	before := strings.Join(lines[:start], "\n")
	after := strings.Join(lines[end:], "\n")
	desired := strings.TrimLeft(desiredTable, "\n")

	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	switch {
	case before == "" && after == "":
		return desired
	case before == "":
		return desired + "\n" + after
	case after == "":
		return before + "\n\n" + desired
	default:
		return before + "\n\n" + desired + "\n" + after
	}
}

func UpdateClaudeConfigFile(path string, apiKey string) (bool, error) {
	raw, mode, err := readFileWithMode(path)
	if err != nil {
		return false, err
	}

	updated, changed, err := UpdateClaudeConfig(raw, apiKey)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	return true, writeFileAtomic(path, []byte(updated), mode)
}

func UpdateClaudeConfig(contents string, apiKey string) (string, bool, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", false, errors.New("api key is required")
	}

	var root map[string]any
	if strings.TrimSpace(contents) != "" {
		if err := json.Unmarshal([]byte(contents), &root); err != nil {
			return "", false, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}
	if root == nil {
		root = map[string]any{}
	}

	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		root["mcpServers"] = servers
	}

	if claudeDatagenServerIsCurrent(servers["datagen"], apiKey) {
		return ensureTrailingNewline(contents), false, nil
	}

	// Keep a stable key ordering inside the Datagen server object; some tools are
	// picky about formatting and users prefer "headers" last.
	type claudeServer struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	encoded, err := json.Marshal(claudeServer{
		Type: "http",
		URL:  DatagenMCPURL,
		Headers: map[string]string{
			"X-API-Key": apiKey,
		},
	})
	if err != nil {
		return "", false, err
	}
	servers["datagen"] = json.RawMessage(encoded)

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", false, err
	}
	outStr := string(out) + "\n"
	return outStr, outStr != contents, nil
}

func UpdateGeminiConfigFile(path string, apiKey string) (bool, error) {
	raw, mode, err := readFileWithMode(path)
	if err != nil {
		return false, err
	}

	updated, changed, err := UpdateGeminiConfig(raw, apiKey)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	return true, writeFileAtomic(path, []byte(updated), mode)
}

func UpdateGeminiConfig(contents string, apiKey string) (string, bool, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", false, errors.New("api key is required")
	}

	var root map[string]any
	if strings.TrimSpace(contents) != "" {
		if err := json.Unmarshal([]byte(contents), &root); err != nil {
			return "", false, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}
	if root == nil {
		root = map[string]any{}
	}

	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		root["mcpServers"] = servers
	}

	if geminiDatagenServerIsCurrent(servers["datagen"], apiKey) {
		return ensureTrailingNewline(contents), false, nil
	}

	servers["datagen"] = map[string]any{
		"httpUrl": DatagenMCPURL,
		"headers": map[string]any{
			"X-API-KEY": apiKey,
		},
		"timeout": 30000,
		"trust":   false,
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", false, err
	}
	outStr := string(out) + "\n"
	return outStr, outStr != contents, nil
}

func claudeDatagenServerIsCurrent(v any, apiKey string) bool {
	switch t := v.(type) {
	case map[string]any:
		if t["type"] != "http" || t["url"] != DatagenMCPURL {
			return false
		}
		headers, _ := t["headers"].(map[string]any)
		if headers == nil {
			return false
		}
		return headers["X-API-Key"] == apiKey
	case json.RawMessage:
		var m map[string]any
		if err := json.Unmarshal(t, &m); err != nil {
			return false
		}
		return claudeDatagenServerIsCurrent(m, apiKey)
	case []byte:
		return claudeDatagenServerIsCurrent(json.RawMessage(t), apiKey)
	default:
		return false
	}
}

func geminiDatagenServerIsCurrent(v any, apiKey string) bool {
	m, _ := v.(map[string]any)
	if m == nil {
		return false
	}
	if m["httpUrl"] != DatagenMCPURL || m["timeout"] != float64(30000) || m["trust"] != false {
		return false
	}
	headers, _ := m["headers"].(map[string]any)
	if headers == nil {
		return false
	}
	return headers["X-API-KEY"] == apiKey
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func readFileWithMode(path string) (string, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	return string(data), info.Mode().Perm(), nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp := filepath.Join(dir, "."+filepath.Base(path)+".datagen.tmp")
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Windows does not allow replacing an existing file with rename.
		if removeErr := os.Remove(path); removeErr == nil {
			if renameErr := os.Rename(tmp, path); renameErr == nil {
				return nil
			}
		}
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
