package rift

import (
	"crypto/md5" //nolint:gosec // MD5 is used only for SSH fingerprint display, not security
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
)

// SSHKeyEntry holds the DB-stored fingerprint and the secret URI that points
// to the actual SSH public key in secret storage (YC Lockbox / AWS SM / Vault).
//
// Design rationale:
//   - The DB stores only the SHA256 fingerprint (safe to store, used for lookup/audit).
//   - The actual public key is retrieved at runtime from the secret URI.
//   - Runners receive the secret URI; they resolve the key themselves.
type SSHKeyEntry struct {
	// EggName is the Egg this key belongs to.
	EggName string
	// Fingerprint is the SHA256 fingerprint of the public key (e.g. "SHA256:abc123...").
	// This is what is stored in the database under the egg's record.
	Fingerprint string
	// SecretURI is the secret storage reference for the actual public key
	// (e.g. "yc-lockbox://rift-ssh-keys/my-egg-public-key").
	// Runners use this URI to retrieve the key at runtime.
	SecretURI string
}

// SSHKeyStore manages per-Egg SSH key entries for the Rift VM.
//
// What lives where:
//   - Database  → fingerprint only (SSHKeyEntry.Fingerprint), stored under egg info
//   - Secret storage → actual SSH public key, referenced by SSHKeyEntry.SecretURI
//   - Runner env → SecretURI injected so the runner can resolve the key itself
type SSHKeyStore struct {
	mu      sync.RWMutex
	entries map[string]*SSHKeyEntry // egg name → entry
}

// NewSSHKeyStore creates an empty SSHKeyStore.
func NewSSHKeyStore() *SSHKeyStore {
	return &SSHKeyStore{entries: make(map[string]*SSHKeyEntry)}
}

// Set stores or replaces the key entry for the given egg.
// fingerprint must be a non-empty SHA256 fingerprint string.
// secretURI must be a non-empty secret URI (yc-lockbox://, aws-sm://, vault://).
func (s *SSHKeyStore) Set(eggName, fingerprint, secretURI string) error {
	if eggName == "" {
		return fmt.Errorf("rift sshkeys: egg name must not be empty")
	}
	if fingerprint == "" {
		return fmt.Errorf("rift sshkeys: fingerprint for egg %q must not be empty", eggName)
	}
	if secretURI == "" {
		return fmt.Errorf("rift sshkeys: secret URI for egg %q must not be empty", eggName)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[eggName] = &SSHKeyEntry{
		EggName:     eggName,
		Fingerprint: fingerprint,
		SecretURI:   secretURI,
	}
	return nil
}

// Get returns the key entry for the given egg, and whether it was found.
func (s *SSHKeyStore) Get(eggName string) (*SSHKeyEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[eggName]
	if !ok {
		return nil, false
	}
	cp := *e
	return &cp, true
}

// Delete removes the key entry for the given egg.
func (s *SSHKeyStore) Delete(eggName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, eggName)
}

// All returns a snapshot of all entries.
func (s *SSHKeyStore) All() []*SSHKeyEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*SSHKeyEntry, 0, len(s.entries))
	for _, e := range s.entries {
		cp := *e
		out = append(out, &cp)
	}
	return out
}

// FingerprintMap returns a snapshot of egg name → fingerprint, suitable for
// writing to the database under each egg's record.
func (s *SSHKeyStore) FingerprintMap() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.entries))
	for egg, e := range s.entries {
		out[egg] = e.Fingerprint
	}
	return out
}

// SecretURIMap returns a snapshot of egg name → secret URI, used when
// injecting the key reference into runner environment variables.
func (s *SSHKeyStore) SecretURIMap() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.entries))
	for egg, e := range s.entries {
		out[egg] = e.SecretURI
	}
	return out
}

// FingerprintFromPublicKey computes the SHA256 fingerprint of a raw SSH public
// key string (the "ssh-rsa AAAA..." or "ecdsa-sha2-nistp256 AAAA..." format).
// The result matches the format produced by `ssh-keygen -l -E sha256`.
func FingerprintFromPublicKey(publicKey string) (string, error) {
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return "", fmt.Errorf("rift sshkeys: invalid public key format")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("rift sshkeys: failed to decode public key: %w", err)
	}
	sum := sha256.Sum256(keyBytes)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

// MD5FingerprintFromPublicKey computes the legacy MD5 fingerprint (colon-hex format).
// Used only for display/compatibility; SHA256 is preferred for storage.
func MD5FingerprintFromPublicKey(publicKey string) (string, error) {
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return "", fmt.Errorf("rift sshkeys: invalid public key format")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("rift sshkeys: failed to decode public key: %w", err)
	}
	//nolint:gosec // MD5 used only for legacy SSH fingerprint display
	sum := md5.Sum(keyBytes)
	parts2 := make([]string, len(sum))
	for i, b := range sum {
		parts2[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts2, ":"), nil
}
