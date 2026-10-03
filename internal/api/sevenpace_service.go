package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// CreateWorkLog posts a single worklog to 7pace Timetracker.
func (c *SevenPaceClient) CreateWorkLog(ctx context.Context, workLog SevenPaceWorkLog) (*SevenPaceWorkLog, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/workLogs?api-version=3.0", workLog)
	if err != nil {
		return nil, err
	}

	var created SevenPaceWorkLog
	if reqErr := c.doRequest(req, &created); reqErr != nil {
		return nil, reqErr
	}

	return &created, nil
}

func (c *SevenPaceClient) newRequest(ctx context.Context, method, endpoint string, body any) (*http.Request, error) {
	req, err := newJSONRequest(ctx, method, c.BaseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	c.setAuth(req)

	return req, nil
}

// doRequest sends req, adding a hint about the configured credentials when
// the server rejects them.
func (c *SevenPaceClient) doRequest(req *http.Request, result any) error {
	err := doJSON(c.HTTPClient, req, result)

	var statusErr *statusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w (server auth schemes: %q)\n"+
			"check sevenpace.domain/username/password in your config; the Windows credentials were rejected",
			err, statusErr.Header.Get("WWW-Authenticate"))
	}

	return err
}

// setAuth sets the Basic-auth credentials that the NTLM negotiator uses for
// the handshake. The username is qualified with the domain (DOMAIN\user) when
// a domain is configured.
func (c *SevenPaceClient) setAuth(req *http.Request) {
	user := c.Username
	if c.Domain != "" {
		user = c.Domain + "\\" + c.Username
	}

	req.SetBasicAuth(user, c.Password)
}
