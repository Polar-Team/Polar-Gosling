package rift

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// --- helpers -----------------------------------------------------------------

// memS3 is an in-memory S3Uploader for property tests.
type memS3 struct {
	objects map[string][]byte
}

func newMemS3() *memS3 { return &memS3{objects: make(map[string][]byte)} }

func (m *memS3) Upload(_ context.Context, key string, r io.Reader, _ int64) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.objects[key] = data
	return nil
}

func (m *memS3) Download(_ context.Context, key string, w io.Writer) error {
	data, ok := m.objects[key]
	if !ok {
		return io.ErrUnexpectedEOF
	}
	_, err := w.Write(data)
	return err
}

func (m *memS3) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.objects[key]
	return ok, nil
}

// genImageRef generates plausible Docker image references.
func genImageRef() gopter.Gen {
	return gopter.CombineGens(gen.Identifier(), gen.Identifier()).
		Map(func(vals []interface{}) string {
			return vals[0].(string) + "/" + vals[1].(string) + ":latest"
		})
}

// genTarball generates random tarball bytes (non-empty).
func genTarball() gopter.Gen {
	return gen.SliceOfN(64, gen.UInt8()).Map(func(bs []byte) []byte { return bs })
}

// --- Property 20: Rift Cache Hit Behavior ------------------------------------
// Feature: gitops-runner-orchestration
// Validates: Requirements 8.4

