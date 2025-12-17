package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/datagendev/datagen-cli/internal/auth"
	"github.com/spf13/cobra"
)

var (
	loginAPIKey    string
	loginShell     string
	loginProfile   string
	loginEnvVar    string
	loginYes       bool
	loginPrintOnly bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save your DataGen API key",
	Long: `Save your DataGen API key as an environment variable.

Note: A CLI cannot modify your current shell's environment. This command writes
to your shell profile so new terminals inherit DATAGEN_API_KEY.`,
	Run: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "DataGen API key (if empty, prompts securely)")
	loginCmd.Flags().StringVar(&loginShell, "shell", "", "Shell type (bash, zsh, fish, powershell)")
	loginCmd.Flags().StringVar(&loginProfile, "profile", "", "Shell profile file to update (defaults based on shell)")
	loginCmd.Flags().StringVar(&loginEnvVar, "env", "DATAGEN_API_KEY", "Environment variable name to set")
	loginCmd.Flags().BoolVarP(&loginYes, "yes", "y", false, "Skip confirmation prompts")
	loginCmd.Flags().BoolVar(&loginPrintOnly, "print", false, "Print the command to set the env var (does not write files)")
}

func runLogin(cmd *cobra.Command, args []string) {
	envVar := strings.TrimSpace(loginEnvVar)
	if envVar == "" {
		fmt.Fprintln(os.Stderr, "Error: --env cannot be empty")
		os.Exit(1)
	}

	apiKey := strings.TrimSpace(loginAPIKey)
	if apiKey == "" {
		if err := survey.AskOne(&survey.Password{
			Message: "Enter your DataGen API key:",
		}, &apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		apiKey = strings.TrimSpace(apiKey)
	}

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key cannot be empty")
		os.Exit(1)
	}

	if existing, ok := os.LookupEnv(envVar); ok && existing != "" && existing != apiKey && !loginYes {
		overwrite := false
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("%s is already set in your current environment. Overwrite?", envVar),
			Default: false,
		}, &overwrite); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if !overwrite {
			fmt.Println("No changes made.")
			return
		}
	}

	if loginPrintOnly {
		printLoginCommand(envVar, apiKey)
		return
	}

	if runtime.GOOS == "windows" {
		persistWindowsEnvVar(envVar, apiKey)
		return
	}

	shell := auth.DetectShell(runtime.GOOS, os.Getenv("SHELL"))
	if loginShell != "" {
		parsed, ok := auth.ParseShell(loginShell)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: unsupported --shell %q (use bash, zsh, fish, powershell)\n", loginShell)
			os.Exit(1)
		}
		shell = parsed
	}

	profilePath := strings.TrimSpace(loginProfile)
	if profilePath == "" {
		var err error
		profilePath, err = auth.DefaultProfilePath(shell)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	block, err := auth.RenderProfileBlock(shell, envVar, apiKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !loginYes {
		confirm := true
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Write %s to %s?", envVar, profilePath),
			Default: true,
		}, &confirm); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if !confirm {
			fmt.Println("No changes made.")
			return
		}
	}

	if err := auth.EnsureProfileUpdated(profilePath, block); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Saved %s in %s\n", envVar, profilePath)
	if shell == auth.ShellPowerShell {
		fmt.Printf("Restart your shell or run: . %s\n", profilePath)
	} else {
		fmt.Printf("Restart your terminal or run: source %s\n", profilePath)
	}
}

func printLoginCommand(envVar string, apiKey string) {
	shell := auth.DetectShell(runtime.GOOS, os.Getenv("SHELL"))
	if loginShell != "" {
		if parsed, ok := auth.ParseShell(loginShell); ok {
			shell = parsed
		}
	}

	block, err := auth.RenderProfileBlock(shell, envVar, apiKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(strings.TrimSpace(block), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "set -g") || strings.HasPrefix(line, "$env:") {
			fmt.Println(line)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Error: could not render a login command for shell %q\n", shell)
	os.Exit(1)
}

func persistWindowsEnvVar(envVar string, apiKey string) {
	_ = os.Setenv(envVar, apiKey)

	// setx persists for future shells (not the current one).
	out, err := exec.Command("setx", envVar, apiKey).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to persist %s with setx: %v\n", envVar, err)
		_ = out // avoid printing; output may include sensitive values
		fmt.Println("You can set it manually in PowerShell with:")
		fmt.Printf("  $env:%s = <your-key>\n", envVar)
		os.Exit(1)
	}

	fmt.Printf("✅ Saved %s for future terminals (Windows user env)\n", envVar)
	fmt.Println("Restart your terminal for it to take effect.")
}
