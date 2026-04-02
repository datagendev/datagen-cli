package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current, remote string
		want            bool
	}{
		{"v0.3.0", "v0.3.1", true},
		{"v0.3.1", "v0.3.1", false},
		{"v0.4.0", "v0.3.1", false},
		{"v1.0.0", "v0.9.9", false},
		{"v0.1.0", "v1.0.0", true},
		{"v0.0.1", "v0.0.2", true},
		{"v0.0.2", "v0.0.1", false},
		// Pre-release / dirty suffixes stripped
		{"v0.3.1-2-gabcdef", "v0.3.1", false},
		{"v0.3.0-dirty", "v0.3.1", true},
		// Without v prefix
		{"0.3.0", "0.3.1", true},
		// Invalid input
		{"dev", "v0.3.1", false},
		{"v0.3.1", "dev", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got := IsNewer(tt.current, tt.remote)
		if got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.remote, got, tt.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"v1.2.3", []int{1, 2, 3}},
		{"0.3.1", []int{0, 3, 1}},
		{"v0.3.1-2-gabcdef", []int{0, 3, 1}},
		{"dev", nil},
		{"v1.2", nil},
		{"v1.2.x", nil},
	}
	for _, tt := range tests {
		got := parseSemver(tt.input)
		if tt.want == nil {
			if got != nil {
				t.Errorf("parseSemver(%q) = %v, want nil", tt.input, got)
			}
			continue
		}
		if got == nil || got[0] != tt.want[0] || got[1] != tt.want[1] || got[2] != tt.want[2] {
			t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsCI(t *testing.T) {
	// Save and clear all CI vars
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "TF_BUILD", "CODEBUILD_BUILD_ID"}
	saved := map[string]string{}
	for _, v := range ciVars {
		saved[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	if isCI() {
		t.Error("isCI() = true with no CI vars set")
	}

	os.Setenv("GITHUB_ACTIONS", "true")
	if !isCI() {
		t.Error("isCI() = false with GITHUB_ACTIONS=true")
	}
	os.Unsetenv("GITHUB_ACTIONS")
}

func TestCacheReadWrite(t *testing.T) {
	tmp := t.TempDir()
	cacheFile := filepath.Join(tmp, "update-check.json")

	c := updateCache{
		LastChecked:   time.Now().Add(-25 * time.Hour),
		LatestVersion: "v0.4.0",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var loaded updateCache
	raw, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.LatestVersion != "v0.4.0" {
		t.Errorf("LatestVersion = %q, want v0.4.0", loaded.LatestVersion)
	}
	if time.Since(loaded.LastChecked) < 24*time.Hour {
		t.Error("expected LastChecked to be older than 24h")
	}
}
