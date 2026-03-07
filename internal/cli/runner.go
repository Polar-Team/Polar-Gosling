package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/polar-gosling/gosling/internal/runner"
	"github.com/spf13/cobra"
)

var (
	runnerEggName           string
	runnerID                string
	runnerTokenSecret       string
	runnerGitLabServer      string
	runnerTags              string
	runnerAgentVersion      string
	runnerMotherGooseURL    string
	runnerAPIKey            string
	runnerMetricsInterval   time.Duration
	runnerHeartbeatInterval time.Duration
)

var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Start the GitLab Runner Agent manager",
	Long: `Start the GitLab Runner Agent manager on a deployed runner.

This command is the entrypoint used inside deployed runner containers/VMs.
It registers the GitLab Runner Agent with GitLab, manages the agent process
lifecycle, synchronizes the agent version, and reports health metrics.`,
	RunE: runRunner,
}

func init() {
	rootCmd.AddCommand(runnerCmd)
	runnerCmd.Flags().StringVar(&runnerEggName, "egg-name", "", "Name of the Egg this runner belongs to")
	runnerCmd.Flags().StringVar(&runnerID, "runner-id", "", "Unique runner ID (defaults to runner-<egg-name>)")
	runnerCmd.Flags().StringVar(&runnerTokenSecret, "token-secret", "", "Secret URI for the GitLab runner token (e.g., yc-lockbox://gitlab/gitlab.com/my-app/runner-token)")
	runnerCmd.Flags().StringVar(&runnerGitLabServer, "gitlab-server", "gitlab.com", "GitLab server FQDN")
	runnerCmd.Flags().StringVar(&runnerTags, "tags", "", "Comma-separated runner tags")
	runnerCmd.Flags().StringVar(&runnerAgentVersion, "agent-version", "", "Required GitLab Runner Agent version (uses latest if not specified)")
	runnerCmd.Flags().StringVar(&runnerMotherGooseURL, "mothergoose-url", "", "MotherGoose API URL for metrics reporting")
	runnerCmd.Flags().StringVar(&runnerAPIKey, "api-key", "", "MotherGoose API key")
	runnerCmd.Flags().DurationVar(&runnerMetricsInterval, "metrics-interval", 30*time.Second, "How often to report full metrics to MotherGoose")
	runnerCmd.Flags().DurationVar(&runnerHeartbeatInterval, "heartbeat-interval", 30*time.Second, "How often to send heartbeat pings to MotherGoose")
	mustMarkRequired(runnerCmd, "egg-name")
	mustMarkRequired(runnerCmd, "token-secret")
}

func runRunner(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &runner.Config{
		EggName:           runnerEggName,
		RunnerID:          runnerID,
		TokenSecretURI:    runnerTokenSecret,
		GitLabServer:      runnerGitLabServer,
		Tags:              runner.ParseTags(runnerTags),
		AgentVersion:      runnerAgentVersion,
		MotherGooseURL:    runnerMotherGooseURL,
		APIKey:            runnerAPIKey,
		MetricsInterval:   runnerMetricsInterval,
		HeartbeatInterval: runnerHeartbeatInterval,
	}

	mgr, err := runner.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runner manager: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGTERM, syscall.SIGINT:
				fmt.Printf("Received signal %s, initiating graceful shutdown\n", sig)
				cancel()
			case syscall.SIGHUP:
				fmt.Println("Received SIGHUP, reloading configuration")
				mgr.ReloadConfig()
			}
		}
	}()

	return mgr.Run(ctx)
}
