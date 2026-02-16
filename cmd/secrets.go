package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/datagendev/datagen-cli/internal/api"
	"github.com/datagendev/datagen-cli/internal/auth"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets stored in DataGen",
	Long: `Manage API keys and secrets stored in DataGen for use by your agents.

Use "datagen secrets list" to view your stored secrets.
Use "datagen secrets set KEY=VALUE" or "datagen secrets set KEY" to push secrets.`,
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored secrets",
	Run:   runSecretsList,
}

var secretsSetCmd = &cobra.Command{
	Use:   "set KEY=VALUE or set KEY",
	Short: "Create or update a secret",
	Long: `Create or update a secret in DataGen.

Examples:
  datagen secrets set OPENAI_API_KEY=sk-abc123     Set with explicit value
  datagen secrets set OPENAI_API_KEY               Read value from local environment variable`,
	Args: cobra.ExactArgs(1),
	Run:  runSecretsSet,
}

func init() {
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsSetCmd)
}

func newSecretsAPIClient() *api.Client {
	apiKey, _, ok := auth.FindEnvVarOrProfile("DATAGEN_API_KEY")
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: DATAGEN_API_KEY not found. Run 'datagen login' first.")
		os.Exit(1)
	}
	return api.NewClient(apiKey)
}

func runSecretsList(cmd *cobra.Command, args []string) {
	client := newSecretsAPIClient()

	resp, err := client.ListSecrets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Data.SecretKeys) == 0 {
		fmt.Println("No secrets found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMASKED VALUE\tPROVIDER")
	fmt.Fprintln(w, "----\t------------\t--------")
	for _, s := range resp.Data.SecretKeys {
		provider := "-"
		if s.Provider != nil && *s.Provider != "" {
			provider = *s.Provider
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.MaskedValue, provider)
	}
	w.Flush()
}

func runSecretsSet(cmd *cobra.Command, args []string) {
	arg := args[0]

	var name, value string

	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		name = arg[:idx]
		value = arg[idx+1:]
	} else {
		name = arg
		val, ok := os.LookupEnv(name)
		if !ok || val == "" {
			fmt.Fprintf(os.Stderr, "Error: environment variable %s is not set\n", name)
			os.Exit(1)
		}
		value = val
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "Error: secret name cannot be empty")
		os.Exit(1)
	}
	if value == "" {
		fmt.Fprintln(os.Stderr, "Error: secret value cannot be empty")
		os.Exit(1)
	}

	client := newSecretsAPIClient()

	resp, err := client.UpsertSecret(name, value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	action := "updated"
	if resp.Created {
		action = "created"
	}
	fmt.Printf("Secret '%s' %s: %s\n", resp.Secret.Name, action, resp.Secret.MaskedValue)
}
