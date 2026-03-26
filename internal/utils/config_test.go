package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// resetViper clears all viper state between tests.
func resetViper() {
	viper.Reset()
}

// ---------- GetToken ----------

func TestGetToken_Success(t *testing.T) {
	resetViper()
	viper.Set("toggl.token", "my-secret-token")

	token, err := GetToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "my-secret-token" {
		t.Errorf("got %q, want %q", token, "my-secret-token")
	}
}

func TestGetToken_Missing(t *testing.T) {
	resetViper()

	_, err := GetToken()
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
	if !strings.Contains(err.Error(), "toggl.token") {
		t.Errorf("error message should mention toggl.token, got: %q", err.Error())
	}
}

func TestGetToken_EmptyString(t *testing.T) {
	resetViper()
	viper.Set("toggl.token", "")

	if _, err := GetToken(); err == nil {
		t.Error("expected error for empty token string")
	}
}

// ---------- GetConfig ----------

func TestGetConfig_BothPresent(t *testing.T) {
	resetViper()
	viper.Set("toggl.token", "tok123")
	viper.Set("toggl.workspace_id", 42)

	token, wsID, err := GetConfig()
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

func TestGetConfig_MissingToken(t *testing.T) {
	resetViper()
	viper.Set("toggl.workspace_id", 42)

	_, _, err := GetConfig()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "toggl.token") {
		t.Errorf("error should mention toggl.token, got: %q", err.Error())
	}
}

func TestGetConfig_MissingWorkspaceID(t *testing.T) {
	resetViper()
	viper.Set("toggl.token", "tok123")

	_, _, err := GetConfig()
	if err == nil {
		t.Fatal("expected error for missing workspace_id")
	}
	if !strings.Contains(err.Error(), "toggl.workspace_id") {
		t.Errorf("error should mention toggl.workspace_id, got: %q", err.Error())
	}
}

func TestGetConfig_BothMissing(t *testing.T) {
	resetViper()

	if _, _, err := GetConfig(); err == nil {
		t.Error("expected error when both token and workspace_id are missing")
	}
}

func TestGetConfig_EmptyToken(t *testing.T) {
	resetViper()
	viper.Set("toggl.token", "")
	viper.Set("toggl.workspace_id", 1)

	if _, _, err := GetConfig(); err == nil {
		t.Error("expected error for empty token string")
	}
}

func TestGetConfig_ZeroWorkspaceID(t *testing.T) {
	resetViper()
	viper.Set("toggl.token", "tok")
	viper.Set("toggl.workspace_id", 0)

	if _, _, err := GetConfig(); err == nil {
		t.Error("expected error for zero workspace_id")
	}
}

// ---------- GetTimezone ----------

func TestGetTimezone_Unset(t *testing.T) {
	resetViper()

	loc, err := GetTimezone()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc != time.Local {
		t.Errorf("expected time.Local, got %v", loc)
	}
}

func TestGetTimezone_Valid(t *testing.T) {
	resetViper()
	viper.Set("toggl.timezone", "America/New_York")

	loc, err := GetTimezone()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := time.LoadLocation("America/New_York")
	if loc.String() != want.String() {
		t.Errorf("got %q, want %q", loc, want)
	}
}

func TestGetTimezone_Invalid(t *testing.T) {
	resetViper()
	viper.Set("toggl.timezone", "Not/A/Timezone")

	_, err := GetTimezone()
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
	if !strings.Contains(err.Error(), "invalid timezone") {
		t.Errorf("error should mention 'invalid timezone', got: %q", err.Error())
	}
}
