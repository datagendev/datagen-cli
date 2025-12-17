package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Shell string

const (
	ShellBash       Shell = "bash"
	ShellZsh        Shell = "zsh"
	ShellFish       Shell = "fish"
	ShellPowerShell Shell = "powershell"
)

const (
	datagenBlockStart = "# >>> datagen login >>>"
	datagenBlockEnd   = "# <<< datagen login <<<"
)

func ParseShell(s string) (Shell, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bash":
		return ShellBash, true
	case "zsh":
		return ShellZsh, true
	case "fish":
		return ShellFish, true
	case "powershell", "pwsh", "ps":
		return ShellPowerShell, true
	default:
		return "", false
	}
}

func DetectShell(goos string, shellEnv string) Shell {
	if goos == "windows" {
		return ShellPowerShell
	}

	// shellEnv is commonly an absolute path like /bin/zsh.
	base := filepath.Base(strings.TrimSpace(shellEnv))
	if sh, ok := ParseShell(base); ok {
		return sh
	}

	// Reasonable default for Unix-like systems.
	return ShellBash
}

func DefaultProfilePath(shell Shell) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home dir: %w", err)
	}

	switch shell {
	case ShellZsh:
		return filepath.Join(home, ".zshrc"), nil
	case ShellBash:
		return filepath.Join(home, ".bashrc"), nil
	case ShellFish:
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	case ShellPowerShell:
		// Prefer a cross-platform profile location rather than trying to mirror
		// each platform's shell-specific default.
		return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %q", shell)
	}
}

func RenderProfileBlock(shell Shell, envVar string, value string) (string, error) {
	if strings.TrimSpace(envVar) == "" {
		return "", errors.New("env var name is required")
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("value is required")
	}
	if strings.Contains(value, "\x00") {
		return "", errors.New("value contains NUL byte")
	}

	var line string
	switch shell {
	case ShellFish:
		line = fmt.Sprintf("set -gx %s %s", envVar, quoteFish(value))
	case ShellPowerShell:
		line = fmt.Sprintf("$env:%s = %s", envVar, quotePowerShell(value))
	case ShellBash, ShellZsh:
		line = fmt.Sprintf("export %s=%s", envVar, quotePOSIX(value))
	default:
		return "", fmt.Errorf("unsupported shell: %q", shell)
	}

	return strings.Join([]string{
		datagenBlockStart,
		line,
		datagenBlockEnd,
		"",
	}, "\n"), nil
}

func UpsertProfileBlock(existing string, block string) (string, error) {
	if !strings.Contains(block, datagenBlockStart) || !strings.Contains(block, datagenBlockEnd) {
		return "", errors.New("block is missing datagen markers")
	}

	startIdx := strings.Index(existing, datagenBlockStart)
	if startIdx == -1 {
		return appendWithNewline(existing, block), nil
	}

	endIdx := strings.Index(existing[startIdx:], datagenBlockEnd)
	if endIdx == -1 {
		// Corrupted/partial block; fall back to appending a fresh one.
		return appendWithNewline(existing, block), nil
	}
	endIdx = startIdx + endIdx + len(datagenBlockEnd)

	// Include the trailing newline after the end marker if present.
	if endIdx < len(existing) && existing[endIdx] == '\n' {
		endIdx++
	}

	updated := existing[:startIdx] + block + existing[endIdx:]
	if !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	return updated, nil
}

func EnsureProfileUpdated(profilePath string, block string) error {
	existing, mode, err := readFileWithMode(profilePath)
	if err != nil {
		return err
	}

	updated, err := UpsertProfileBlock(existing, block)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return fmt.Errorf("failed to create profile parent dir: %w", err)
	}

	return writeFileAtomic(profilePath, []byte(updated), mode)
}

func appendWithNewline(existing, block string) string {
	if existing == "" {
		return block
	}
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if strings.HasSuffix(existing, "\n\n") {
		return existing + block
	}
	return existing + "\n" + block
}

func quotePOSIX(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func quoteFish(s string) string {
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	return `"` + escaped + `"`
}

func quotePowerShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func readFileWithMode(path string) (string, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0644, nil
		}
		return "", 0, fmt.Errorf("failed to stat profile: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read profile: %w", err)
	}

	return string(data), info.Mode().Perm(), nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp := filepath.Join(dir, "."+filepath.Base(path)+".datagen.tmp")

	if err := os.WriteFile(tmp, data, mode); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		// Windows does not allow replacing an existing file with rename.
		if removeErr := os.Remove(path); removeErr == nil {
			if renameErr := os.Rename(tmp, path); renameErr == nil {
				return nil
			}
		}
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to replace profile: %w", err)
	}

	return nil
}
