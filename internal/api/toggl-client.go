package api

import (
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
}

func NewAPIClient(authToken string) *Client {
	return &Client{
		BaseURL: "https://api.track.toggl.com/api/v9",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		AuthToken: authToken,
	}
}
