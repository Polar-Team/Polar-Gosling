package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "gosling",
	Short: "Gosling CLI - GitOps Runner Orchestration",
	Long: `Gosling is a CLI tool for managing GitOps-based CI/CD runner orchestration.
It provides commands to bootstrap Nest repositories, manage Egg configurations,
and deploy runners across multiple cloud providers.`,
	Version: Version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("Gosling version %s (commit: %s, built: %s)\n", Version, GitCommit, BuildDate))
}
