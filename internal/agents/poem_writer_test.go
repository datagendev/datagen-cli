package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPoemWriterClassification(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	agentFile := filepath.Join(dir, "poem-writer.md")

	content := `---
name: poem-writer
description: An agent that writes poems using DataGen tools
tools:
  - mcp__datagen__searchtools
  - mcp__datagen__executetool
  - mcp__datagen__gettooldetails
---

You are a poem-writing agent. Use the DataGen tools to search for inspiration and write beautiful poems.
`
	if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	agent, err := parseAgentFile(agentFile)
	if err != nil {
		t.Fatalf("Failed to parse poem-writer.md: %v", err)
	}

	// Verify it has the correct tools
	expectedTools := []string{
		"mcp__datagen__searchtools",
		"mcp__datagen__executetool",
		"mcp__datagen__gettooldetails",
	}

	if len(agent.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d: %v", len(expectedTools), len(agent.Tools), agent.Tools)
	}

	for i, expected := range expectedTools {
		if i >= len(agent.Tools) {
			t.Errorf("Missing tool at index %d: expected %s", i, expected)
			continue
		}
		if agent.Tools[i] != expected {
			t.Errorf("Tool at index %d: expected %s, got %s", i, expected, agent.Tools[i])
		}
	}

	// Verify it's classified as DatagenOnly
	if agent.Kind != KindDatagenOnly {
		t.Errorf("Expected Kind=%s, got %s", KindDatagenOnly, agent.Kind)
	}

	// Verify basic metadata
	if agent.Name != "poem-writer" {
		t.Errorf("Expected Name=poem-writer, got %s", agent.Name)
	}

	if agent.Description == "" {
		t.Error("Expected non-empty description")
	}
}
