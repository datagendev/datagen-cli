package auth

import "testing"

func TestExtractEnvVarFromDatagenBlock_POSIX(t *testing.T) {
	block, err := RenderProfileBlock(ShellZsh, "DATAGEN_API_KEY", "abc'def")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	got, ok := extractEnvVarFromDatagenBlock(block, ShellZsh, "DATAGEN_API_KEY")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got != "abc'def" {
		t.Fatalf("got %q, want %q", got, "abc'def")
	}
}

func TestExtractEnvVarFromDatagenBlock_Fish(t *testing.T) {
	value := `a"b$c\`
	block, err := RenderProfileBlock(ShellFish, "DATAGEN_API_KEY", value)
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	got, ok := extractEnvVarFromDatagenBlock(block, ShellFish, "DATAGEN_API_KEY")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got != value {
		t.Fatalf("got %q, want %q", got, value)
	}
}

func TestExtractEnvVarFromDatagenBlock_PowerShell(t *testing.T) {
	block, err := RenderProfileBlock(ShellPowerShell, "DATAGEN_API_KEY", "ab'cd")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	got, ok := extractEnvVarFromDatagenBlock(block, ShellPowerShell, "DATAGEN_API_KEY")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got != "ab'cd" {
		t.Fatalf("got %q, want %q", got, "ab'cd")
	}
}
