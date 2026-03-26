package app

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// ResourceEntry matches the server's manifest entry format.
type ResourceEntry struct {
	Name   string `json:"name"`
	SHA512 string `json:"sha512"`
	Size   int64  `json:"size"`
}

// ResourceCache manages locally cached resources downloaded from the server.
// Resources are stored in ~/.cache/agach/resources/.
type ResourceCache struct {
	dir      string
	manifest map[string]string // name → sha512
	mu       sync.RWMutex
	logger   *logrus.Logger
}

// NewResourceCache creates a cache at ~/.cache/agach/resources/.
func NewResourceCache(logger *logrus.Logger) (*ResourceCache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get user cache dir: %w", err)
	}
	dir := filepath.Join(cacheDir, "agach", "resources")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create resource cache dir: %w", err)
	}
	return &ResourceCache{
		dir:      dir,
		manifest: make(map[string]string),
		logger:   logger,
	}, nil
}

// Sync compares the server manifest against local cache and downloads
// any resources whose SHA-512 differs or are missing.
func (rc *ResourceCache) Sync(ctx context.Context, entries []ResourceEntry, download func(ctx context.Context, name string) ([]byte, error)) {
	for _, entry := range entries {
		rc.mu.RLock()
		currentHash := rc.manifest[entry.Name]
		rc.mu.RUnlock()

		if currentHash == entry.SHA512 {
			continue
		}

		// Check if the file on disk already matches
		localPath := filepath.Join(rc.dir, entry.Name)
		if diskHash, err := hashFile(localPath); err == nil && diskHash == entry.SHA512 {
			rc.mu.Lock()
			rc.manifest[entry.Name] = entry.SHA512
			rc.mu.Unlock()
			rc.logger.WithField("resource", entry.Name).Debug("Resource already up to date on disk")
			continue
		}

		// Download the resource
		rc.logger.WithField("resource", entry.Name).Info("Downloading resource")
		data, err := download(ctx, entry.Name)
		if err != nil {
			rc.logger.WithError(err).WithField("resource", entry.Name).Error("Failed to download resource")
			continue
		}

		// Verify hash
		h := sha512.Sum512(data)
		downloadedHash := hex.EncodeToString(h[:])
		if downloadedHash != entry.SHA512 {
			rc.logger.WithFields(logrus.Fields{
				"resource": entry.Name,
				"expected": entry.SHA512,
				"got":      downloadedHash,
			}).Error("Resource hash mismatch after download")
			continue
		}

		// Write to cache
		if err := os.WriteFile(localPath, data, 0755); err != nil {
			rc.logger.WithError(err).WithField("resource", entry.Name).Error("Failed to write resource to cache")
			continue
		}

		rc.mu.Lock()
		rc.manifest[entry.Name] = entry.SHA512
		rc.mu.Unlock()

		rc.logger.WithFields(logrus.Fields{
			"resource": entry.Name,
			"size":     len(data),
		}).Info("Resource cached")
	}
}

// GetPath returns the full path to a cached resource.
func (rc *ResourceCache) GetPath(name string) string {
	return filepath.Join(rc.dir, name)
}

// hashFile computes the SHA-512 hex digest of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
