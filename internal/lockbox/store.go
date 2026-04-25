package lockbox

import (
	"context"
	"time"
)

// requiredEntries is the internal backing slice — not exported to prevent mutation.
var requiredEntries = []string{"runner-token", "webhook-secret", "repo-url"}

// RequiredEntries returns a copy of the required secret entry keys every egg needs.
func RequiredEntries() []string {
	out := make([]string, len(requiredEntries))
	copy(out, requiredEntries)
	return out
}

// CreateParams holds input for creating a new secret store.
type CreateParams struct {
	Provider string // "yandex" or "aws"
	EggName  string
	FolderID string // YC only
	Region   string // AWS only
}

// CreateResult holds the output of a successful secret creation.
type CreateResult struct {
	ID   string            // Secret UUID (YC) or secret name (AWS)
	URIs map[string]string // key -> full secret URI
}

// SecretInfo represents a single secret store entry in list output.
type SecretInfo struct {
	Name      string
	ID        string // Secret UUID (YC) or ARN (AWS)
	EggName   string // From egg-name label/tag
	CreatedAt time.Time
}

// VerifyResult holds the output of a secret verification.
type VerifyResult struct {
	Present []string
	Missing []string
}

// SecretStore abstracts cloud secret store operations.
type SecretStore interface {
	// Create provisions a new secret store with placeholder entries.
	Create(ctx context.Context, params CreateParams) (*CreateResult, error)

	// List returns all Polar Gosling-tagged secrets.
	List(ctx context.Context) ([]SecretInfo, error)

	// Verify checks that a secret exists and contains all RequiredEntries.
	Verify(ctx context.Context, secretRef string) (*VerifyResult, error)
}
