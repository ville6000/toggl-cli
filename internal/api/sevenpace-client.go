package api

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Azure/go-ntlmssp"
	"github.com/ville6000/toggl-cli/internal/utils"
)

// SevenPaceClient talks to an on-prem 7pace Timetracker REST API using NTLM
// (Windows/Negotiate) authentication.
type SevenPaceClient struct {
	BaseURL        string
	HTTPClient     *http.Client
	Domain         string
	Username       string
	Password       string
	ActivityTypeID string
}

// NewSevenPaceClient builds a client from the given 7pace configuration. The
// HTTP transport is wrapped with an NTLM negotiator so that Basic-auth
// credentials set on each request are used to perform the NTLM handshake.
func NewSevenPaceClient(cfg utils.SevenPaceConfig) *SevenPaceClient {
	// Force HTTP/1.1: NTLM authenticates a TCP connection, which does not work
	// over HTTP/2's multiplexed connections. Setting TLSNextProto to a non-nil
	// empty map disables the automatic HTTP/2 upgrade.
	transport := &http.Transport{
		TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
	}
	if cfg.InsecureSkipTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 - opt-in for self-signed corp certs
	}

	return &SevenPaceClient{
		BaseURL: cfg.BaseURL,
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: ntlmssp.Negotiator{RoundTripper: transport},
		},
		Domain:         cfg.Domain,
		Username:       cfg.Username,
		Password:       cfg.Password,
		ActivityTypeID: cfg.ActivityTypeID,
	}
}
