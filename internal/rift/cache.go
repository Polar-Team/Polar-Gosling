package rift

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CacheEntry describes a cached Docker image tarball.
type CacheEntry struct {
	// ImageRef is the full image reference (e.g. "docker.io/library/alpine:3.18").
	ImageRef string
	// LocalPath is the path to the tarball on disk.
	LocalPath string
	// S3Key is the object key in the S3 bucket.
	S3Key string
	// Size is the tarball size in bytes.
	Size int64
	// CachedAt is when the entry was last written.
	CachedAt time.Time
	// UploadedAt is when the entry was last synced to S3 (zero if not yet uploaded).
	UploadedAt time.Time
}

// S3Uploader is the interface for writing objects to S3-compatible storage.
// Implementations wrap the AWS SDK or a compatible client.
type S3Uploader interface {
	// Upload writes r to the given key in the configured bucket.
	Upload(ctx context.Context, key string, r io.Reader, size int64) error
	// Download retrieves the object at key and writes it to w.
	Download(ctx context.Context, key string, w io.Writer) error
	// Exists reports whether the object at key exists.
	Exists(ctx context.Context, key string) (bool, error)
}

// ImageCache manages Docker image tarballs on local disk and syncs them to S3.
type ImageCache struct {
	mu       sync.RWMutex
	dir      string
	prefix   string
	uploader S3Uploader
	entries  map[string]*CacheEntry // keyed by ImageRef
}

// NewImageCache creates an ImageCache backed by the given local directory and S3 uploader.
func NewImageCache(dir, s3KeyPrefix string, uploader S3Uploader) *ImageCache {
	return &ImageCache{
		dir:      dir,
		prefix:   s3KeyPrefix,
		uploader: uploader,
		entries:  make(map[string]*CacheEntry),
	}
}

// imageRefToKey converts an image reference to a safe S3 key / filename.
func imageRefToKey(ref, prefix string) string {
	safe := strings.NewReplacer("/", "_", ":", "_", "@", "_").Replace(ref)
	return filepath.Join(prefix, safe+".tar")
}

// Get returns the local path for the cached image, downloading from S3 if needed.
// Returns ("", false, nil) when the image is not in cache and not in S3.
func (c *ImageCache) Get(ctx context.Context, imageRef string) (string, bool, error) {
	c.mu.RLock()
	entry, ok := c.entries[imageRef]
	c.mu.RUnlock()

	if ok {
		// Verify the file still exists on disk.
		if _, err := os.Stat(entry.LocalPath); err == nil {
			return entry.LocalPath, true, nil
		}
	}

	// Try to pull from S3.
	s3Key := imageRefToKey(imageRef, c.prefix)
	exists, err := c.uploader.Exists(ctx, s3Key)
	if err != nil || !exists {
		return "", false, err
	}

	localPath := filepath.Join(c.dir, filepath.Base(s3Key))
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return "", false, fmt.Errorf("rift cache: mkdir %s: %w", c.dir, err)
	}

	f, err := os.Create(localPath)
	if err != nil {
		return "", false, fmt.Errorf("rift cache: create %s: %w", localPath, err)
	}
	defer f.Close()

	if err := c.uploader.Download(ctx, s3Key, f); err != nil {
		_ = os.Remove(localPath)
		return "", false, fmt.Errorf("rift cache: download %s: %w", s3Key, err)
	}

	info, _ := f.Stat()
	var size int64
	if info != nil {
		size = info.Size()
	}

	c.mu.Lock()
	c.entries[imageRef] = &CacheEntry{
		ImageRef:   imageRef,
		LocalPath:  localPath,
		S3Key:      s3Key,
		Size:       size,
		CachedAt:   time.Now(),
		UploadedAt: time.Now(), // already in S3
	}
	c.mu.Unlock()

	return localPath, true, nil
}

// Put stores a Docker image tarball from r into the local cache and marks it
// for upload to S3 on the next Sync call.
func (c *ImageCache) Put(ctx context.Context, imageRef string, r io.Reader) error {
	s3Key := imageRefToKey(imageRef, c.prefix)
	localPath := filepath.Join(c.dir, filepath.Base(s3Key))

	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("rift cache: mkdir %s: %w", c.dir, err)
	}

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("rift cache: create %s: %w", localPath, err)
	}
	defer f.Close()

	size, err := io.Copy(f, r)
	if err != nil {
		_ = os.Remove(localPath)
		return fmt.Errorf("rift cache: write %s: %w", localPath, err)
	}

	c.mu.Lock()
	c.entries[imageRef] = &CacheEntry{
		ImageRef:  imageRef,
		LocalPath: localPath,
		S3Key:     s3Key,
		Size:      size,
		CachedAt:  time.Now(),
	}
	c.mu.Unlock()

	return nil
}

// Sync uploads all dirty (not yet uploaded) cache entries to S3.
// This must be called before shutdown to ensure no data is lost.
func (c *ImageCache) Sync(ctx context.Context) error {
	c.mu.Lock()
	dirty := make([]*CacheEntry, 0)
	for _, e := range c.entries {
		if e.UploadedAt.IsZero() {
			dirty = append(dirty, e)
		}
	}
	c.mu.Unlock()

	for _, e := range dirty {
		f, err := os.Open(e.LocalPath)
		if err != nil {
			return fmt.Errorf("rift cache: open %s for sync: %w", e.LocalPath, err)
		}

		if err := c.uploader.Upload(ctx, e.S3Key, f, e.Size); err != nil {
			f.Close()
			return fmt.Errorf("rift cache: upload %s: %w", e.S3Key, err)
		}
		f.Close()

		c.mu.Lock()
		e.UploadedAt = time.Now()
		c.mu.Unlock()
	}

	return nil
}

// Entries returns a snapshot of all cache entries.
func (c *ImageCache) Entries() []*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*CacheEntry, 0, len(c.entries))
	for _, e := range c.entries {
		cp := *e
		out = append(out, &cp)
	}
	return out
}
