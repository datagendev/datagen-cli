package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/datagendev/datagen-cli/internal/api"
	"github.com/datagendev/datagen-cli/internal/auth"
	"github.com/spf13/cobra"
)

var (
	githubConnectTimeout int
	githubRepoFullName   string
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Manage GitHub connection",
	Long: `Connect your GitHub account to DataGen via the GitHub App.

This allows DataGen to discover agents from your repositories and deploy them.`,
}

var githubConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Install GitHub App and connect repos",
	Long: `Opens your browser to install the DataGen GitHub App.

After installation, DataGen can access your repositories to discover
and deploy agents from .claude/agents/ directories.`,
	Run: runGitHubConnect,
}

var githubReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List available repositories",
	Long:  `List all repositories accessible via your GitHub App installations.`,
	Run:   runGitHubRepos,
}

var githubConnectedCmd = &cobra.Command{
	Use:   "connected",
	Short: "List connected repositories",
	Long:  `List repositories that are connected and being monitored for agents.`,
	Run:   runGitHubConnected,
}

var githubConnectRepoCmd = &cobra.Command{
	Use:   "connect-repo <owner/repo>",
	Short: "Connect a specific repository",
	Long:  `Connect a repository by its full name (e.g., owner/repo) to discover agents.`,
	Args:  cobra.ExactArgs(1),
	Run:   runGitHubConnectRepo,
}

var githubSyncCmd = &cobra.Command{
	Use:   "sync <repo-id>",
	Short: "Sync agents in a connected repository",
	Long:  `Refresh agent discovery for a connected repository.`,
	Args:  cobra.ExactArgs(1),
	Run:   runGitHubSync,
}

var githubStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check GitHub connection status",
	Long:  `Show the status of your GitHub App installations.`,
	Run:   runGitHubStatus,
}

func init() {
	githubConnectCmd.Flags().IntVar(&githubConnectTimeout, "timeout", 300, "Timeout in seconds to wait for GitHub App installation")

	githubCmd.AddCommand(githubConnectCmd)
	githubCmd.AddCommand(githubReposCmd)
	githubCmd.AddCommand(githubConnectedCmd)
	githubCmd.AddCommand(githubConnectRepoCmd)
	githubCmd.AddCommand(githubSyncCmd)
	githubCmd.AddCommand(githubStatusCmd)
}

func getAPIClient() (*api.Client, error) {
	apiKey, _, ok := auth.FindEnvVarOrProfile("DATAGEN_API_KEY")
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("DATAGEN_API_KEY not found. Run 'datagen login' first")
	}
	return api.NewClient(apiKey), nil
}

func runGitHubConnect(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔗 Getting GitHub App installation URL...")

	resp, err := client.GetGitHubInstallUrl()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if user already has installations
	initialResp, err := client.ListGitHubInstallations()
	initialCount := 0
	if err == nil {
		initialCount = len(initialResp.Installations)
	}

	if initialCount > 0 {
		// Existing installation(s) -- GitHub will modify rather than create new
		fmt.Printf("\nYou already have %d GitHub App installation(s).\n", initialCount)
		fmt.Printf("Opening browser to modify repository access...\n")
		fmt.Printf("   URL: %s\n\n", resp.InstallUrl)

		if err := openBrowser(resp.InstallUrl); err != nil {
			fmt.Printf("Could not open browser automatically.\n")
			fmt.Printf("   Please open this URL manually: %s\n\n", resp.InstallUrl)
		}

		fmt.Println("After updating access on GitHub, run:")
		fmt.Println("   datagen repo list             -- to see available repos")
		fmt.Println("   datagen repo add <owner/repo> -- to connect a repo")
		return
	}

	// First-time installation -- open browser and poll for new installation
	fmt.Printf("\nOpening browser to install the DataGen GitHub App...\n")
	fmt.Printf("   URL: %s\n\n", resp.InstallUrl)

	if err := openBrowser(resp.InstallUrl); err != nil {
		fmt.Printf("Could not open browser automatically.\n")
		fmt.Printf("   Please open this URL manually: %s\n\n", resp.InstallUrl)
	}

	fmt.Println("Waiting for GitHub App installation...")
	fmt.Println("   (Press Ctrl+C to cancel)")
	fmt.Println()

	// Poll for installation
	timeout := time.Duration(githubConnectTimeout) * time.Second
	pollInterval := 5 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		installations, err := client.ListGitHubInstallations()
		if err != nil {
			continue // Retry on error
		}

		if len(installations.Installations) > initialCount {
			// New installation found
			newInstall := installations.Installations[len(installations.Installations)-1]
			fmt.Printf("GitHub App installed successfully!\n")
			fmt.Printf("   Account: %s (%s)\n", newInstall.AccountLogin, newInstall.AccountType)
			fmt.Println()

			// List available repos
			reposResp, err := client.ListAvailableRepos()
			if err != nil {
				fmt.Printf("Could not list repos: %v\n", err)
				return
			}

			// Count total repos across all installations
			totalRepos := 0
			for _, inst := range reposResp.Installations {
				totalRepos += len(inst.Repos)
			}

			if totalRepos == 0 {
				fmt.Println("No repositories found. Make sure the GitHub App has access to your repos.")
			} else {
				fmt.Printf("Found %d accessible repositories:\n", totalRepos)
				count := 0
				for _, inst := range reposResp.Installations {
					for _, repo := range inst.Repos {
						if count >= 10 {
							fmt.Printf("   ... and %d more\n", totalRepos-10)
							break
						}
						fmt.Printf("   %s\n", repo.FullName)
						count++
					}
					if count >= 10 {
						break
					}
				}
			}

			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("   1. Run 'datagen repo add <owner/repo>' to connect a repository")
			fmt.Println("   2. Run 'datagen agents list' to see discovered agents")
			return
		}

		fmt.Print(".")
	}

	fmt.Println()
	fmt.Println("Timed out waiting for GitHub App installation.")
	fmt.Println("   If you completed the installation, run 'datagen github status' to check.")
}

