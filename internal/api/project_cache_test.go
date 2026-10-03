package api

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *ProjectCache {
	t.Helper()
	return &ProjectCache{CacheDir: t.TempDir()}
}

// ---------- Path ----------

func TestPath_Deterministic(t *testing.T) {
	cs := newTestCache(t)
	p1, err1 := cs.Path(123)
	p2, err2 := cs.Path(123)
	if err1 != nil || err2 != nil {
		t.Fatalf("Path errors: %v, %v", err1, err2)
	}
	if p1 != p2 {
		t.Errorf("expected same path for same workspace, got %q and %q", p1, p2)
	}
}

func TestPath_DifferentWorkspaces(t *testing.T) {
	cs := newTestCache(t)
	p1, _ := cs.Path(1)
	p2, _ := cs.Path(2)
	if p1 == p2 {
		t.Error("different workspaces must produce different cache paths")
	}
}

func TestPath_ContainsCacheDir(t *testing.T) {
	cs := newTestCache(t)
	path, err := cs.Path(10)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if len(path) == 0 {
		t.Error("expected non-empty path")
	}
	// Path should be inside the configured cache dir.
	if !strings.HasPrefix(path, cs.CacheDir) {
		t.Errorf("path %q is not inside CacheDir %q", path, cs.CacheDir)
	}
}

// ---------- SaveProjects / Projects round-trip ----------

func TestSaveAndGetProjects(t *testing.T) {
	cs := newTestCache(t)
	projects := []Project{
		{ID: 1, Name: "Alpha"},
		{ID: 2, Name: "Beta"},
	}

	if err := cs.SaveProjects(10, projects); err != nil {
		t.Fatalf("SaveProjects: %v", err)
	}

	got, err := cs.Projects(10)
	if err != nil {
		t.Fatalf("Projects: %v", err)
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
	if err := cs.SaveProjects(1, []Project{}); err != nil {
		t.Fatalf("SaveProjects: %v", err)
	}
	got, err := cs.Projects(1)
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}

func TestSaveProjects_OverwritesPreviousCache(t *testing.T) {
	cs := newTestCache(t)

	original := []Project{{ID: 1, Name: "Original"}}
	if err := cs.SaveProjects(5, original); err != nil {
		t.Fatalf("SaveProjects (original): %v", err)
	}

	updated := []Project{{ID: 2, Name: "Updated"}}
	if err := cs.SaveProjects(5, updated); err != nil {
		t.Fatalf("SaveProjects (updated): %v", err)
	}

	got, err := cs.Projects(5)
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Updated" {
		t.Errorf("expected updated cache, got %+v", got)
	}
}

func TestSaveProjects_IsolatedByWorkspace(t *testing.T) {
	cs := newTestCache(t)
	p1 := []Project{{ID: 1, Name: "WS-1"}}
	p2 := []Project{{ID: 2, Name: "WS-2"}}

	if err := cs.SaveProjects(100, p1); err != nil {
		t.Fatalf("SaveProjects ws 100: %v", err)
	}
	if err := cs.SaveProjects(200, p2); err != nil {
		t.Fatalf("SaveProjects ws 200: %v", err)
	}

	got1, _ := cs.Projects(100)
	got2, _ := cs.Projects(200)

	if len(got1) != 1 || got1[0].Name != "WS-1" {
		t.Errorf("ws 100: expected WS-1, got %+v", got1)
	}
	if len(got2) != 1 || got2[0].Name != "WS-2" {
		t.Errorf("ws 200: expected WS-2, got %+v", got2)
	}
}

// ---------- Projects: cache miss ----------

func TestProjects_CacheMiss(t *testing.T) {
	cs := newTestCache(t)
	if _, err := cs.Projects(999); err == nil {
		t.Error("expected error for cache miss (file does not exist)")
	}
}

// ---------- Projects: cache expiry ----------

func TestProjects_CacheExpired(t *testing.T) {
	cs := newTestCache(t)
	projects := []Project{{ID: 1, Name: "Stale"}}

	cacheFile, err := cs.Path(42)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	expired := cachedProjects{
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

	if _, err := cs.Projects(42); err == nil {
		t.Error("expected error for expired cache (>24h old)")
	}
}

func TestProjects_CacheJustUnderTTL(t *testing.T) {
	cs := newTestCache(t)
	projects := []Project{{ID: 1, Name: "Fresh"}}

	cacheFile, err := cs.Path(7)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	// Just under 24 hours — should still be valid.
	fresh := cachedProjects{
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

	got, err := cs.Projects(7)
	if err != nil {
		t.Fatalf("Projects: %v (expected valid cache)", err)
	}
	if len(got) != 1 || got[0].Name != "Fresh" {
		t.Errorf("unexpected projects: %+v", got)
	}
}

func TestProjects_CacheExactlyAtTTL(t *testing.T) {
	cs := newTestCache(t)
	projects := []Project{{ID: 1, Name: "Boundary"}}

	cacheFile, err := cs.Path(8)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	// Exactly 24 hours ago — should be considered expired (>= 24h).
	boundary := cachedProjects{
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
	if _, err := cs.Projects(8); err == nil {
		t.Error("expected error: cache at exactly 24h should be considered expired")
	}
}

// ---------- Corrupted cache file ----------

func TestProjects_CorruptedCacheFile(t *testing.T) {
	cs := newTestCache(t)

	cacheFile, err := cs.Path(55)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("not valid json {{{{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := cs.Projects(55); err == nil {
		t.Error("expected error for corrupted cache file")
	}
}
