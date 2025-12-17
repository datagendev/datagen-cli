package auth

import (
	"strings"
	"testing"
)

func TestRenderProfileBlock_POSIXQuoting(t *testing.T) {
	block, err := RenderProfileBlock(ShellBash, "DATAGEN_API_KEY", "abc'def")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	if !strings.Contains(block, "export DATAGEN_API_KEY='abc'\\''def'") {
		t.Fatalf("unexpected block:\n%s", block)
	}
}

func TestRenderProfileBlock_FishQuoting(t *testing.T) {
	value := `a"b$c\`
	block, err := RenderProfileBlock(ShellFish, "DATAGEN_API_KEY", value)
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	want := "set -gx DATAGEN_API_KEY " + quoteFish(value)
	if !strings.Contains(block, want) {
		t.Fatalf("expected block to contain %q, got:\n%s", want, block)
	}
}

func TestRenderProfileBlock_PowerShellQuoting(t *testing.T) {
	block, err := RenderProfileBlock(ShellPowerShell, "DATAGEN_API_KEY", "ab'cd")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	if !strings.Contains(block, "$env:DATAGEN_API_KEY = 'ab''cd'") {
		t.Fatalf("unexpected block:\n%s", block)
	}
}

func TestUpsertProfileBlock_AppendsWhenMissing(t *testing.T) {
	block, err := RenderProfileBlock(ShellZsh, "DATAGEN_API_KEY", "k1")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}

	got, err := UpsertProfileBlock("export OTHER=1\n", block)
	if err != nil {
		t.Fatalf("UpsertProfileBlock() error = %v", err)
	}
	if !strings.Contains(got, "export OTHER=1") || !strings.Contains(got, "export DATAGEN_API_KEY") {
		t.Fatalf("unexpected updated content:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected trailing newline, got %q", got[len(got)-1:])
	}
}

func TestUpsertProfileBlock_ReplacesExisting(t *testing.T) {
	block1, err := RenderProfileBlock(ShellZsh, "DATAGEN_API_KEY", "k1")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}
	block2, err := RenderProfileBlock(ShellZsh, "DATAGEN_API_KEY", "k2")
	if err != nil {
		t.Fatalf("RenderProfileBlock() error = %v", err)
	}

	existing := "export OTHER=1\n" + block1 + "export AFTER=1\n"
	got, err := UpsertProfileBlock(existing, block2)
	if err != nil {
		t.Fatalf("UpsertProfileBlock() error = %v", err)
	}
	if strings.Contains(got, "k1") {
		t.Fatalf("expected old value removed, got:\n%s", got)
	}
	if !strings.Contains(got, "k2") || !strings.Contains(got, "export AFTER=1") {
		t.Fatalf("unexpected updated content:\n%s", got)
	}
}