func runGitHubRepos(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📁 Fetching available repositories...")

	reposResp, err := client.ListAvailableRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Count total repos
	totalRepos := 0
	for _, inst := range reposResp.Installations {
		totalRepos += len(inst.Repos)
	}

	if totalRepos == 0 {
		fmt.Println("\nNo repositories found.")
		fmt.Println("Run 'datagen github connect' to install the GitHub App first.")
		return
	}

	fmt.Printf("\n📦 Available repositories (%d):\n\n", totalRepos)

	for _, inst := range reposResp.Installations {
		if len(inst.Repos) > 0 {
			fmt.Printf("📍 %s (%s)\n", inst.Installation.AccountLogin, inst.Installation.AccountType)
			for _, repo := range inst.Repos {
				visibility := "private"
				if !repo.Private {
					visibility = "public"
				}
				connectedIcon := ""
				if repo.IsConnected {
					connectedIcon = " ✓"
				}
				fmt.Printf("  • %s (%s)%s\n", repo.FullName, visibility, connectedIcon)
			}
			fmt.Println()
		}
	}

	fmt.Println("Use 'datagen repo add <owner/repo>' to connect a repository.")
}

func runGitHubConnected(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📁 Fetching connected repositories...")

	repos, err := client.ListConnectedRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(repos.Repos) == 0 {
		fmt.Println("\nNo connected repositories.")
		fmt.Println("Run 'datagen repo add <owner/repo>' to connect one.")
		return
	}

	fmt.Printf("\n🔗 Connected repositories (%d):\n\n", len(repos.Repos))

	for _, repo := range repos.Repos {
		statusIcon := "✅"
		switch repo.SyncStatus {
		case "SYNCING":
			statusIcon = "🔄"
		case "ERROR":
			statusIcon = "❌"
		case "PENDING":
			statusIcon = "⏳"
		}

		fmt.Printf("  %s %s\n", statusIcon, repo.FullName)
		fmt.Printf("    ID: %s | Status: %s | Agents: %d\n", repo.ID, repo.SyncStatus, repo.AgentCount)
	}

	fmt.Println()
	fmt.Println("Use 'datagen agents list' to see all discovered agents.")
}

func runGitHubConnectRepo(cmd *cobra.Command, args []string) {
	fullName := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔗 Connecting repository: %s\n", fullName)

	resp, err := client.ConnectRepo(fullName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Connected: %s\n", resp.Repo.FullName)
	fmt.Printf("   ID: %s\n", resp.Repo.ID)
	fmt.Printf("   Agents discovered: %d\n", resp.AgentsDiscovered)

	if resp.AgentsDiscovered > 0 {
		fmt.Println()
		fmt.Println("📝 Next steps:")
		fmt.Println("   1. Run 'datagen agents list' to see discovered agents")
		fmt.Println("   2. Run 'datagen agents deploy <agent-id>' to deploy an agent")
	} else {
		fmt.Println()
		fmt.Println("💡 No agents found in .claude/agents/ directory.")
		fmt.Println("   Create an agent file and run 'datagen github sync <repo-id>' to refresh.")
	}
}

func runGitHubSync(cmd *cobra.Command, args []string) {
	repoID := args[0]

	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔄 Syncing repository: %s\n", repoID)

	resp, err := client.SyncRepo(repoID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Sync complete!\n")
	fmt.Printf("   Agents found: %d\n", resp.AgentsFound)
	fmt.Printf("   New agents: %d\n", resp.NewAgents)
	fmt.Printf("   Updated agents: %d\n", resp.UpdatedAgents)
}

func runGitHubStatus(cmd *cobra.Command, args []string) {
	client, err := getAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔍 Checking GitHub connection status...")

	installations, err := client.ListGitHubInstallations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(installations.Installations) == 0 {
		fmt.Println("\n❌ No GitHub App installations found.")
		fmt.Println("   Run 'datagen github connect' to install the GitHub App.")
		return
	}

	fmt.Printf("\n✅ GitHub App installations (%d):\n\n", len(installations.Installations))

	for _, install := range installations.Installations {
		statusIcon := "✅"
		if !install.IsActive {
			statusIcon = "⚠️"
		}

		fmt.Printf("  %s %s (%s)\n", statusIcon, install.AccountLogin, install.AccountType)
		fmt.Printf("    Installation ID: %d\n", install.InstallationID)
		fmt.Printf("    Active: %v\n", install.IsActive)
	}

	// Also show connected repos count
	repos, err := client.ListConnectedRepos()
	if err == nil {
		fmt.Printf("\n📁 Connected repositories: %d\n", len(repos.Repos))
	}

	// Show agents count
	agents, err := client.ListAgents()
	if err == nil {
		deployedCount := 0
		for _, a := range agents.Agents {
			if a.IsDeployed {
				deployedCount++
			}
		}
		fmt.Printf("🤖 Discovered agents: %d (%d deployed)\n", len(agents.Agents), deployedCount)
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
