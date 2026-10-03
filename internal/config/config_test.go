package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// ---------- Token ----------

func TestToken_Success(t *testing.T) {
	v := viper.New()
	v.Set("toggl.token", "my-secret-token")

	token, err := Token(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "my-secret-token" {
		t.Errorf("got %q, want %q", token, "my-secret-token")
	}
}

func TestToken_Missing(t *testing.T) {
	v := viper.New()

	_, err := Token(v)
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
	if !strings.Contains(err.Error(), "toggl.token") {
		t.Errorf("error message should mention toggl.token, got: %q", err.Error())
	}
}

func TestToken_EmptyString(t *testing.T) {
	v := viper.New()
	v.Set("toggl.token", "")

	if _, err := Token(v); err == nil {
		t.Error("expected error for empty token string")
	}
}

// ---------- TokenAndWorkspace ----------

func TestTokenAndWorkspace_BothPresent(t *testing.T) {
	v := viper.New()
	v.Set("toggl.token", "tok123")
	v.Set("toggl.workspace_id", 42)

	token, wsID, err := TokenAndWorkspace(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok123" {
		t.Errorf("token: got %q, want %q", token, "tok123")
	}
	if wsID != 42 {
		t.Errorf("workspace_id: got %d, want 42", wsID)
	}
}

func TestTokenAndWorkspace_MissingToken(t *testing.T) {
	v := viper.New()
	v.Set("toggl.workspace_id", 42)

	_, _, err := TokenAndWorkspace(v)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "toggl.token") {
		t.Errorf("error should mention toggl.token, got: %q", err.Error())
	}
}

func TestTokenAndWorkspace_MissingWorkspaceID(t *testing.T) {
	v := viper.New()
	v.Set("toggl.token", "tok123")

	_, _, err := TokenAndWorkspace(v)
	if err == nil {
		t.Fatal("expected error for missing workspace_id")
	}
	if !strings.Contains(err.Error(), "toggl.workspace_id") {
		t.Errorf("error should mention toggl.workspace_id, got: %q", err.Error())
	}
}

func TestTokenAndWorkspace_BothMissing(t *testing.T) {
	v := viper.New()

	if _, _, err := TokenAndWorkspace(v); err == nil {
		t.Error("expected error when both token and workspace_id are missing")
	}
}

func TestTokenAndWorkspace_EmptyToken(t *testing.T) {
	v := viper.New()
	v.Set("toggl.token", "")
	v.Set("toggl.workspace_id", 1)

	if _, _, err := TokenAndWorkspace(v); err == nil {
		t.Error("expected error for empty token string")
	}
}

func TestTokenAndWorkspace_ZeroWorkspaceID(t *testing.T) {
	v := viper.New()
	v.Set("toggl.token", "tok")
	v.Set("toggl.workspace_id", 0)

	if _, _, err := TokenAndWorkspace(v); err == nil {
		t.Error("expected error for zero workspace_id")
	}
}

// ---------- Timezone ----------

func TestTimezone_Unset(t *testing.T) {
	v := viper.New()

	loc, err := Timezone(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc != time.Local {
		t.Errorf("expected time.Local, got %v", loc)
	}
}

func TestTimezone_Valid(t *testing.T) {
	v := viper.New()
	v.Set("toggl.timezone", "America/New_York")

	loc, err := Timezone(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("unexpected error from time.LoadLocation: %v", err)
	}
	if loc.String() != want.String() {
		t.Errorf("got %s, want %s", loc.String(), want.String())
	}
}

func TestTimezone_Invalid(t *testing.T) {
	v := viper.New()
	v.Set("toggl.timezone", "Not/A/Timezone")

	_, err := Timezone(v)
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
	if !strings.Contains(err.Error(), "invalid timezone") {
		t.Errorf("error should mention 'invalid timezone', got: %q", err.Error())
	}
}

// ---------- environment ----------

func TestEnvVar(t *testing.T) {
	tests := map[string]string{
		"toggl.token":          "TOGGL_CLI_TOGGL_TOKEN",
		"toggl.workspace_id":   "TOGGL_CLI_TOGGL_WORKSPACE_ID",
		"start.ticket_pattern": "TOGGL_CLI_START_TICKET_PATTERN",
	}
	for key, want := range tests {
		if got := EnvVar(key); got != want {
			t.Errorf("EnvVar(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestUseEnv_ReadsPrefixedVariables(t *testing.T) {
	t.Setenv("TOGGL_CLI_TOGGL_TOKEN", "env-token")
	t.Setenv("TOGGL_CLI_TOGGL_WORKSPACE_ID", "42")

	v := viper.New()
	UseEnv(v)

	token, workspaceID, err := TokenAndWorkspace(v)
	if err != nil {
		t.Fatalf("TokenAndWorkspace: %v", err)
	}
	if token != "env-token" || workspaceID != 42 {
		t.Errorf("got token %q, workspace %d; want env-token, 42", token, workspaceID)
	}
}

// Unprefixed names such as TOGGL_TOKEN are left to other tools.
func TestUseEnv_IgnoresUnprefixedVariables(t *testing.T) {
	t.Setenv("TOGGL_TOKEN", "other-tool")

	v := viper.New()
	UseEnv(v)

	if _, err := Token(v); err == nil {
		t.Error("expected TOGGL_TOKEN to be ignored")
	}
}

func TestMissingKeyErrorNamesTheEnvVar(t *testing.T) {
	_, err := Token(viper.New())
	if err == nil || !strings.Contains(err.Error(), "TOGGL_CLI_TOGGL_TOKEN") {
		t.Errorf("error should name TOGGL_CLI_TOGGL_TOKEN, got %v", err)
	}
}
