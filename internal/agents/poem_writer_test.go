package agents

import (
	"testing"
)

func TestPoemWriterClassification(t *testing.T) {
	t.Parallel()

	// Parse the actual poem-writer.md file
	agent, err := parseAgentFile("../../.claude/agents/poem-writer.md")
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
