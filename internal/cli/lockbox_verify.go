package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/polar-gosling/gosling/internal/lockbox"
	"github.com/spf13/cobra"
)

var (
	verifyProvider   string
	verifySecretID   string
	verifySecretName string
	verifyFolderID   string
	verifyRegion     string
)

var lockboxVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a secret store has all required entries",
	Long: `Verify that a cloud secret store exists and contains all required entries
(runner-token, webhook-secret, repo-url).

Prints a success message if all entries are present, or a report of present
and missing entries if any are missing.

Examples:
  gosling lockbox verify --provider yandex --folder-id abc123 --secret-id e6q...
  gosling lockbox verify --provider aws --secret-name polar-gosling/my-app`,
	RunE: runLockboxVerify,
}

func init() {
	lockboxCmd.AddCommand(lockboxVerifyCmd)

	lockboxVerifyCmd.Flags().StringVar(&verifyProvider, "provider", "", "Cloud provider: yandex or aws")
	lockboxVerifyCmd.Flags().StringVar(&verifySecretID, "secret-id", "", "Yandex Cloud Lockbox secret ID")
	lockboxVerifyCmd.Flags().StringVar(&verifySecretName, "secret-name", "", "AWS Secrets Manager secret name")
	lockboxVerifyCmd.Flags().StringVar(&verifyFolderID, "folder-id", "", "Yandex Cloud folder ID (required for yandex)")
	lockboxVerifyCmd.Flags().StringVar(&verifyRegion, "region", "", "AWS region (optional, uses SDK default if omitted)")

	mustMarkRequired(lockboxVerifyCmd, "provider")
}

func runLockboxVerify(cmd *cobra.Command, args []string) error {
	if err := lockbox.ValidateProviderFlags(verifyProvider, verifyFolderID); err != nil {
		return err
	}

	// Determine the secret reference based on provider
	var secretRef string
	switch verifyProvider {
	case "yandex":
		if verifySecretID == "" {
			return fmt.Errorf("secret-id is required for Yandex Cloud provider")
		}
		secretRef = verifySecretID
	case "aws":
		if verifySecretName == "" {
			return fmt.Errorf("secret-name is required for AWS provider")
		}
		secretRef = verifySecretName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := newSecretStore(ctx, verifyProvider, verifyFolderID, verifyRegion)
	if err != nil {
		return err
	}

	result, err := store.Verify(ctx, secretRef)
	if err != nil {
		return err
	}

	if len(result.Missing) == 0 {
		fmt.Fprintf(os.Stdout, "OK: All required entries found: %s\n", strings.Join(result.Present, ", "))
		return nil
	}

	fmt.Fprintln(os.Stderr, "Secret store is missing required entries:")
	if len(result.Present) > 0 {
		fmt.Fprintf(os.Stderr, "  Present: %s\n", strings.Join(result.Present, ", "))
	}
	fmt.Fprintf(os.Stderr, "  Missing: %s\n", strings.Join(result.Missing, ", "))

	// Return error to trigger non-zero exit code
	return fmt.Errorf("missing %d required entries", len(result.Missing))
}
