package api

import (
	"os"
	"testing"
)

func TestNewProjectCache_ReturnsNonNil(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cs, err := NewProjectCache()
	if err != nil {
		t.Fatalf("NewProjectCache: %v", err)
	}
	if cs == nil {
		t.Fatal("expected non-nil ProjectCache")
	}
}

func TestNewProjectCache_CacheDirExists(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cs, err := NewProjectCache()
	if err != nil {
		t.Fatalf("NewProjectCache: %v", err)
	}
	if cs.CacheDir == "" {
		t.Error("CacheDir should not be empty")
	}
	if _, statErr := os.Stat(cs.CacheDir); statErr != nil {
		t.Errorf("CacheDir %q does not exist: %v", cs.CacheDir, statErr)
	}
}
