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
