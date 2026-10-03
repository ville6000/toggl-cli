package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/ville6000/toggl-cli/internal/cache"
	"github.com/ville6000/toggl-cli/internal/config"
)

// DefaultBaseURL is the public Toggl API endpoint used unless overridden.
const DefaultBaseURL = "https://api.track.toggl.com/api/v9"

// Client talks to the Toggl Track v9 API, authenticating with an API token.
// Projects are cached on disk through Cache when it is set.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
	Cache      *cache.Service
}

// ClientOption customises a Client built by NewClient.
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
func WithCache(cacheService *cache.Service) ClientOption {
	return func(c *Client) {
		c.Cache = cacheService
	}
}

// NewClient builds a Toggl API client. The project cache is best-effort: if
// the cache directory is unavailable the client still works, it just refetches
// projects on every call.
func NewClient(authToken string, opts ...ClientOption) *Client {
	// A cache failure is not fatal: the client still works, it just refetches
	// projects instead of reading them from disk.
	cacheService, _ := cache.NewService()

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

// NewClientFromConfig builds the client used by commands, honouring the
// optional toggl.base_url config override.
func NewClientFromConfig(authToken string) *Client {
	var opts []ClientOption
	if baseURL := config.TogglBaseURL(); baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}

	return NewClient(authToken, opts...)
}
