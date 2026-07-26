// Package ycstore implements bootstrap.MGSecretStore for Yandex Cloud Lockbox.
package ycstore

import (
	"context"
	"fmt"
	"strings"

	lockboxpb "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"

	"github.com/polar-gosling/gosling/internal/bootstrap"
)

const (
	// SecretName is the fixed name for the MG API secret in YC Lockbox.
	SecretName = "pg-mothergoose-secrets"

	// EntryKeyAPIURL is the Lockbox entry key for the API Gateway URL.
	EntryKeyAPIURL = "api-url"

	// EntryKeyAPIKey is the Lockbox entry key for the API key.
	EntryKeyAPIKey = "api-key"

	// LabelPolarGosling is the label key marking Polar Gosling resources.
	LabelPolarGosling = "polar-gosling"

	// LabelResourceType is the label key indicating the resource type.
	LabelResourceType = "resource-type"

	// SecretURIScheme is the URI scheme for YC Lockbox secrets.
	SecretURIScheme = "yc-lockbox"
)

// YCMGSecretStore implements bootstrap.MGSecretStore for Yandex Cloud Lockbox.
type YCMGSecretStore struct {
	sdk      *ycsdk.SDK
	folderID string
}

// New creates a new YCMGSecretStore.
func New(sdk *ycsdk.SDK, folderID string) *YCMGSecretStore {
	return &YCMGSecretStore{sdk: sdk, folderID: folderID}
}

// Discover searches for an existing MG API secret named "pg-mothergoose-secrets"
// in the configured folder. Returns nil, nil if no secret exists.
// Returns bootstrap.ErrSecretDeleted if the secret is in INACTIVE state (pending deletion).
func (s *YCMGSecretStore) Discover(ctx context.Context) (*bootstrap.MGSecret, error) {
	secret, err := s.findSecretByName(ctx, SecretName)
	if err != nil {
		if isPermissionDenied(err) {
			return nil, fmt.Errorf("listing secrets in folder %s: ensure the service account has the lockbox.viewer role: %w", s.folderID, err)
		}
		return nil, fmt.Errorf("listing secrets in folder %s: %w", s.folderID, err)
	}
	if secret == nil {
		return nil, nil
	}

	// Check if the secret is in a deletion/inactive state
	if secret.Status == lockboxpb.Secret_INACTIVE {
		return nil, bootstrap.ErrSecretDeleted
	}

	// Get payload to read current entry values
	payload, err := s.sdk.LockboxPayload().Payload().Get(ctx, &lockboxpb.GetPayloadRequest{
		SecretId: secret.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving payload for secret %s: %w", secret.Id, err)
	}

	creds := extractCredentials(payload)
	populated := creds.APIURL != "" && creds.APIKey != ""

	return &bootstrap.MGSecret{
		ID:          secret.Id,
		Credentials: creds,
		Populated:   populated,
	}, nil
}

// Create provisions a new MG API secret with empty placeholder entries for api-url and api-key.
// Labels the secret with polar-gosling=true and resource-type=mothergoose-api.
func (s *YCMGSecretStore) Create(ctx context.Context) (*bootstrap.MGSecret, error) {
	entries := []*lockboxpb.PayloadEntryChange{
		{
			Key:   EntryKeyAPIURL,
			Value: &lockboxpb.PayloadEntryChange_TextValue{TextValue: ""},
		},
		{
			Key:   EntryKeyAPIKey,
			Value: &lockboxpb.PayloadEntryChange_TextValue{TextValue: ""},
		},
	}

	op, err := s.sdk.WrapOperation(s.sdk.LockboxSecret().Secret().Create(ctx, &lockboxpb.CreateSecretRequest{
		FolderId: s.folderID,
		Name:     SecretName,
		Labels: map[string]string{
			LabelPolarGosling: "true",
			LabelResourceType: "mothergoose-api",
		},
		VersionPayloadEntries: entries,
	}))
	if err != nil {
		if isPermissionDenied(err) {
			return nil, fmt.Errorf("creating secret %q in folder %s: ensure the service account has the lockbox.editor role: %w", SecretName, s.folderID, err)
		}
		return nil, fmt.Errorf("creating secret %q in folder %s: %w", SecretName, s.folderID, err)
	}
	if err := op.Wait(ctx); err != nil {
		return nil, fmt.Errorf("waiting for secret creation: %w", err)
	}
	res, err := op.Response()
	if err != nil {
		return nil, fmt.Errorf("getting secret creation response: %w", err)
	}
	created, ok := res.(*lockboxpb.Secret)
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T from secret creation", res)
	}

	return &bootstrap.MGSecret{
		ID: created.Id,
		Credentials: bootstrap.Credentials{
			APIURL: "",
			APIKey: "",
		},
		Populated: false,
	}, nil
}

// Update adds a new version to the secret with populated api-url and api-key entries.
func (s *YCMGSecretStore) Update(ctx context.Context, secretID string, creds bootstrap.Credentials) error {
	entries := []*lockboxpb.PayloadEntryChange{
		{
			Key:   EntryKeyAPIURL,
			Value: &lockboxpb.PayloadEntryChange_TextValue{TextValue: creds.APIURL},
		},
		{
			Key:   EntryKeyAPIKey,
			Value: &lockboxpb.PayloadEntryChange_TextValue{TextValue: creds.APIKey},
		},
	}

	op, err := s.sdk.WrapOperation(s.sdk.LockboxSecret().Secret().AddVersion(ctx, &lockboxpb.AddVersionRequest{
		SecretId:       secretID,
		PayloadEntries: entries,
	}))
	if err != nil {
		if isPermissionDenied(err) {
			return fmt.Errorf("updating secret %s: ensure the service account has the lockbox.editor role: %w", secretID, err)
		}
		return fmt.Errorf("updating secret %s: %w", secretID, err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("waiting for secret version update: %w", err)
	}

	return nil
}

// SecretURI returns the full URI for the given secret ID using the yc-lockbox:// scheme.
func SecretURI(secretID string) string {
	return fmt.Sprintf("%s://%s", SecretURIScheme, secretID)
}

// findSecretByName searches for a secret with the given name in the folder.
func (s *YCMGSecretStore) findSecretByName(ctx context.Context, name string) (*lockboxpb.Secret, error) {
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

// extractCredentials reads api-url and api-key values from a Lockbox payload.
func extractCredentials(payload *lockboxpb.Payload) bootstrap.Credentials {
	var creds bootstrap.Credentials
	for _, entry := range payload.Entries {
		switch entry.Key {
		case EntryKeyAPIURL:
			creds.APIURL = entry.GetTextValue()
		case EntryKeyAPIKey:
			creds.APIKey = entry.GetTextValue()
		}
	}
	return creds
}

// isPermissionDenied checks if the error indicates a permission denied response.
func isPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "PermissionDenied") || strings.Contains(errStr, "permission denied")
}
