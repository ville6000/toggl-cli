package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/cache"
)

func TestNewAPIClient_ReturnsNonNil(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("test-token")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
}

func TestNewAPIClient_SetsAuthToken(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("my-token")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.AuthToken != "my-token" {
		t.Errorf("AuthToken: got %q, want %q", client.AuthToken, "my-token")
	}
}

func TestNewAPIClient_SetsBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.BaseURL == "" {
		t.Error("BaseURL should not be empty")
	}
}

func TestNewAPIClient_SetsHTTPClient(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestNewAPIClient_SetsCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.Cache == nil {
		t.Error("Cache should not be nil")
	}
}

func TestNewAPIClient_WithBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("tok", WithBaseURL("http://127.0.0.1:8080/api/"))
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	// The trailing slash is trimmed so endpoints do not end up with "//".
	if want := "http://127.0.0.1:8080/api"; client.BaseURL != want {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, want)
	}
}

func TestNewAPIClient_DefaultsToTheTogglAPI(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, DefaultBaseURL)
	}
}

func TestNewAPIClient_WithHTTPClientAndCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	httpClient := &http.Client{Timeout: time.Second}
	cacheService := &cache.CacheService{CacheDir: t.TempDir()}

	client := NewAPIClient("tok", WithHTTPClient(httpClient), WithCache(cacheService))
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.HTTPClient != httpClient {
		t.Error("WithHTTPClient did not replace the HTTP client")
	}
	if client.Cache != cacheService {
		t.Error("WithCache did not replace the cache")
	}
}

func TestNewAPIClientFromConfig_UsesTheConfiguredBaseURL(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("toggl.base_url", "http://stub.invalid")

	client := NewAPIClientFromConfig("tok")
	if client == nil {
		t.Fatal("NewAPIClientFromConfig returned nil")
	}
	if want := "http://stub.invalid"; client.BaseURL != want {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, want)
	}
}

func TestNewAPIClientFromConfig_DefaultsWithoutOverride(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	client := NewAPIClientFromConfig("tok")
	if client == nil {
		t.Fatal("NewAPIClientFromConfig returned nil")
	}
	if client.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL: got %q, want %q", client.BaseURL, DefaultBaseURL)
	}
}
