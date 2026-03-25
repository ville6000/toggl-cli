package cache

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/data"
)

func newTestCache(t *testing.T) *CacheService {
	t.Helper()
	return &CacheService{CacheDir: t.TempDir()}
}

// ---------- GetCachePath ----------

func TestGetCachePath_Deterministic(t *testing.T) {
	cs := newTestCache(t)
	p1, err1 := cs.GetCachePath(123)
	p2, err2 := cs.GetCachePath(123)
	if err1 != nil || err2 != nil {
		t.Fatalf("GetCachePath errors: %v, %v", err1, err2)
	}
	if p1 != p2 {
		t.Errorf("expected same path for same workspace, got %q and %q", p1, p2)
	}
}

func TestGetCachePath_DifferentWorkspaces(t *testing.T) {
	cs := newTestCache(t)
	p1, _ := cs.GetCachePath(1)
	p2, _ := cs.GetCachePath(2)
	if p1 == p2 {
		t.Error("different workspaces must produce different cache paths")
	}
}

func TestGetCachePath_ContainsCacheDir(t *testing.T) {
	cs := newTestCache(t)
	path, err := cs.GetCachePath(10)
	if err != nil {
		t.Fatalf("GetCachePath: %v", err)
	}
	if len(path) == 0 {
		t.Error("expected non-empty path")
	}
	// Path should be inside the configured cache dir.
	if path[:len(cs.CacheDir)] != cs.CacheDir {
		t.Errorf("path %q is not inside CacheDir %q", path, cs.CacheDir)
	}
}

// ---------- SaveProjects / GetProjects round-trip ----------

func TestSaveAndGetProjects(t *testing.T) {
	cs := newTestCache(t)
	projects := []data.Project{
		{ID: 1, Name: "Alpha"},
		{ID: 2, Name: "Beta"},
	}

	if err := cs.SaveProjects(10, projects); err != nil {
		t.Fatalf("SaveProjects: %v", err)
	}

	got, err := cs.GetProjects(10)
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}

	if len(got) != len(projects) {
		t.Fatalf("got %d projects, want %d", len(got), len(projects))
	}
	for i := range projects {
		if got[i] != projects[i] {
			t.Errorf("project[%d]: got %+v, want %+v", i, got[i], projects[i])
		}
	}
}

func TestSaveProjects_EmptySlice(t *testing.T) {
	cs := newTestCache(t)
	if err := cs.SaveProjects(1, []data.Project{}); err != nil {
		t.Fatalf("SaveProjects: %v", err)
	}
	got, err := cs.GetProjects(1)
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}

func TestSaveProjects_OverwritesPreviousCache(t *testing.T) {
	cs := newTestCache(t)

	original := []data.Project{{ID: 1, Name: "Original"}}
	if err := cs.SaveProjects(5, original); err != nil {
		t.Fatalf("SaveProjects (original): %v", err)
	}

	updated := []data.Project{{ID: 2, Name: "Updated"}}
	if err := cs.SaveProjects(5, updated); err != nil {
		t.Fatalf("SaveProjects (updated): %v", err)
	}

	got, err := cs.GetProjects(5)
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Updated" {
		t.Errorf("expected updated cache, got %+v", got)
	}
}

func TestSaveProjects_IsolatedByWorkspace(t *testing.T) {
	cs := newTestCache(t)
	p1 := []data.Project{{ID: 1, Name: "WS-1"}}
	p2 := []data.Project{{ID: 2, Name: "WS-2"}}

	if err := cs.SaveProjects(100, p1); err != nil {
		t.Fatalf("SaveProjects ws 100: %v", err)
	}
	if err := cs.SaveProjects(200, p2); err != nil {
		t.Fatalf("SaveProjects ws 200: %v", err)
	}

	got1, _ := cs.GetProjects(100)
	got2, _ := cs.GetProjects(200)

	if len(got1) != 1 || got1[0].Name != "WS-1" {
		t.Errorf("ws 100: expected WS-1, got %+v", got1)
	}
	if len(got2) != 1 || got2[0].Name != "WS-2" {
		t.Errorf("ws 200: expected WS-2, got %+v", got2)
	}
}

// ---------- GetProjects: cache miss ----------

func TestGetProjects_CacheMiss(t *testing.T) {
	cs := newTestCache(t)
	if _, err := cs.GetProjects(999); err == nil {
		t.Error("expected error for cache miss (file does not exist)")
	}
}

// ---------- GetProjects: cache expiry ----------

func TestGetProjects_CacheExpired(t *testing.T) {
	cs := newTestCache(t)
	projects := []data.Project{{ID: 1, Name: "Stale"}}

	cacheFile, err := cs.GetCachePath(42)
	if err != nil {
		t.Fatalf("GetCachePath: %v", err)
	}

	expired := data.ProjectCache{
		Timestamp: time.Now().Add(-25 * time.Hour),
		Data:      projects,
	}
	content, err := json.MarshalIndent(expired, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cacheFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := cs.GetProjects(42); err == nil {
		t.Error("expected error for expired cache (>24h old)")
	}
}

func TestGetProjects_CacheJustUnderTTL(t *testing.T) {
	cs := newTestCache(t)
	projects := []data.Project{{ID: 1, Name: "Fresh"}}

	cacheFile, err := cs.GetCachePath(7)
	if err != nil {
		t.Fatalf("GetCachePath: %v", err)
	}

	// Just under 24 hours — should still be valid.
	fresh := data.ProjectCache{
		Timestamp: time.Now().Add(-23*time.Hour - 59*time.Minute),
		Data:      projects,
	}
	content, err := json.MarshalIndent(fresh, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cacheFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := cs.GetProjects(7)
	if err != nil {
		t.Fatalf("GetProjects: %v (expected valid cache)", err)
	}
	if len(got) != 1 || got[0].Name != "Fresh" {
		t.Errorf("unexpected projects: %+v", got)
	}
}

func TestGetProjects_CacheExactlyAtTTL(t *testing.T) {
	cs := newTestCache(t)
	projects := []data.Project{{ID: 1, Name: "Boundary"}}

	cacheFile, err := cs.GetCachePath(8)
	if err != nil {
		t.Fatalf("GetCachePath: %v", err)
	}

	// Exactly 24 hours ago — should be considered expired (>= 24h).
	boundary := data.ProjectCache{
		Timestamp: time.Now().Add(-24 * time.Hour),
		Data:      projects,
	}
	content, err := json.MarshalIndent(boundary, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cacheFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// The condition is `>= 24h`, so this should be expired.
	if _, err := cs.GetProjects(8); err == nil {
		t.Error("expected error: cache at exactly 24h should be considered expired")
	}
}

// ---------- Corrupted cache file ----------

func TestGetProjects_CorruptedCacheFile(t *testing.T) {
	cs := newTestCache(t)

	cacheFile, err := cs.GetCachePath(55)
	if err != nil {
		t.Fatalf("GetCachePath: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("not valid json {{{{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := cs.GetProjects(55); err == nil {
		t.Error("expected error for corrupted cache file")
	}
}
