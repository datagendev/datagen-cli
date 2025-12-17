package auth

import (
	"os"
	"path/filepath"
	"strings"
)

// FindEnvVarOrProfile returns the env var value either from the current process
// environment or (if missing) from a "datagen login" block in common shell
// profile files.
func FindEnvVarOrProfile(envVar string) (value string, source string, ok bool) {
	envVar = strings.TrimSpace(envVar)
	if envVar == "" {
		return "", "", false
	}

	if v, exists := os.LookupEnv(envVar); exists && strings.TrimSpace(v) != "" {
		return v, "environment", true
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", "", false
	}

	candidates := []struct {
		shell Shell
		path  string
	}{
		{shell: ShellZsh, path: filepath.Join(home, ".zshrc")},
		{shell: ShellBash, path: filepath.Join(home, ".bashrc")},
		{shell: ShellFish, path: filepath.Join(home, ".config", "fish", "config.fish")},
		{shell: ShellPowerShell, path: filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")},
	}

	for _, c := range candidates {
		data, err := os.ReadFile(c.path)
		if err != nil {
			continue
		}
		if v, found := extractEnvVarFromDatagenBlock(string(data), c.shell, envVar); found {
			return v, c.path, true
		}
	}

	return "", "", false
}

func extractEnvVarFromDatagenBlock(contents string, shell Shell, envVar string) (string, bool) {
	start := strings.Index(contents, datagenBlockStart)
	if start == -1 {
		return "", false
	}
	end := strings.Index(contents[start:], datagenBlockEnd)
	if end == -1 {
		return "", false
	}
	end = start + end

	block := contents[start:end]
	lines := strings.Split(block, "\n")

	switch shell {
	case ShellFish:
		prefix := "set -gx " + envVar + " "
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				return unquoteFish(rest)
			}
		}
	case ShellPowerShell:
		prefix := "$env:" + envVar + " = "
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				return unquotePowerShell(rest)
			}
		}
	case ShellBash, ShellZsh:
		prefix := "export " + envVar + "="
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				return unquotePOSIX(rest)
			}
		}
	}

	return "", false
}

func unquotePOSIX(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '\'' || s[len(s)-1] != '\'' {
		return "", false
	}
	inner := s[1 : len(s)-1]
	return strings.ReplaceAll(inner, "'\\''", "'"), true
}

func unquoteFish(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", false
	}
	inner := s[1 : len(s)-1]
	var b strings.Builder
	b.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if ch == '\\' && i+1 < len(inner) {
			next := inner[i+1]
			switch next {
			case '\\', '"', '$':
				b.WriteByte(next)
				i++
				continue
			}
		}
		b.WriteByte(ch)
	}
	return b.String(), true
}

func unquotePowerShell(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '\'' || s[len(s)-1] != '\'' {
		return "", false
	}
	inner := s[1 : len(s)-1]
	return strings.ReplaceAll(inner, "''", "'"), true
}
