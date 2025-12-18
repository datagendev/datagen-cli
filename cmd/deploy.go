package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var (
	deployOutputDir   string
	deployVars        []string
	deployProjectName string
)

type variableValue struct {
	Value  string
	Source string // flag, .env, env, prompt
}

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
	deployCmd.Flags().StringArrayVarP(&deployVars, "var", "v", nil, "Set Railway environment variables (repeatable). Formats: KEY=VALUE or KEY (use current env value)")
	deployCmd.Flags().StringVar(&deployProjectName, "project-name", "", "Railway project name (defaults to current folder name)")
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

	// Collect variables (don't set yet - service doesn't exist!)
	fmt.Println("🔐 Collecting environment variables...")
	collectedVars, err := collectRailwayVariables()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to collect variables: %v\n", err)
		os.Exit(1)
	}

	// Initial deployment - this CREATES the Railway service
	fmt.Println("🚀 Initial deployment (creating Railway service)...")
	if len(collectedVars) > 0 {
		fmt.Println("⏳ Service will be created first, then variables will be configured...")
	} else {
		fmt.Println("⏳ Deploying with default configuration...")
	}
	fmt.Println()

	if err := deployToRailway(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Initial deployment failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'railway logs --build' to see build errors\n")
		os.Exit(1)
	}
	fmt.Println("✅ Railway service created successfully!")
	fmt.Println()

	// Link to the service (Railway creates it but doesn't auto-link for CLI commands)
	if len(collectedVars) > 0 {
		if err := linkRailwayService(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Could not link to service: %v\n", err)
			fmt.Fprintf(os.Stderr, "You may need to run 'railway service' manually before setting variables\n")
		}
	}

	// Now set variables on the created service
	if len(collectedVars) > 0 {
		fmt.Println("🔧 Configuring environment variables...")
		if err := setRailwayVariables(collectedVars); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to set variables: %v\n", err)
			fmt.Fprintf(os.Stderr, "Service is running but variables need to be set manually\n")
			os.Exit(1)
		}

		// Redeploy to apply variables
		fmt.Println("🔄 Redeploying to apply variables...")
		if err := deployToRailway(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Redeploy failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Variables are set but may need manual redeploy: railway up\n")
			// Don't exit - show status below
		} else {
			fmt.Println("✅ Variables applied and active!")
		}
		fmt.Println()
	}

	// Show deployment status
	fmt.Println()
	if err := showDeploymentStatus(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not retrieve deployment status: %v\n", err)
	}

	// Get the deployment URL
	fmt.Println()
	if err := showDeploymentInfo(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not retrieve deployment URL: %v\n", err)
		fmt.Println("\nYou can get your deployment URL by running: railway domain")
	}

	fmt.Println("\n🎉 Deployment Complete!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. View deployment status: railway status")
	fmt.Println("  2. View environment variables: railway variables --kv")
	fmt.Println("  3. View deployment logs: railway logs --deployment")
	fmt.Println("  4. View build logs: railway logs --build")
	fmt.Println("  5. Open in browser: railway open")
	fmt.Println("  6. Update and redeploy: railway up")
	if len(collectedVars) > 0 {
		fmt.Println("\n💡 Your service is live with environment variables configured!")
	} else {
		fmt.Println("\n💡 Your service is live!")
	}
}

// collectRailwayVariables collects environment variables from flags, .env, environment, and prompts
// WITHOUT setting them on Railway. Returns the collected variables for later use.
func collectRailwayVariables() (map[string]variableValue, error) {
	requiredKeys := requiredKeysFromEnvExample(".env.example")
	if len(requiredKeys) == 0 {
		// Fallback to standard names used by this CLI.
		requiredKeys = []string{"ANTHROPIC_API_KEY", "DATAGEN_API_KEY"}
	}

	localEnv := loadDotEnvFile(".env")
	vars := map[string]variableValue{}

	// Apply -v / --var values first (explicitly provided values win).
	parsed, err := parseVarFlags(deployVars, localEnv)
	if err != nil {
		return nil, err
	}
	for k, v := range parsed {
		vars[k] = v
	}

	// Fill required keys from environment, prompt if still missing.
	for _, key := range requiredKeys {
		if _, ok := vars[key]; ok {
			continue
		}
		if v := strings.TrimSpace(localEnv[key]); v != "" {
			vars[key] = variableValue{Value: v, Source: ".env"}
			continue
		}
		if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
			vars[key] = variableValue{Value: v, Source: "env"}
			continue
		}

		var entered string
		if err := survey.AskOne(&survey.Password{
			Message: fmt.Sprintf("%s not found. Enter value:", key),
		}, &entered, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
		vars[key] = variableValue{Value: strings.TrimSpace(entered), Source: "prompt"}
	}

	if len(vars) == 0 {
		return nil, nil
	}

	// Show collected variables (preview)
	fmt.Println("🔐 Variables collected for deployment:")
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := vars[k]
		fmt.Printf("  ✓ %s=%s (from %s)\n", k, maskSecret(v.Value), v.Source)
	}
	fmt.Println("  💡 These will be set after the initial deployment")
	fmt.Println()

	return vars, nil
}

