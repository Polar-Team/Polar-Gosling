package lockbox

import (
	"context"
	"fmt"

	lockboxpb "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
)

// YCLockboxStore implements SecretStore for Yandex Cloud Lockbox.
type YCLockboxStore struct {
	sdk      *ycsdk.SDK
	folderID string
}

// NewYCLockboxStore creates a new YCLockboxStore.
func NewYCLockboxStore(sdk *ycsdk.SDK, folderID string) *YCLockboxStore {
	return &YCLockboxStore{sdk: sdk, folderID: folderID}
}

// Create provisions a new Yandex Cloud Lockbox secret with placeholder entries.
func (s *YCLockboxStore) Create(ctx context.Context, params CreateParams) (*CreateResult, error) {
	secretName := SecretNameForProvider("yandex", params.EggName)

	// Check for existing secret with the same name
	existing, err := s.findSecretByName(ctx, secretName)
	if err != nil {
		return nil, fmt.Errorf("checking for existing secret: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf(
			"secret %q already exists in folder %s (id: %s). Use 'gosling lockbox verify --secret-id %s' to check its entries",
			secretName, s.folderID, existing.Id, existing.Id,
		)
	}

	// Build payload entries with empty placeholder values
	entries := make([]*lockboxpb.PayloadEntryChange, 0, len(requiredEntries))
	for _, key := range requiredEntries {
		entries = append(entries, &lockboxpb.PayloadEntryChange{
			Key:   key,
			Value: &lockboxpb.PayloadEntryChange_TextValue{TextValue: ""},
		})
	}

	op, err := s.sdk.WrapOperation(s.sdk.LockboxSecret().Secret().Create(ctx, &lockboxpb.CreateSecretRequest{
		FolderId: s.folderID,
		Name:     secretName,
		Labels: map[string]string{
			"polar-gosling": "true",
			"egg-name":      params.EggName,
		},
		VersionPayloadEntries: entries,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create Lockbox secret: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return nil, fmt.Errorf("waiting for Lockbox secret creation: %w", err)
	}
	res, err := op.Response()
	if err != nil {
		return nil, fmt.Errorf("getting Lockbox secret response: %w", err)
	}
	secret, ok := res.(*lockboxpb.Secret)
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T from Lockbox secret creation", res)
	}

	return &CreateResult{
		ID:   secret.Id,
		URIs: GenerateAllURIs("yandex", secret.Id),
	}, nil
}

// List returns all Polar Gosling-tagged Lockbox secrets in the folder.
func (s *YCLockboxStore) List(ctx context.Context) ([]SecretInfo, error) {
	var results []SecretInfo
	pageToken := ""

	for {
		resp, err := s.sdk.LockboxSecret().Secret().List(ctx, &lockboxpb.ListSecretsRequest{
			FolderId:  s.folderID,
			PageSize:  1000,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing Lockbox secrets: %w", err)
		}

		for _, secret := range resp.Secrets {
			if secret.Labels["polar-gosling"] != "true" {
				continue
			}
			info := SecretInfo{
				Name:    secret.Name,
				ID:      secret.Id,
				EggName: secret.Labels["egg-name"],
			}
			if secret.CreatedAt != nil {
				info.CreatedAt = secret.CreatedAt.AsTime()
			}
			results = append(results, info)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return results, nil
}

// Verify checks that a Lockbox secret exists and contains all RequiredEntries.
func (s *YCLockboxStore) Verify(ctx context.Context, secretRef string) (*VerifyResult, error) {
	payload, err := s.sdk.LockboxPayload().Payload().Get(ctx, &lockboxpb.GetPayloadRequest{
		SecretId: secretRef,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving Lockbox payload for secret %q: %w", secretRef, err)
	}

	presentKeys := make(map[string]bool, len(payload.Entries))
	for _, entry := range payload.Entries {
		presentKeys[entry.Key] = true
	}

	result := &VerifyResult{}
	for _, key := range requiredEntries {
		if presentKeys[key] {
			result.Present = append(result.Present, key)
		} else {
			result.Missing = append(result.Missing, key)
		}
	}

	return result, nil
}

// findSecretByName searches for a secret with the given name in the folder.
func (s *YCLockboxStore) findSecretByName(ctx context.Context, name string) (*lockboxpb.Secret, error) {
	pageToken := ""
	for {
		resp, err := s.sdk.LockboxSecret().Secret().List(ctx, &lockboxpb.ListSecretsRequest{
			FolderId:  s.folderID,
			PageSize:  1000,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, secret := range resp.Secrets {
			if secret.Name == name {
				return secret, nil
			}
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return nil, nil
}
