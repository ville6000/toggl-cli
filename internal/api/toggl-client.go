package api

import (
	"net/http"
	"time"

	"github.com/ville6000/toggl-cli/internal/cache"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
	Cache      *cache.CacheService
}

func NewAPIClient(authToken string) *Client {
	cacheService, err := cache.NewCacheService()
	if err != nil {
		return nil
	}

	return &Client{
		BaseURL: "https://api.track.toggl.com/api/v9",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		AuthToken: authToken,
		Cache:     cacheService,
	}
}
