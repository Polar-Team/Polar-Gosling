package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/polar-gosling/gosling/internal/mothergoose"
	"github.com/spf13/cobra"
)

var (
	statusEgg    string
	statusAll    bool
	statusAPIURL string
	statusAPIKey string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status",
	Long:  "Show the current deployment status for eggs.",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusEgg, "egg", "", "Egg name")
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Show all eggs")
	statusCmd.Flags().StringVar(&statusAPIURL, "api-url", "", "MotherGoose API URL")
	statusCmd.Flags().StringVar(&statusAPIKey, "api-key", "", "MotherGoose API key")
	statusCmd.MarkFlagRequired("api-url")
	statusCmd.MarkFlagRequired("api-key")
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	if statusEgg == "" && !statusAll {
		return fmt.Errorf("either --egg or --all flag must be specified")
	}

	client := mothergoose.NewClient(statusAPIURL, statusAPIKey)

	if statusAll {
		return showAllStatus(ctx, client)
	}
	return showEggStatus(ctx, client, statusEgg)
}

func showEggStatus(ctx context.Context, client mothergoose.MotherGooseClient, eggName string) error {
	fmt.Printf("=== Deployment Status for Egg: %s ===\n\n", eggName)
	status, err := client.GetEggStatus(ctx, eggName)
	if err != nil {
		return fmt.Errorf("failed to get egg status: %w", err)
	}

	if status.LatestPlan == nil {
		fmt.Println("No deployment found for this egg")
		return nil
	}

	latestPlan := status.LatestPlan
	fmt.Println("Current Deployment:")
	fmt.Printf("  Plan ID:      %s\n", latestPlan.ID)
	fmt.Printf("  Status:       %s\n", latestPlan.Status)
	fmt.Printf("  Config Hash:  %s\n", latestPlan.ConfigHash)
	fmt.Printf("  Created At:   %s\n", latestPlan.CreatedAt.Format(time.RFC3339))
	if latestPlan.AppliedAt != nil {
		fmt.Printf("  Applied At:   %s\n", latestPlan.AppliedAt.Format(time.RFC3339))
	}
	fmt.Printf("  Plan Type:    %s\n", latestPlan.PlanType)
	if len(latestPlan.Metadata) > 0 {
		fmt.Println("\n  Metadata:")
		for key, value := range latestPlan.Metadata {
			fmt.Printf("    %s: %v\n", key, value)
		}
	}

	if len(status.ActiveRunners) > 0 {
		fmt.Printf("\n\nActive Runners (%d):\n", len(status.ActiveRunners))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RUNNER ID\tTYPE\tSTATE\tCLOUD\tREGION\tLAST HEARTBEAT")
		fmt.Fprintln(w, "---------\t----\t-----\t-----\t------\t--------------")
		for _, runner := range status.ActiveRunners {
			runnerID := runner.ID
			if len(runnerID) > 12 {
				runnerID = runnerID[:12] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				runnerID,
				runner.Type,
				runner.State,
				runner.CloudProvider,
				runner.Region,
				runner.LastHeartbeat.Format("2006-01-02 15:04"))
		}
		w.Flush()
	}

	if len(status.DeploymentHistory) > 1 {
		fmt.Printf("\n\nDeployment History (%d plans):\n", len(status.DeploymentHistory))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PLAN ID\tSTATUS\tCREATED\tAPPLIED")
		fmt.Fprintln(w, "-------\t------\t-------\t-------")
		for _, plan := range status.DeploymentHistory {
			planID := plan.ID
			if len(planID) > 8 {
				planID = planID[:8] + "..."
			}
			appliedStr := "-"
			if plan.AppliedAt != nil {
				appliedStr = plan.AppliedAt.Format("2006-01-02 15:04")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", planID, plan.Status, plan.CreatedAt.Format("2006-01-02 15:04"), appliedStr)
		}
		w.Flush()
	}
	return nil
}

func showAllStatus(ctx context.Context, client mothergoose.MotherGooseClient) error {
	eggs, err := client.ListEggs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list eggs: %w", err)
	}

	if len(eggs) == 0 {
		fmt.Println("No eggs found")
		return nil
	}

	fmt.Println("=== Deployment Status for All Eggs ===")
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "EGG NAME\tSTATUS\tPLAN ID\tAPPLIED AT\tCONFIG HASH")
	fmt.Fprintln(w, "--------\t------\t-------\t----------\t-----------")

	for _, egg := range eggs {
		eggName := egg.Name
		status, err := client.GetEggStatus(ctx, eggName)
		if err != nil || status.LatestPlan == nil {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", eggName, "not deployed", "-", "-", "-")
			continue
		}

		latestPlan := status.LatestPlan
		planID := latestPlan.ID
		if len(planID) > 8 {
			planID = planID[:8] + "..."
		}
		configHash := latestPlan.ConfigHash
		if len(configHash) > 12 {
			configHash = configHash[:12] + "..."
		}
		appliedStr := "-"
		if latestPlan.AppliedAt != nil {
			appliedStr = latestPlan.AppliedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", eggName, latestPlan.Status, planID, appliedStr, configHash)
	}
	w.Flush()
	return nil
}
