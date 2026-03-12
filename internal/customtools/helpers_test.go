package customtools

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInferThirdPartyImports(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]\nname='demo'\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(pyproject.toml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "local_module.py"), []byte("VALUE = 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(local_module.py) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "localpkg"), 0o755); err != nil {
		t.Fatalf("Mkdir(localpkg) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "localpkg", "__init__.py"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile(localpkg/__init__.py) error = %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "tool.py")
	code := `
import os
import json
import requests, pandas as pd
from bs4 import BeautifulSoup
from .relative import helper
from local_module import helper
from localpkg.submodule import thing
`

	got := InferThirdPartyImports(code, scriptPath)
	want := []string{"bs4", "pandas", "requests"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("InferThirdPartyImports() = %v, want %v", got, want)
	}
}

func TestParseJSONObject(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "defaults.json")
	if err := os.WriteFile(jsonPath, []byte(`{"count":2,"enabled":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile(defaults.json) error = %v", err)
	}

	got, provided, err := ParseJSONObject("", jsonPath)
	if err != nil {
		t.Fatalf("ParseJSONObject() error = %v", err)
	}
	if !provided {
		t.Fatalf("ParseJSONObject() provided = false, want true")
	}
	if count := got["count"].(float64); count != 2 {
		t.Fatalf("ParseJSONObject() count = %v, want 2", count)
	}

	if _, _, err := ParseJSONObject(`{"a":1}`, jsonPath); err == nil {
		t.Fatalf("ParseJSONObject() error = nil, want mutual exclusivity error")
	}
}

func TestResolveCode(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "tool.py")
	if err := os.WriteFile(scriptPath, []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(tool.py) error = %v", err)
	}

	got, err := ResolveCode("", scriptPath, true)
	if err != nil {
		t.Fatalf("ResolveCode() error = %v", err)
	}
	if got != "print('hello')\n" {
		t.Fatalf("ResolveCode() = %q, want %q", got, "print('hello')\n")
	}

	if _, err := ResolveCode("", "", true); err == nil {
		t.Fatalf("ResolveCode() error = nil, want required code error")
	}
}
