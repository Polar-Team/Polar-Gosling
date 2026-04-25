package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/polar-gosling/gosling/internal/lockbox"
	"github.com/spf13/cobra"
	ycsdk "github.com/yandex-cloud/go-sdk"
)

var (
	createProvider string
	createEggName  string
	createFolderID string
	createRegion   string
)

var lockboxCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cloud secret store for an Egg",
	Long: `Create a new cloud secret store with placeholder entries for an Egg configuration.

The secret store is created with three required entries (runner-token, webhook-secret,
repo-url) set to empty placeholder values. After creation, the generated Secret URIs
are printed to stdout for use in config.fly.

Examples:
  gosling lockbox create --provider yandex --egg-name my-app --folder-id abc123
  gosling lockbox create --provider aws --egg-name my-app --region us-east-1`,
	RunE: runLockboxCreate,
}

func init() {
	lockboxCmd.AddCommand(lockboxCreateCmd)

	lockboxCreateCmd.Flags().StringVar(&createProvider, "provider", "", "Cloud provider: yandex or aws")
	lockboxCreateCmd.Flags().StringVar(&createEggName, "egg-name", "", "Name of the Egg")
	lockboxCreateCmd.Flags().StringVar(&createFolderID, "folder-id", "", "Yandex Cloud folder ID (required for yandex)")
	lockboxCreateCmd.Flags().StringVar(&createRegion, "region", "", "AWS region (optional, uses SDK default if omitted)")

	mustMarkRequired(lockboxCreateCmd, "provider")
	mustMarkRequired(lockboxCreateCmd, "egg-name")
}

func runLockboxCreate(cmd *cobra.Command, args []string) error {
	params := lockbox.CreateParams{
		Provider: createProvider,
		EggName:  createEggName,
		FolderID: createFolderID,
		Region:   createRegion,
	}

	if err := lockbox.ValidateCreateInput(params); err != nil {
		return err
	}

	ctx := context.Background()

	store, err := newSecretStore(ctx, params.Provider, params.FolderID, params.Region)
	if err != nil {
		return err
	}

	result, err := store.Create(ctx, params)
	if err != nil {
		return err
	}

	// Print secret URIs to stdout, one per line
	for _, key := range lockbox.RequiredEntries() {
		fmt.Fprintln(os.Stdout, result.URIs[key])
	}

	return nil
}

// newSecretStore creates the appropriate SecretStore based on provider.
func newSecretStore(ctx context.Context, provider, folderID, region string) (lockbox.SecretStore, error) {
	switch provider {
	case "yandex":
		sdk, err := ycsdk.Build(ctx, ycsdk.Config{
			Credentials: ycsdk.InstanceServiceAccount(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Yandex Cloud SDK: %w", err)
		}
		return lockbox.NewYCLockboxStore(sdk, folderID), nil
	case "aws":
		store, err := lockbox.NewAWSSecretsManagerStore(ctx, region)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize AWS Secrets Manager: %w", err)
		}
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}
