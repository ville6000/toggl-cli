package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestNewClient_ReturnsNonNil(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("test-token")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_SetsAuthToken(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("my-token")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.AuthToken != "my-token" {
		t.Errorf("AuthToken: got %q, want %q", client.AuthToken, "my-token")
	}
}

func TestNewClient_SetsBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("tok")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.BaseURL == "" {
		t.Error("BaseURL should not be empty")
	}
}

func TestNewClient_SetsHTTPClient(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("tok")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestNewClient_SetsCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("tok")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.Cache == nil {
		t.Error("Cache should not be nil")
	}
}

func TestNewClient_WithBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("tok", WithBaseURL("http://127.0.0.1:8080/api/"))
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	// The trailing slash is trimmed so endpoints do not end up with "//".
	if want := "http://127.0.0.1:8080/api"; client.BaseURL != want {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, want)
	}
}

func TestNewClient_DefaultsToTheTogglAPI(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient("tok")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, DefaultBaseURL)
	}
}

func TestNewClient_WithHTTPClientAndCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	httpClient := &http.Client{Timeout: time.Second}
	projectCache := &ProjectCache{CacheDir: t.TempDir()}

	client := NewClient("tok", WithHTTPClient(httpClient), WithCache(projectCache))
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.HTTPClient != httpClient {
		t.Error("WithHTTPClient did not replace the HTTP client")
	}
	if client.Cache != projectCache {
		t.Error("WithCache did not replace the cache")
	}
}

func TestNewClientFromConfig_UsesTheConfiguredBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("toggl.base_url", "http://stub.invalid")

	client := NewClientFromConfig("tok")
	if client == nil {
		t.Fatal("NewClientFromConfig returned nil")
	}
	if want := "http://stub.invalid"; client.BaseURL != want {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, want)
	}
}

func TestNewClientFromConfig_DefaultsWithoutOverride(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	client := NewClientFromConfig("tok")
	if client == nil {
		t.Fatal("NewClientFromConfig returned nil")
	}
	if client.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, DefaultBaseURL)
	}
}
