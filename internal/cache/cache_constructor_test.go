package cache

import (
	"os"
	"testing"
)

func TestNewService_ReturnsNonNil(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cs, err := NewService()
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if cs == nil {
		t.Fatal("expected non-nil Service")
	}
}

func TestNewService_CacheDirExists(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cs, err := NewService()
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if cs.CacheDir == "" {
		t.Error("CacheDir should not be empty")
	}
	if _, statErr := os.Stat(cs.CacheDir); statErr != nil {
		t.Errorf("CacheDir %q does not exist: %v", cs.CacheDir, statErr)
	}
}
