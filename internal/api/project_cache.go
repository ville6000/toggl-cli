package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProjectCache caches each workspace's project list as a JSON file in
// CacheDir, so commands don't refetch it on every run.
type ProjectCache struct {
	CacheDir string
}

// cachedProjects is the on-disk format of a cached project list.
type cachedProjects struct {
	Timestamp time.Time `json:"timestamp"`
	Data      []Project `json:"data"`
}

// NewProjectCache returns a ProjectCache storing its files under the user's
// cache directory, creating it if needed.
func NewProjectCache() (*ProjectCache, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(dir, "toggl-cli")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, err
	}

	return &ProjectCache{CacheDir: cacheDir}, nil
}

// Path returns the cache file for workspaceID.
func (c *ProjectCache) Path(workspaceID int) (string, error) {
	sum := sha256.Sum256(fmt.Appendf(nil, "%d", workspaceID))
	hashStr := hex.EncodeToString(sum[:])

	cacheFile := filepath.Join(c.CacheDir, fmt.Sprintf("projects_%s.json", hashStr))

	return cacheFile, nil
}

// SaveProjects writes the projects for workspaceID to the cache, stamped with
// the current time.
func (c *ProjectCache) SaveProjects(workspaceID int, projects []Project) error {
	cacheFile, err := c.Path(workspaceID)
	if err != nil {
		return err
	}

	cached := cachedProjects{
		Timestamp: time.Now(),
		Data:      projects,
	}

	content, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, content, 0o600)
}

// Projects returns the cached projects for workspaceID, or an error when there
// is no cache file or it is 24 hours old or more.
func (c *ProjectCache) Projects(workspaceID int) ([]Project, error) {
	cacheFile, err := c.Path(workspaceID)
	if err != nil {
		return nil, err
	}

	fileContent, err := os.ReadFile(cacheFile) // #nosec G304 -- path is built by Path inside CacheDir
	if err != nil {
		return nil, err
	}

	var cached cachedProjects
	if err := json.Unmarshal(fileContent, &cached); err != nil {
		return nil, err
	}

	if time.Since(cached.Timestamp) >= 24*time.Hour {
		return nil, errors.New("cache is outdated")
	}

	return cached.Data, nil
}
