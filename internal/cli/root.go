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

// mustMarkRequired marks a flag as required and panics if the flag doesn't exist.
// This is intentional: missing required flags are programming errors caught at startup.
func mustMarkRequired(cmd *cobra.Command, flag string) {
	if err := cmd.MarkFlagRequired(flag); err != nil {
		panic(fmt.Sprintf("failed to mark flag %q as required on %q: %v", flag, cmd.Name(), err))
	}
}

func init() {
	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("Gosling version %s (commit: %s, built: %s)\n", Version, GitCommit, BuildDate))
}
