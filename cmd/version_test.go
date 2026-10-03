package cmd

import (
	"strings"
	"testing"
)

func TestVersionFlag_ReportsTheStampedVersion(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })
	version = "1.2.3"

	v := setupCLITest(t, nil)
	out, _, err := executeCommand(t, v, "--version")
	if err != nil {
		t.Fatalf("--version: %v", err)
	}
	if !strings.Contains(out, "toggl-cli version 1.2.3") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestBuildVersion_DefaultsToDevForLocalBuilds(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })
	version = ""

	// Test binaries are built from the working tree, which Go records as
	// "(devel)" rather than a module version.
	if got := buildVersion(); got != "dev" {
		t.Errorf("buildVersion() = %q, want %q", got, "dev")
	}
}
