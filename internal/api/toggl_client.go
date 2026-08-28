package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/ville6000/toggl-cli/internal/cache"
	"github.com/ville6000/toggl-cli/internal/utils"
)

// DefaultBaseURL is the public Toggl API endpoint used unless overridden.
const DefaultBaseURL = "https://api.track.toggl.com/api/v9"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
	Cache      *cache.CacheService
}

// ClientOption customises a Client built by NewAPIClient.
type ClientOption func(*Client)

// WithBaseURL points the client at a different Toggl API host, which is how
// tests aim commands at a stub server.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.BaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient replaces the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.HTTPClient = httpClient
	}
}

// WithCache replaces the project cache.
func WithCache(cacheService *cache.CacheService) ClientOption {
	return func(c *Client) {
		c.Cache = cacheService
	}
}

// NewAPIClient builds a Toggl API client. The project cache is best-effort: if
// the cache directory is unavailable the client still works, it just refetches
// projects on every call.
func NewAPIClient(authToken string, opts ...ClientOption) *Client {
	// A cache failure is not fatal: the client still works, it just refetches
	// projects instead of reading them from disk.
	cacheService, _ := cache.NewCacheService()

	client := &Client{
		BaseURL: DefaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		AuthToken: authToken,
		Cache:     cacheService,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// NewAPIClientFromConfig builds the client used by commands, honouring the
// optional toggl.base_url config override.
func NewAPIClientFromConfig(authToken string) *Client {
	var opts []ClientOption
	if baseURL := utils.GetTogglBaseURL(); baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}

	return NewAPIClient(authToken, opts...)
}