// setRailwayVariables sets the collected variables on the Railway service
func setRailwayVariables(vars map[string]variableValue) error {
	if len(vars) == 0 {
		return nil
	}

	fmt.Println("🔐 Setting environment variables on Railway...")

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := vars[k]
		fmt.Printf("  → %s=%s\n", k, maskSecret(v.Value))
	}

	args := []string{"variables", "--skip-deploys"}
	for _, k := range keys {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, vars[k].Value))
	}

	setCmd := exec.Command("railway", args...)
	setCmd.Stdout = os.Stdout
	setCmd.Stderr = os.Stderr
	if err := setCmd.Run(); err != nil {
		return fmt.Errorf("railway variables set failed: %w", err)
	}

	fmt.Printf("  ✅ Successfully configured %d variable(s): %s\n", len(keys), strings.Join(keys, ", "))
	fmt.Println()
	return nil
}

func railwayExistingVariableKeys() (map[string]bool, error) {
	cmd := exec.Command("railway", "variables", "--kv")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	keys := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if k != "" {
			keys[k] = true
		}
	}
	return keys, nil
}

func parseVarFlags(flags []string, localEnv map[string]string) (map[string]variableValue, error) {
	out := map[string]variableValue{}
	for _, raw := range flags {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if strings.Contains(raw, "=") {
			parts := strings.SplitN(raw, "=", 2)
			k := strings.TrimSpace(parts[0])
			if k == "" {
				return nil, fmt.Errorf("invalid --var %q (missing KEY)", raw)
			}
			out[k] = variableValue{Value: parts[1], Source: "flag"}
			continue
		}

		// KEY form: pull from local .env first, then current process environment.
		if v := strings.TrimSpace(localEnv[raw]); v != "" {
			out[raw] = variableValue{Value: v, Source: ".env"}
			continue
		}
		if v, ok := os.LookupEnv(raw); ok && strings.TrimSpace(v) != "" {
			out[raw] = variableValue{Value: v, Source: "env"}
			continue
		}

		// If not in env, leave unset; required vars will be prompted later.
	}
	return out, nil
}

func maskSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	n := len(v)
	if n <= 8 {
		return fmt.Sprintf("****(len=%d)", n)
	}
	return fmt.Sprintf("%s…%s(len=%d)", v[:3], v[n-3:], n)
}

func loadDotEnvFile(path string) map[string]string {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return map[string]string{}
	}

	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			continue
		}
		out[key] = strings.Trim(val, `"'`)
	}
	return out
}

func requiredKeysFromEnvExample(path string) []string {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	inRequired := false
	var keys []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			if inRequired {
				break
			}
			continue
		}
		if strings.HasPrefix(trim, "#") {
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trim, "#")), "Required") {
				inRequired = true
				continue
			}
			if inRequired && strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trim, "#")), "Optional") {
				break
			}
			continue
		}
		if !inRequired {
			continue
		}
		if k, _, ok := strings.Cut(trim, "="); ok {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
	}
	return keys
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

// linkRailwayService links to the Railway service after it's created
func linkRailwayService() error {
	fmt.Println("🔗 Linking to Railway service...")

	linkCmd := exec.Command("railway", "service")
	linkCmd.Stdin = os.Stdin
	linkCmd.Stdout = os.Stdout
	linkCmd.Stderr = os.Stderr

	if err := linkCmd.Run(); err != nil {
		return fmt.Errorf("railway service link failed: %w", err)
	}

	fmt.Println("✓ Service linked")
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

	projectName := strings.TrimSpace(deployProjectName)
	if projectName == "" {
		defaultName := defaultProjectName()
		if defaultName != "" {
			if err := survey.AskOne(&survey.Input{
				Message: "Project Name:",
				Default: defaultName,
				Help:    "Press Enter to use the default (current folder name).",
			}, &projectName); err != nil {
				return err
			}
			projectName = strings.TrimSpace(projectName)
		}
	}

	args := []string{"init"}
	if projectName != "" && railwayInitSupportsNameFlag() {
		args = append(args, "--name", projectName)
	} else if projectName != "" {
		fmt.Printf("ℹ️  When prompted for Project Name, use: %s\n\n", projectName)
	}

	cmd := exec.Command("railway", args...)
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

func defaultProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	base := filepath.Base(cwd)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func railwayInitSupportsNameFlag() bool {
	helpCmd := exec.Command("railway", "init", "--help")
	out, err := helpCmd.Output()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	return strings.Contains(s, "--name") || strings.Contains(s, "-n, --name") || strings.Contains(s, "--project-name")
}

// deployToRailway runs the railway up command
func deployToRailway() error {
	// Deploy the code (service should already be linked at this point)
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

// showRailwayVariables displays all currently set Railway variables
func showRailwayVariables() error {
	fmt.Println("\n📋 Current Railway environment variables:")

	cmd := exec.Command("railway", "variables", "--kv")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to fetch variables: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		fmt.Println("  (no variables set)")
		return nil
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			fmt.Printf("  ✓ %s=%s\n", k, maskSecret(v))
		}
	}
	fmt.Println()
	return nil
}

// showDeploymentStatus displays current Railway project status
func showDeploymentStatus() error {
	fmt.Println("📊 Deployment Status:")

	cmd := exec.Command("railway", "status")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Println()
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
