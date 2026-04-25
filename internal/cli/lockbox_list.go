package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/polar-gosling/gosling/internal/lockbox"
	"github.com/spf13/cobra"
)

var (
	listProvider string
	listFolderID string
	listRegion   string
)

var lockboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Polar Gosling secret stores",
	Long: `List all cloud secret stores tagged for Polar Gosling use.

Displays the name, ID/ARN, associated egg name, and creation date for each secret store.

Examples:
  gosling lockbox list --provider yandex --folder-id abc123
  gosling lockbox list --provider aws --region us-east-1`,
	RunE: runLockboxList,
}

func init() {
	lockboxCmd.AddCommand(lockboxListCmd)

	lockboxListCmd.Flags().StringVar(&listProvider, "provider", "", "Cloud provider: yandex or aws")
	lockboxListCmd.Flags().StringVar(&listFolderID, "folder-id", "", "Yandex Cloud folder ID (required for yandex)")
	lockboxListCmd.Flags().StringVar(&listRegion, "region", "", "AWS region (optional, uses SDK default if omitted)")

	mustMarkRequired(lockboxListCmd, "provider")
}

func runLockboxList(cmd *cobra.Command, args []string) error {
	if err := lockbox.ValidateProviderFlags(listProvider, listFolderID); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := newSecretStore(ctx, listProvider, listFolderID, listRegion)
	if err != nil {
		return err
	}

	secrets, err := store.List(ctx)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		fmt.Fprintln(os.Stderr, "no Polar Gosling secret stores found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tEGG\tCREATED")
	for _, s := range secrets {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.ID, s.EggName, s.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()

	return nil
}
