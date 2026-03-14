// Package rift implements the Rift remote Docker context and artifact cache server.
//
// Rift is an optional component that provides:
//   - Remote Docker/Podman/nerdctl context for runners
//   - Docker image caching (tarballs) to/from S3
//   - GitLab CI cache integration backed by S3
//   - SSH key management per Egg project
//   - Lifecycle orchestration: scheduled up/down, on-demand wake, anti-flap
package rift

import (
	"time"
)

// State represents the current lifecycle state of the Rift server.
type State string

const (
	// StateOff means the Rift VM is stopped/deallocated.
	StateOff State = "off"
	// StateStarting means the Rift VM is being provisioned or booting.
	StateStarting State = "starting"
	// StateRunning means the Rift VM is up and accepting connections.
	StateRunning State = "running"
	// StateStopping means the Rift VM is flushing caches and shutting down.
	StateStopping State = "stopping"
	// StateDestroying means the Rift VM is being permanently destroyed.
	StateDestroying State = "destroying"
)

// ScheduleWindow defines a time window during which Rift should be running.
type ScheduleWindow struct {
	// Start is the cron expression for when to bring Rift up (e.g. "0 8 * * 1-5").
	Start string
	// Stop is the cron expression for when to bring Rift down (e.g. "0 20 * * 1-5").
	Stop string
}

// S3Config holds S3 (or compatible) bucket configuration for cache storage.
type S3Config struct {
	// Endpoint is the S3-compatible endpoint URL (empty = AWS default).
	Endpoint string
	// Bucket is the S3 bucket name.
	Bucket string
	// Region is the AWS/YC region.
	Region string
	// KeyPrefix is the path prefix inside the bucket (e.g. "rift/").
	KeyPrefix string
	// CredentialsSecretURI is a secret URI for S3 credentials (yc-lockbox://, aws-sm://).
	CredentialsSecretURI string
}

// Config is the full configuration for a Rift server instance.
type Config struct {
	// ServerID is a unique identifier for this Rift instance.
	ServerID string

	// ListenAddr is the TCP address the Docker context proxy listens on (e.g. ":2376").
	ListenAddr string

	// AuthToken is the shared secret runners use to authenticate with Rift.
	// Stored as a secret URI (yc-lockbox://, aws-sm://, vault://).
	AuthTokenSecretURI string

	// S3 holds the cache bucket configuration.
	S3 S3Config

	// DockerSocketPath is the path to the local Docker daemon socket.
	DockerSocketPath string

	// ImageCacheDir is the local directory for Docker image tarballs before S3 upload.
	ImageCacheDir string

	// MotherGooseURL is the MotherGoose API base URL for state reporting.
	MotherGooseURL string

	// APIKey is the MotherGoose API key.
	APIKey string

	// Schedule defines optional time windows when Rift should be running.
	// Empty slice means Rift is managed purely on-demand.
	Schedule []ScheduleWindow

	// AntiFlap is the minimum duration Rift must stay in its current state
	// before transitioning to off/destroy, even if no runners are connected.
	// Prevents rapid on/off cycling when requests arrive near shutdown time.
	AntiFlap time.Duration

	// IdleTimeout is how long Rift waits with zero active connections before
	// initiating a shutdown (if not within a schedule window).
	IdleTimeout time.Duration

	// CacheSyncInterval is how often running Rift syncs the image cache to S3.
	CacheSyncInterval time.Duration

	// EggSSHKeyFingerprints maps egg name → SHA256 fingerprint of the SSH public key.
	// Only the fingerprint is stored in the database under each egg's record.
	// The actual public key is stored in secret storage and referenced via EggSSHKeySecretURIs.
	EggSSHKeyFingerprints map[string]string

	// EggSSHKeySecretURIs maps egg name → secret URI for the SSH public key
	// (e.g. "yc-lockbox://rift-ssh-keys/my-egg-public-key").
	// Injected into runner environment so runners can resolve the key at runtime.
	EggSSHKeySecretURIs map[string]string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:            ":2376",
		DockerSocketPath:      "/var/run/docker.sock",
		ImageCacheDir:         "/var/cache/rift/images",
		AntiFlap:              2 * time.Minute,
		IdleTimeout:           10 * time.Minute,
		CacheSyncInterval:     5 * time.Minute,
		EggSSHKeyFingerprints: make(map[string]string),
		EggSSHKeySecretURIs:   make(map[string]string),
	}
}
