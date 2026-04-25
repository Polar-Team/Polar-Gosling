package cli

import (
	"github.com/spf13/cobra"
)

// lockboxCmd is the parent command: gosling lockbox
var lockboxCmd = &cobra.Command{
	Use:   "lockbox",
	Short: "Manage cloud secret stores for Egg configurations",
	Long: `Manage cloud secret stores for Egg configurations.

Provides subcommands to create, list, and verify secret stores
in Yandex Cloud Lockbox or AWS Secrets Manager.`,
}

func init() {
	rootCmd.AddCommand(lockboxCmd)
}
