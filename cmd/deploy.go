package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	deployOutputDir string
)

var deployCmd = &cobra.Command{
	Use:   "deploy [platform]",
	Short: "Deploy to cloud platform",
	Long:  `Deploy your project to a cloud platform (currently supports: railway)`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runDeploy,
}

func init() {
	deployCmd.Flags().StringVarP(&deployOutputDir, "output", "o", ".", "Directory containing the project to deploy")
	deployCmd.MarkFlagDirname("output")
}

func runDeploy(cmd *cobra.Command, args []string) {
	platform := "railway"
	if len(args) > 0 {
		platform = args[0]
	}

	if platform != "railway" {
		fmt.Fprintf(os.Stderr, "Error: Only 'railway' platform is currently supported\n")
		os.Exit(1)
	}

	// Change to output directory if specified
	if deployOutputDir != "." {
		if err := os.Chdir(deployOutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Cannot access directory %s: %v\n", deployOutputDir, err)
			os.Exit(1)
		}
		fmt.Printf("📁 Working in directory: %s\n", deployOutputDir)
	}

	fmt.Println("🚀 Deploying to Railway...")

	// Check if railway CLI is installed
	if _, err := exec.LookPath("railway"); err != nil {
		fmt.Println("⚠️  Railway CLI is not installed")
		fmt.Println()

		// Offer to install Railway CLI
		if !promptInstallRailway() {
			fmt.Println("\nPlease install Railway CLI manually from: https://docs.railway.com/guides/cli")
			os.Exit(1)
		}

		// Verify installation succeeded
		if _, err := exec.LookPath("railway"); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Installation failed or Railway CLI not found in PATH\n")
			fmt.Println("Please try installing manually: https://docs.railway.com/guides/cli")
			os.Exit(1)
		}

		fmt.Println("✅ Railway CLI installed successfully!")
		fmt.Println()
	}

	// Check if logged in
	checkCmd := exec.Command("railway", "whoami")
	if err := checkCmd.Run(); err != nil {
		fmt.Println("⚠️  Not logged in to Railway")
		fmt.Println()

		// Offer to run railway login
		if !promptRailwayLogin() {
			fmt.Println("\nPlease run 'railway login' to authenticate")
			os.Exit(1)
		}

		// Verify login succeeded
		checkCmd := exec.Command("railway", "whoami")
		if err := checkCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Login failed or was cancelled\n")
			fmt.Println("Please try 'railway login' manually")
			os.Exit(1)
		}

		fmt.Println("✅ Successfully logged in to Railway!")
		fmt.Println()
	}

	// Check for required files
	if err := checkRequiredFiles(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// Initialize Railway project if needed
	if err := ensureRailwayProject(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize Railway project: %v\n", err)
		os.Exit(1)
	}

	// Deploy the project
	fmt.Println("🚢 Deploying to Railway...")
	if err := deployToRailway(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Deployment failed: %v\n", err)
		os.Exit(1)
	}

	// Get the deployment URL
	fmt.Println()
	if err := showDeploymentInfo(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not retrieve deployment URL: %v\n", err)
		fmt.Println("\nYou can get your deployment URL by running: railway domain")
	}

	fmt.Println("\n✅ Deployment successful!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Set environment variables: railway variables")
	fmt.Println("  2. View logs: railway logs")
	fmt.Println("  3. Open in browser: railway open")
}

// checkRequiredFiles verifies that all required deployment files exist
func checkRequiredFiles() error {
	requiredFiles := []string{"requirements.txt", "Procfile"}
	optionalFiles := []string{"railway.json", "Dockerfile"}

	fmt.Println("📋 Checking required files...")

	// Check required files
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required file missing: %s", file)
		}
		fmt.Printf("  ✓ %s\n", file)
	}

	// Check optional files
	for _, file := range optionalFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("  ✓ %s\n", file)
		}
	}

	fmt.Println()
	return nil
}

// ensureRailwayProject checks if Railway is initialized and initializes if needed
func ensureRailwayProject() error {
	// Check if .railway directory exists (indicates initialized project)
	if _, err := os.Stat(".railway"); err == nil {
		fmt.Println("✓ Railway project already initialized")
		return nil
	}

	fmt.Println("🔧 Initializing Railway project...")
	fmt.Println()

	cmd := exec.Command("railway", "init")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("railway init failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Railway project initialized")
	return nil
}

