package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAndClassify(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(agentsDir, name), []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("datagen_only.md", `---
name: datagen-only
description: Uses only datagen MCP
tools:
  - datagen
---

hi
`)

	write("no_mcp.md", `---
name: no-mcp
description: No tools
---

hi
`)

	write("other.md", `---
name: other
tools:
  - datagen
  - github
---
hi
`)

	write("datagen_tool_names.md", `---
name: datagen-tool-names
tools:
  - mcp__Datagen__executeTool
  - mcp__Datagen__getToolDetails
---
hi
`)

	write("no_frontmatter.md", `hello`)

	agents, err := Discover(agentsDir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	byName := map[string]Agent{}
	for _, a := range agents {
		byName[filepath.Base(a.Path)] = a
	}

	if got := byName["datagen_only.md"].Kind; got != KindDatagenOnly {
		t.Fatalf("datagen_only.md kind = %q; want %q", got, KindDatagenOnly)
	}
	if got := byName["no_mcp.md"].Kind; got != KindNoMCP {
		t.Fatalf("no_mcp.md kind = %q; want %q", got, KindNoMCP)
	}
	if got := byName["other.md"].Kind; got != KindOtherMCP {
		t.Fatalf("other.md kind = %q; want %q", got, KindOtherMCP)
	}
	if got := byName["no_frontmatter.md"].Kind; got != KindNoMCP {
		t.Fatalf("no_frontmatter.md kind = %q; want %q", got, KindNoMCP)
	}
	if got := byName["datagen_tool_names.md"].Kind; got != KindDatagenOnly {
		t.Fatalf("datagen_tool_names.md kind = %q; want %q", got, KindDatagenOnly)
	}
}
