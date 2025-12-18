package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVarFlags(t *testing.T) {
	t.Setenv("FROM_ENV", "env-value")

	got, err := parseVarFlags([]string{
		"FOO=bar",
		"FROM_ENV",
		"EMPTY_ENV",
		"  SPACED = spaced-value  ",
	}, map[string]string{})
	if err != nil {
		t.Fatalf("parseVarFlags error: %v", err)
	}

	if got["FOO"].Value != "bar" || got["FOO"].Source != "flag" {
		t.Fatalf("FOO=%+v; want Value=bar Source=flag", got["FOO"])
	}
	if got["FROM_ENV"].Value != "env-value" || got["FROM_ENV"].Source != "env" {
		t.Fatalf("FROM_ENV=%+v; want Value=env-value Source=env", got["FROM_ENV"])
	}
	if _, ok := got["EMPTY_ENV"]; ok {
		t.Fatalf("EMPTY_ENV should not be set")
	}
	if got["SPACED"].Value != " spaced-value" || got["SPACED"].Source != "flag" {
		t.Fatalf("SPACED=%+v; want Value=' spaced-value' Source=flag", got["SPACED"])
	}
}

func TestParseVarFlags_Invalid(t *testing.T) {
	_, err := parseVarFlags([]string{"=nope"}, map[string]string{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseVarFlags_PrefersDotEnvOverProcessEnv(t *testing.T) {
	t.Setenv("PICK_ME", "from-process-env")

	got, err := parseVarFlags([]string{"PICK_ME"}, map[string]string{"PICK_ME": "from-dotenv"})
	if err != nil {
		t.Fatalf("parseVarFlags error: %v", err)
	}
	if got["PICK_ME"].Value != "from-dotenv" || got["PICK_ME"].Source != ".env" {
		t.Fatalf("PICK_ME=%+v; want Value=from-dotenv Source=.env", got["PICK_ME"])
	}
}

func TestRequiredKeysFromEnvExample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.example")
	content := `# Required
ANTHROPIC_API_KEY=your-anthropic-api-key-here
DATAGEN_API_KEY=your-datagen-api-key-here

# Optional
FOO=bar
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write env example: %v", err)
	}

	keys := requiredKeysFromEnvExample(path)
	if len(keys) != 2 || keys[0] != "ANTHROPIC_API_KEY" || keys[1] != "DATAGEN_API_KEY" {
		t.Fatalf("keys=%v; want [ANTHROPIC_API_KEY DATAGEN_API_KEY]", keys)
	}
}
