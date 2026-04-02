package cmd

import (
	"fmt"
	"os"

	"github.com/datagendev/datagen-cli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and check for updates",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("datagen version %s\n", version.Version)

		if version.Version == "dev" {
			fmt.Println("Development build -- version checking disabled.")
			return
		}

		fmt.Println("Checking for updates...")
		latest, err := version.FetchLatestVersion()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not check for updates: %v\n", err)
			return
		}

		if version.IsNewer(version.Version, latest) {
			fmt.Printf("\nA newer version is available: %s (current: %s)\n", latest, version.Version)
			fmt.Println("Update with: curl -fsSL https://raw.githubusercontent.com/datagendev/datagen-cli/main/install.sh | sh")
		} else {
			fmt.Printf("You are up to date. (%s)\n", latest)
		}
	},
}
