package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_MissingFileIsNotAnError(t *testing.T) {
	stub := newAPIStub(t)
	stub.respond(http.MethodGet, "/workspaces", http.StatusOK, []any{})
	v := setupCLITest(t, stub)

	missing := filepath.Join(t.TempDir(), "missing.yaml")
	_, stderr, err := executeCommand(t, v, "--config", missing, "workspaces")
	if err != nil {
		t.Fatalf("workspaces with a missing config file: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected no stderr output, got %q", stderr)
	}
}

func TestLoadConfig_ReportsAnUnparsableFile(t *testing.T) {
	v := setupCLITest(t, nil)

	bad := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(bad, []byte("toggl: [oops"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := executeCommand(t, v, "--config", bad, "workspaces")
	if err == nil || !strings.Contains(err.Error(), "failed to read config file") {
		t.Fatalf("expected a config read error, got %v", err)
	}
}