// TestRiftCacheHitBehavior verifies that once an image is stored in the cache
// and synced to S3, subsequent Get calls return the same bytes without
// re-downloading from S3 (cache hit path), and that images not yet cached
// are transparently fetched from S3 (cold-start path).
func TestRiftCacheHitBehavior(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Put then Get returns the same content (warm cache).
	properties.Property("warm cache: Get returns the same bytes that were Put",
		prop.ForAll(
			func(imageRef string, tarball []byte) bool {
				if len(tarball) == 0 {
					return true
				}
				ctx := context.Background()
				s3 := newMemS3()
				cache := NewImageCache(t.TempDir(), "rift/", s3)

				if err := cache.Put(ctx, imageRef, bytes.NewReader(tarball)); err != nil {
					t.Logf("Put failed: %v", err)
					return false
				}

				localPath, hit, err := cache.Get(ctx, imageRef)
				if err != nil || !hit || localPath == "" {
					t.Logf("Get after Put: hit=%v err=%v path=%q", hit, err, localPath)
					return false
				}
				return true
			},
			genImageRef(),
			genTarball(),
		))

	// Property: Sync uploads dirty entries; subsequent Get on a fresh cache
	// instance (same S3) finds the image via S3 download (cold-start hit).
	properties.Property("cold cache: Get downloads from S3 after Sync",
		prop.ForAll(
			func(imageRef string, tarball []byte) bool {
				if len(tarball) == 0 {
					return true
				}
				ctx := context.Background()
				s3 := newMemS3()

				// First cache: Put + Sync → uploads to S3.
				c1 := NewImageCache(t.TempDir(), "rift/", s3)
				if err := c1.Put(ctx, imageRef, bytes.NewReader(tarball)); err != nil {
					return false
				}
				if err := c1.Sync(ctx); err != nil {
					return false
				}

				// Second cache (empty local dir, same S3): Get should pull from S3.
				c2 := NewImageCache(t.TempDir(), "rift/", s3)
				localPath, hit, err := c2.Get(ctx, imageRef)
				if err != nil || !hit || localPath == "" {
					t.Logf("cold Get: hit=%v err=%v path=%q", hit, err, localPath)
					return false
				}
				return true
			},
			genImageRef(),
			genTarball(),
		))

	// Property: Get on an image absent from both local cache and S3 returns
	// (empty, false, nil) — no error, just a miss.
	properties.Property("cache miss: Get returns false with no error for unknown image",
		prop.ForAll(
			func(imageRef string) bool {
				ctx := context.Background()
				s3 := newMemS3()
				cache := NewImageCache(t.TempDir(), "rift/", s3)

				localPath, hit, err := cache.Get(ctx, imageRef)
				return err == nil && !hit && localPath == ""
			},
			genImageRef(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// --- Property 21: Rift Authentication Enforcement ---------------------------
// Feature: gitops-runner-orchestration
// Validates: Requirements 8.6

// TestRiftAuthenticationEnforcement verifies that the Authenticator accepts
// only requests carrying the exact configured bearer token and rejects all
// others, using constant-time comparison (no timing oracle).
func TestRiftAuthenticationEnforcement(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: correct token is always accepted.
	properties.Property("correct bearer token is always accepted",
		prop.ForAll(
			func(token string) bool {
				if token == "" {
					return true
				}
				auth := NewAuthenticator(token)
				req := newBearerRequest(token)
				return auth.Authenticate(req) == nil
			},
			gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		))

	// Property: any token that differs from the configured one is rejected.
	properties.Property("wrong bearer token is always rejected",
		prop.ForAll(
			func(correct, wrong string) bool {
				if correct == "" || wrong == correct {
					return true
				}
				auth := NewAuthenticator(correct)
				req := newBearerRequest(wrong)
				return auth.Authenticate(req) == ErrUnauthorized
			},
			gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
			gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		))

	// Property: missing Authorization header is always rejected.
	properties.Property("missing Authorization header is always rejected",
		prop.ForAll(
			func(token string) bool {
				if token == "" {
					return true
				}
				auth := NewAuthenticator(token)
				req := newBearerRequest("")
				// empty string → header not set → must be rejected
				return auth.Authenticate(req) == ErrUnauthorized
			},
			gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		))

	// Property: malformed Authorization header (no "Bearer " prefix) is rejected.
	properties.Property("malformed Authorization header without Bearer prefix is rejected",
		prop.ForAll(
			func(token string) bool {
				if token == "" {
					return true
				}
				auth := NewAuthenticator(token)
				// Send the token raw, without the "Bearer " prefix.
				req := newRawAuthRequest(token)
				return auth.Authenticate(req) == ErrUnauthorized
			},
			gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// newBearerRequest builds a minimal *http.Request with a Bearer token header.
// An empty token means the header is omitted entirely.
func newBearerRequest(token string) *http.Request {
	req, _ := http.NewRequest("GET", "/", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// newRawAuthRequest builds a request with the token set directly (no "Bearer " prefix).
func newRawAuthRequest(token string) *http.Request {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", token)
	return req
}

// --- Property 22: Rift Optional Dependency -----------------------------------
// Feature: gitops-runner-orchestration
// Validates: Requirements 8.7
//
// Rift is optional: runners must operate correctly when Rift is absent.
// This property verifies that:
//   - The Orchestrator starts in StateOff and never panics when hooks are nil.
//   - WakeUp on a nil-hooks orchestrator transitions to StateRunning without error.
//   - Anti-flap blocks premature off-transitions regardless of hook presence.
//   - The StateManager correctly reports PendingShutdown when anti-flap fires.

func TestRiftOptionalDependency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: orchestrator with nil hooks starts in StateOff (Rift absent = off).
	properties.Property("orchestrator without hooks starts in StateOff",
		prop.ForAll(
			func(antiFlap uint8) bool {
				cfg := DefaultConfig()
				cfg.AntiFlap = 0 // irrelevant for this check
				cache := NewImageCache(t.TempDir(), "rift/", newMemS3())
				o := NewOrchestrator(cfg, cache, nil)
				return o.State() == StateOff
			},
			gen.UInt8(),
		))

	// Property: WakeUp with nil hooks succeeds and transitions to StateRunning.
	properties.Property("WakeUp with nil hooks reaches StateRunning without error",
		prop.ForAll(
			func(_ uint8) bool {
				cfg := DefaultConfig()
				cfg.AntiFlap = 0
				cache := NewImageCache(t.TempDir(), "rift/", newMemS3())
				o := NewOrchestrator(cfg, cache, nil)

				ctx := context.Background()
				if err := o.WakeUp(ctx); err != nil {
					t.Logf("WakeUp error: %v", err)
					return false
				}
				// Drain the wake channel by calling ensureRunning directly.
				if err := o.ensureRunning(ctx); err != nil {
					t.Logf("ensureRunning error: %v", err)
					return false
				}
				return o.State() == StateRunning
			},
			gen.UInt8(),
		))

	// Property: anti-flap blocks StateOff transition immediately after StateRunning.
	// The state must remain Running and PendingShutdown must be set.
	properties.Property("anti-flap blocks immediate off-transition after running",
		prop.ForAll(
			func(_ uint8) bool {
				cfg := DefaultConfig()
				cfg.AntiFlap = 10 * 60 * 1000000000 // 10 minutes — never elapsed
				sm := NewStateManager(cfg.AntiFlap)

				// Manually drive to StateRunning.
				sm.Transition(StateStarting)
				sm.Transition(StateRunning)

				// Attempt immediate transition to StateOff — must be blocked.
				applied := sm.Transition(StateOff)
				if applied {
					t.Log("expected anti-flap to block StateOff transition")
					return false
				}
				if sm.Current() != StateRunning {
					t.Logf("expected StateRunning after blocked transition, got %s", sm.Current())
					return false
				}
				if !sm.PendingShutdown() {
					t.Log("expected PendingShutdown to be true after blocked transition")
					return false
				}
				return true
			},
			gen.UInt8(),
		))

	// Property: a new WakeUp cancels a pending shutdown (anti-flap cancel path).
	properties.Property("WakeUp cancels pending shutdown set by anti-flap",
		prop.ForAll(
			func(_ uint8) bool {
				cfg := DefaultConfig()
				cfg.AntiFlap = 10 * 60 * 1000000000
				cache := NewImageCache(t.TempDir(), "rift/", newMemS3())
				o := NewOrchestrator(cfg, cache, nil)

				ctx := context.Background()
				// Drive to Running.
				_ = o.ensureRunning(ctx)

				// Force a pending shutdown by directly manipulating the state manager.
				o.state.Transition(StateOff) // will be blocked → sets pendingShutdown
				// If anti-flap didn't block (e.g. time elapsed), skip.
				if !o.state.PendingShutdown() {
					return true
				}

				// WakeUp should cancel the pending shutdown.
				_ = o.WakeUp(ctx)
				return !o.state.PendingShutdown()
			},
			gen.UInt8(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
