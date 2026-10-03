package cmd

import (
	"testing"

	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
)

func TestNewTogglClient_UsesTheConfiguredBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	v := viper.New()
	v.Set("toggl.base_url", "http://stub.invalid")

	client := newTogglClient(v, "tok")
	if want := "http://stub.invalid"; client.BaseURL != want {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, want)
	}
}

func TestNewTogglClient_DefaultsWithoutOverride(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	client := newTogglClient(viper.New(), "tok")
	if client.BaseURL != api.DefaultBaseURL {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, api.DefaultBaseURL)
	}
}
