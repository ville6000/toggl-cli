package cache

import (
	"os"
	"testing"
)

func TestNewCacheService_ReturnsNonNil(t *testing.T) {
	cs, err := NewCacheService()
	if err != nil {
		t.Fatalf("NewCacheService: %v", err)
	}
	if cs == nil {
		t.Fatal("expected non-nil CacheService")
	}
}

func TestNewCacheService_CacheDirExists(t *testing.T) {
	cs, err := NewCacheService()
	if err != nil {
		t.Fatalf("NewCacheService: %v", err)
	}
	if cs.CacheDir == "" {
		t.Error("CacheDir should not be empty")
	}
	if _, statErr := os.Stat(cs.CacheDir); statErr != nil {
		t.Errorf("CacheDir %q does not exist: %v", cs.CacheDir, statErr)
	}
}
