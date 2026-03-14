package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// riftRootCmd is the root command for the standalone `rift` binary.
var riftRootCmd = &cobra.Command{
	Use:   "rift",
	Short: "Rift - Remote Docker context and artifact cache server",
	Long: `Rift is a shared cache and remote Docker context server for CI runners.

It proxies Docker API requests from runners to the local Docker daemon,
caches Docker image tarballs to S3, and integrates with the GitLab CI
cache protocol backed by S3.

Rift is an optional component: runners operate correctly without it,
but benefit from faster image pulls and shared layer caches.`,
}

// ExecuteRift runs the rift root command (used by cmd/rift/main.go).
func ExecuteRift() error {
	if err := riftRootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
