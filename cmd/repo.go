package cmd

import (
	"fmt"
	"os"

	"github.com/datagendev/datagen-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	repoCreateOrg         string
	repoCreatePrivate     bool
	repoCreateDescription string
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
	Long: `List, create, connect, sync, and remove GitHub repositories on DataGen.

Examples:
  datagen repo list                  List available repos
  datagen repo add owner/repo        Connect an existing repo
  datagen repo create my-agent       Create a new repo and connect it
  datagen repo sync <repo-id>        Refresh agent discovery
  datagen repo remove <repo-id>      Disconnect a repo`,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available repositories",
	Long:  `List all repositories accessible via your GitHub App installations.`,
	Run:   runRepoList,
}

var repoAddCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Connect an existing repository",
	Long:  `Connect a repository by its full name (e.g., owner/repo) to discover agents.`,
	Args:  cobra.ExactArgs(1),
	Run:   runRepoAdd,
}

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new repository and connect it",
	Long: `Create a new GitHub repository and automatically connect it to DataGen.

The repo is created using your GitHub App installation. By default it will be
private and created under your personal account. Use --org to create it under
an organization.`,
	Args: cobra.ExactArgs(1),
	Run:  runRepoCreate,
}

var repoSyncCmd = &cobra.Command{
	Use:   "sync <repo-id>",
	Short: "Sync agents in a connected repository",
	Long:  `Refresh agent discovery for a connected repository.`,
	Args:  cobra.ExactArgs(1),
	Run:   runRepoSync,
}

var repoRemoveCmd = &cobra.Command{
	Use:   "remove <repo-id>",
	Short: "Disconnect a repository",
	Long:  `Disconnect a repository from DataGen. This does not delete the GitHub repo.`,
	Args:  cobra.ExactArgs(1),
	Run:   runRepoRemove,
}

func init() {
	repoCreateCmd.Flags().StringVar(&repoCreateOrg, "org", "", "Organization to create the repo under (default: personal account)")
	repoCreateCmd.Flags().BoolVar(&repoCreatePrivate, "private", true, "Create a private repository")
	repoCreateCmd.Flags().StringVar(&repoCreateDescription, "description", "", "Repository description")

	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoSyncCmd)
	repoCmd.AddCommand(repoRemoveCmd)
}

func runRepoList(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Fetching available repositories...")

	reposResp, err := client.ListAvailableRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	totalRepos := 0
	for _, inst := range reposResp.Installations {
		totalRepos += len(inst.Repos)
	}

	if totalRepos == 0 {
		fmt.Println("\nNo repositories found.")
		fmt.Println("Run 'datagen github connect' to install the GitHub App first.")
		return
	}

	fmt.Printf("\nAvailable repositories (%d):\n\n", totalRepos)

	for _, inst := range reposResp.Installations {
		if len(inst.Repos) > 0 {
			fmt.Printf("  %s (%s)\n", inst.Installation.AccountLogin, inst.Installation.AccountType)
			for _, repo := range inst.Repos {
				visibility := "private"
				if !repo.Private {
					visibility = "public"
				}
				connected := ""
				if repo.IsConnected {
					connected = " [connected]"
				}
				fmt.Printf("    %s (%s)%s\n", repo.FullName, visibility, connected)
			}
			fmt.Println()
		}
	}

	fmt.Println("Use 'datagen repo add <owner/repo>' to connect a repository.")
}

func runRepoAdd(cmd *cobra.Command, args []string) {
	fullName := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connecting repository: %s\n", fullName)

	resp, err := client.ConnectRepo(fullName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected: %s\n", resp.Repo.FullName)
	fmt.Printf("  ID: %s\n", resp.Repo.ID)
	fmt.Printf("  Agents discovered: %d\n", resp.AgentsDiscovered)

	if resp.AgentsDiscovered > 0 {
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Run 'datagen agents list' to see discovered agents")
		fmt.Println("  2. Run 'datagen agents deploy <agent-id>' to deploy an agent")
	} else {
		fmt.Println()
		fmt.Println("No agents found in .claude/agents/ directory.")
		fmt.Printf("  Create an agent file and run 'datagen repo sync %s' to refresh.\n", resp.Repo.ID)
	}
}

func runRepoCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	visibility := "private"
	if !repoCreatePrivate {
		visibility = "public"
	}

	location := "your personal account"
	if repoCreateOrg != "" {
		location = repoCreateOrg
	}

	fmt.Printf("Creating %s repository '%s' under %s...\n", visibility, name, location)

	resp, err := client.CreateRepo(api.CreateRepoRequest{
		Name:        name,
		Org:         repoCreateOrg,
		Private:     repoCreatePrivate,
		Description: repoCreateDescription,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created and connected: %s\n", resp.Repo.FullName)
	fmt.Printf("  ID: %s\n", resp.Repo.ID)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Clone the repo and add agent files to .claude/agents/")
	fmt.Printf("  2. Run 'datagen repo sync %s' to discover agents\n", resp.Repo.ID)
	fmt.Println("  3. Run 'datagen agents deploy <agent-id>' to deploy")
}

func runRepoSync(cmd *cobra.Command, args []string) {
	repoID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Syncing repository: %s\n", repoID)

	resp, err := client.SyncRepo(repoID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sync complete!")
	fmt.Printf("  Agents found: %d\n", resp.AgentsFound)
	fmt.Printf("  New agents: %d\n", resp.NewAgents)
	fmt.Printf("  Updated agents: %d\n", resp.UpdatedAgents)
}

func runRepoRemove(cmd *cobra.Command, args []string) {
	repoID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Disconnecting repository: %s\n", repoID)

	resp, err := client.DisconnectRepo(repoID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Repository disconnected.")
	if resp.Message != "" {
		fmt.Printf("  %s\n", resp.Message)
	}
}
