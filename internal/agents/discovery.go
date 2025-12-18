package agents

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

type Kind string

const (
	KindDatagenOnly Kind = "datagen-only"
	KindNoMCP       Kind = "no-mcp"
	KindOtherMCP    Kind = "other-mcp"
)

type Agent struct {
	Path        string
	Name        string
	Description string
	Tools       []string
	Kind        Kind
}

type frontmatterMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Tools       any    `yaml:"tools"`
}

func Discover(agentsDir string) ([]Agent, error) {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, err
	}

	var agents []Agent
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		fullPath := filepath.Join(agentsDir, name)
		agent, err := parseAgentFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("parse agent %s: %w", name, err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func parseAgentFile(path string) (Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Agent{}, err
	}

	agent := Agent{
		Path: path,
		Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Kind: KindNoMCP,
	}

	meta, ok := parseFrontmatter(data)
	if !ok {
		return agent, nil
	}

	if meta.Name != "" {
		agent.Name = meta.Name
	}
	agent.Description = processDescription(meta.Description)
	agent.Tools = normalizeTools(meta.Tools)
	agent.Kind = classifyTools(agent.Tools)
	return agent, nil
}

func processDescription(desc string) string {
	// Convert literal \n escape sequences to actual newlines
	// This handles cases where Claude auto-generates descriptions with \n
	return strings.ReplaceAll(desc, "\\n", "\n")
}

func parseFrontmatter(content []byte) (frontmatterMeta, bool) {
	// Expect YAML frontmatter: ---\n...\n---\n
	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return frontmatterMeta{}, false
	}

	lines := bytes.Split(trimmed, []byte("\n"))
	if len(lines) < 3 {
		return frontmatterMeta{}, false
	}
	if !bytes.Equal(bytes.TrimSpace(lines[0]), []byte("---")) {
		return frontmatterMeta{}, false
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if bytes.Equal(bytes.TrimSpace(lines[i]), []byte("---")) {
			end = i
			break
		}
	}
	if end == -1 {
		return frontmatterMeta{}, false
	}

	// Extract frontmatter lines and wrap long unquoted strings in quotes
	frontmatterLines := lines[1:end]
	processed := preprocessYAML(frontmatterLines)

	var meta frontmatterMeta
	if err := yaml.Unmarshal(processed, &meta); err != nil {
		return frontmatterMeta{}, false
	}
	return meta, true
}

// preprocessYAML wraps long unquoted description values in quotes to help YAML parser
func preprocessYAML(lines [][]byte) []byte {
	var result [][]byte
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := bytes.TrimSpace(line)

		// Check if this is a description line with an unquoted value
		if bytes.HasPrefix(trimmed, []byte("description:")) {
			// Extract the value part after "description:"
			parts := bytes.SplitN(trimmed, []byte(":"), 2)
			if len(parts) == 2 {
				value := bytes.TrimSpace(parts[1])
				// If it doesn't start with a quote or pipe/block indicator, wrap it
				if len(value) > 0 && value[0] != '"' && value[0] != '\'' && value[0] != '|' && value[0] != '>' {
					// Wrap in quotes and escape any existing quotes
					escaped := bytes.ReplaceAll(value, []byte(`"`), []byte(`\"`))
					line = []byte(fmt.Sprintf("description: \"%s\"", escaped))
				}
			}
		}
		result = append(result, line)
	}
	return bytes.Join(result, []byte("\n"))
}

func normalizeTools(v any) []string {
	if v == nil {
		return nil
	}

	switch t := v.(type) {
	case string:
		return splitAndNormalizeTools(t)
	case []string:
		var out []string
		for _, s := range t {
			out = append(out, normalizeTool(s))
		}
		return filterEmpty(out)
	case []any:
		var out []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, normalizeTool(s))
			}
		}
		return filterEmpty(out)
	default:
		return nil
	}
}

func splitAndNormalizeTools(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, normalizeTool(p))
	}
	return filterEmpty(out)
}

func normalizeTool(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func filterEmpty(in []string) []string {
	out := in[:0]
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func classifyTools(tools []string) Kind {
	if len(tools) == 0 {
		return KindNoMCP
	}
	for _, t := range tools {
		if t == "datagen" || strings.HasPrefix(t, "mcp__datagen__") {
			continue
		}
		if t != "" {
			return KindOtherMCP
		}
	}
	return KindDatagenOnly
}
