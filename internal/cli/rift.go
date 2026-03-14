package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/polar-gosling/gosling/internal/rift"
	"github.com/spf13/cobra"
)

var (
	riftListenAddr         string
	riftAuthTokenSecretURI string
	riftS3Endpoint         string
	riftS3Bucket           string
	riftS3Region           string
	riftS3KeyPrefix        string
	riftS3CredentialsURI   string
	riftDockerSocket       string
	riftImageCacheDir      string
	riftMotherGooseURL     string
	riftAPIKey             string
	riftAntiFlap           time.Duration
	riftIdleTimeout        time.Duration
	riftCacheSyncInterval  time.Duration
)

var riftServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Rift server",
	Long:  `Start the Rift Docker context proxy and image cache server.`,
	RunE:  runRiftServe,
}

func init() {
	riftRootCmd.AddCommand(riftServeCmd)

	f := riftServeCmd.Flags()
	f.StringVar(&riftListenAddr, "listen", ":2376", "TCP address to listen on")
	f.StringVar(&riftAuthTokenSecretURI, "auth-token-secret", "", "Secret URI for the bearer token (yc-lockbox://, aws-sm://, vault://)")
	f.StringVar(&riftS3Endpoint, "s3-endpoint", "", "S3-compatible endpoint URL (empty = AWS default)")
	f.StringVar(&riftS3Bucket, "s3-bucket", "", "S3 bucket for image cache storage")
	f.StringVar(&riftS3Region, "s3-region", "", "S3 bucket region")
	f.StringVar(&riftS3KeyPrefix, "s3-key-prefix", "rift/", "Key prefix inside the S3 bucket")
	f.StringVar(&riftS3CredentialsURI, "s3-credentials-secret", "", "Secret URI for S3 credentials")
	f.StringVar(&riftDockerSocket, "docker-socket", "/var/run/docker.sock", "Path to Docker daemon socket")
	f.StringVar(&riftImageCacheDir, "image-cache-dir", "/var/cache/rift/images", "Local directory for image tarballs")
	f.StringVar(&riftMotherGooseURL, "mothergoose-url", "", "MotherGoose API URL for state reporting")
	f.StringVar(&riftAPIKey, "api-key", "", "MotherGoose API key")
	f.DurationVar(&riftAntiFlap, "anti-flap", 2*time.Minute, "Minimum time in running state before shutdown is allowed")
	f.DurationVar(&riftIdleTimeout, "idle-timeout", 10*time.Minute, "Idle time before automatic shutdown")
	f.DurationVar(&riftCacheSyncInterval, "cache-sync-interval", 5*time.Minute, "How often to sync image cache to S3")

	mustMarkRequired(riftServeCmd, "auth-token-secret")
	mustMarkRequired(riftServeCmd, "s3-bucket")
}

func runRiftServe(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &rift.Config{
		ListenAddr:         riftListenAddr,
		AuthTokenSecretURI: riftAuthTokenSecretURI,
		S3: rift.S3Config{
			Endpoint:             riftS3Endpoint,
			Bucket:               riftS3Bucket,
			Region:               riftS3Region,
			KeyPrefix:            riftS3KeyPrefix,
			CredentialsSecretURI: riftS3CredentialsURI,
		},
		DockerSocketPath:      riftDockerSocket,
		ImageCacheDir:         riftImageCacheDir,
		MotherGooseURL:        riftMotherGooseURL,
		APIKey:                riftAPIKey,
		AntiFlap:              riftAntiFlap,
		IdleTimeout:           riftIdleTimeout,
		CacheSyncInterval:     riftCacheSyncInterval,
		EggSSHKeyFingerprints: make(map[string]string),
		EggSSHKeySecretURIs:   make(map[string]string),
	}

	resolvedToken, err := resolveRiftSecret(cfg.AuthTokenSecretURI)
	if err != nil {
		return fmt.Errorf("rift: failed to resolve auth token: %w", err)
	}

	s3Client := rift.NewS3UploaderFromConfig(cfg.S3)
	cache := rift.NewImageCache(cfg.ImageCacheDir, cfg.S3.KeyPrefix, s3Client)
	// hooks are nil here — lifecycle (start/stop VM) is driven by MotherGoose via OpenTofu
	orchestrator := rift.NewOrchestrator(cfg, cache, nil)
	server := rift.NewServer(cfg, resolvedToken, orchestrator)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigCh
		log.Println("rift: received shutdown signal")
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutCancel()
		if err := server.Shutdown(shutCtx); err != nil {
			log.Printf("rift: shutdown error: %v", err)
		}
	}()

	go func() {
		if runErr := orchestrator.Run(ctx); runErr != nil {
			log.Printf("rift: orchestrator stopped: %v", runErr)
		}
	}()

	log.Printf("rift: starting server on %s", cfg.ListenAddr)
	if serveErr := server.ListenAndServe(); serveErr != nil {
		if serveErr.Error() != "http: Server closed" {
			return fmt.Errorf("rift: server error: %w", serveErr)
		}
	}
	return nil
}

// resolveRiftSecret retrieves a secret value from the URI scheme.
// Falls back to RIFT_AUTH_TOKEN env var for local development.
func resolveRiftSecret(uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("secret URI must not be empty")
	}
	// Task 29: wire real secret backends (YC Lockbox, AWS SM, Vault)
	if token := os.Getenv("RIFT_AUTH_TOKEN"); token != "" {
		return token, nil
	}
	return "", fmt.Errorf("rift: secret resolution not yet implemented for %q; set RIFT_AUTH_TOKEN for local dev", uri)
}