// deployToRailway runs the railway up command
func deployToRailway() error {
	cmd := exec.Command("railway", "up", "--detach")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("railway up failed: %w", err)
	}

	return nil
}

// showDeploymentInfo displays the deployment URL and other info
func showDeploymentInfo() error {
	fmt.Println("🌐 Getting deployment information...")

	// Get the domain
	cmd := exec.Command("railway", "domain")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	domain := strings.TrimSpace(string(output))
	if domain != "" {
		fmt.Printf("\n🔗 Deployment URL: https://%s\n", domain)
	}

	return nil
}

// promptInstallRailway asks user if they want to install Railway CLI and does it
func promptInstallRailway() bool {
	fmt.Println("Would you like to install Railway CLI now? (y/n)")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return false
	}

	fmt.Println()
	return installRailwayCLI()
}

// promptRailwayLogin asks user if they want to log in to Railway and does it
func promptRailwayLogin() bool {
	fmt.Println("Would you like to log in to Railway now? (y/n)")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return false
	}

	fmt.Println()
	fmt.Println("🔐 Opening Railway login in your browser...")

	cmd := exec.Command("railway", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Login command failed: %v\n", err)
		return false
	}

	return true
}

// installRailwayCLI attempts to install Railway CLI based on the platform
func installRailwayCLI() bool {
	switch runtime.GOOS {
	case "darwin":
		return installRailwayMacOS()
	case "linux":
		return installRailwayLinux()
	case "windows":
		return installRailwayWindows()
	default:
		fmt.Fprintf(os.Stderr, "Unsupported platform: %s\n", runtime.GOOS)
		return false
	}
}

// installRailwayMacOS installs Railway CLI on macOS
func installRailwayMacOS() bool {
	fmt.Println("Installing Railway CLI on macOS...")

	// Check if Homebrew is available
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("📦 Using Homebrew to install Railway CLI...")
		cmd := exec.Command("brew", "install", "railway")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Homebrew installation failed: %v\n", err)
			return tryUniversalInstall()
		}
		return true
	}

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err == nil {
		fmt.Println("📦 Using npm to install Railway CLI...")
		cmd := exec.Command("npm", "install", "-g", "@railway/cli")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "npm installation failed: %v\n", err)
			return tryUniversalInstall()
		}
		return true
	}

	// Try universal script
	return tryUniversalInstall()
}

// installRailwayLinux installs Railway CLI on Linux
func installRailwayLinux() bool {
	fmt.Println("Installing Railway CLI on Linux...")

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err == nil {
		fmt.Println("📦 Using npm to install Railway CLI...")
		cmd := exec.Command("npm", "install", "-g", "@railway/cli")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "npm installation failed: %v\n", err)
			return tryUniversalInstall()
		}
		return true
	}

	// Try universal script
	return tryUniversalInstall()
}

// installRailwayWindows installs Railway CLI on Windows
func installRailwayWindows() bool {
	fmt.Println("Installing Railway CLI on Windows...")

	// Check if scoop is available
	if _, err := exec.LookPath("scoop"); err == nil {
		fmt.Println("📦 Using Scoop to install Railway CLI...")
		cmd := exec.Command("scoop", "install", "railway")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Scoop installation failed: %v\n", err)
			fmt.Println("\nTrying npm installation...")
		} else {
			return true
		}
	}

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err == nil {
		fmt.Println("📦 Using npm to install Railway CLI...")
		cmd := exec.Command("npm", "install", "-g", "@railway/cli")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "npm installation failed: %v\n", err)
			return false
		}
		return true
	}

	fmt.Println("\n⚠️  No supported package manager found (scoop or npm)")
	fmt.Println("Please install Railway CLI manually:")
	fmt.Println("  - Using npm: npm install -g @railway/cli")
	fmt.Println("  - Using scoop: scoop install railway")
	return false
}

// tryUniversalInstall tries the universal installation script
func tryUniversalInstall() bool {
	fmt.Println("📦 Trying universal installation script...")
	fmt.Println("Running: bash <(curl -fsSL cli.new)")

	// This requires bash and curl to be available
	if _, err := exec.LookPath("bash"); err != nil {
		fmt.Println("⚠️  bash not found, skipping universal install")
		return false
	}
	if _, err := exec.LookPath("curl"); err != nil {
		fmt.Println("⚠️  curl not found, skipping universal install")
		return false
	}

	cmd := exec.Command("bash", "-c", "curl -fsSL cli.new | bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Universal installation failed: %v\n", err)
		return false
	}

	return true
}
