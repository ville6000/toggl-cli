package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ville6000/toggl-cli/internal/data"
	"os"
	"path/filepath"
	"time"
)

type CacheService struct {
	CacheDir string
}

func NewCacheService() (*CacheService, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(dir, "toggl-cli")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}

	return &CacheService{CacheDir: cacheDir}, nil
}

func (c *CacheService) GetCachePath(workspaceId int) (string, error) {
	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%d", workspaceId)))
	hashStr := hex.EncodeToString(hasher.Sum(nil))

	cacheFile := filepath.Join(c.CacheDir, fmt.Sprintf("projects_%s.json", hashStr))

	return cacheFile, nil
}

func (c *CacheService) SaveProjects(workspaceId int, projects []data.Project) error {
	cacheFile, err := c.GetCachePath(workspaceId)
	if err != nil {
		return err
	}

	cached := data.ProjectCache{
		Timestamp: time.Now(),
		Data:      projects,
	}

	content, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, content, 0o644)
}

func (c *CacheService) GetProjects(workspaceId int) ([]data.Project, error) {
	cacheFile, err := c.GetCachePath(workspaceId)
	if err != nil {
		return nil, err
	}

	fileContent, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cached data.ProjectCache
	if err := json.Unmarshal(fileContent, &cached); err != nil {
		return nil, err
	}

	if time.Since(cached.Timestamp) >= 24*time.Hour {
		return nil, fmt.Errorf("cache is outdated")
	}

	return cached.Data, nil
}
